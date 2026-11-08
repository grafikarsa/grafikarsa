package users

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	cardHeader   = "Grafikarsa - SMKN 4 Malang"
	cardFooter   = "Masuk di grafikarsa.com/masuk lalu aktivasi akunmu. Jangan bagikan kartu ini."
	cardGapTitle = "Sandi awal"

	// A4 portrait fits five rows of two 85 x 50 mm cards with 20 mm side
	// margins and 23.5 mm top/bottom margins.
	cardWidth   = 85.0
	cardHeight  = 50.0
	cardColumns = 2
	cardRows    = 5
	cardMarginX = 20.0
	cardMarginY = 23.5

	cardsPerPage = cardColumns * cardRows
)

// AccountCard is the printable data of one not-yet-activated student account.
type AccountCard struct {
	DisplayName       string
	NISN              string
	InitialPassword   string
	ClassName         string
	MajorName         string
	AcademicYearLabel string
	ExpiresAt         time.Time
}

// cardLines returns the text lines rendered on one account card, in order:
// header, name, class and major, NISN, initial password, and expiry. The footer
// is fixed instructional text drawn separately.
func cardLines(card AccountCard) []string {
	classAndMajor := card.ClassName
	if card.MajorName != "" {
		classAndMajor += " - " + card.MajorName
	}
	if card.AcademicYearLabel != "" {
		classAndMajor += " (" + card.AcademicYearLabel + ")"
	}
	return []string{
		cardHeader,
		card.DisplayName,
		classAndMajor,
		"NISN: " + card.NISN,
		cardGapTitle + ": " + card.InitialPassword,
		"Berlaku sampai: " + formatCardDate(card.ExpiresAt),
	}
}

var indonesianMonths = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// formatCardDate renders a date in the Indonesian long form, for example
// "8 Desember 2026".
func formatCardDate(t time.Time) string {
	utc := t.UTC()
	return fmt.Sprintf("%d %s %d", utc.Day(), indonesianMonths[int(utc.Month())-1], utc.Year())
}

// AccountCardsDocument is a generated PDF of account cards plus the metadata
// the HTTP layer needs to name the download.
type AccountCardsDocument struct {
	PDF       []byte
	Count     int
	ClassSlug string
}

// AccountCardsPDF builds an A4 PDF with account cards for every student in the
// class who has not activated yet and still has a valid initial password. It
// requires user.reset_password for the class's major and records an audit entry
// that never contains a password. The number of cards is returned alongside the
// document.
func (s *Service) AccountCardsPDF(ctx context.Context, a Actor, classID uuid.UUID) ([]byte, int, error) {
	document, err := s.AccountCardsDocument(ctx, a, classID)
	if err != nil {
		return nil, 0, err
	}
	return document.PDF, document.Count, nil
}

// AccountCardsDocument is AccountCardsPDF plus the class slug for the download
// file name.
func (s *Service) AccountCardsDocument(ctx context.Context, a Actor, classID uuid.UUID) (AccountCardsDocument, error) {
	scope, ok := a.permissionScope(permissionReset)
	if !ok {
		return AccountCardsDocument{}, apperr.NotFound()
	}

	class, err := s.st.GetClassByID(ctx, classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountCardsDocument{}, apperr.NotFound()
		}
		return AccountCardsDocument{}, fmt.Errorf("users: get class %s: %w", classID, err)
	}
	if !scope.allows([]uuid.UUID{class.MajorID}) {
		return AccountCardsDocument{}, apperr.NotFound()
	}

	year, err := s.st.GetAcademicYearByID(ctx, class.AcademicYearID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountCardsDocument{}, apperr.NotFound()
		}
		return AccountCardsDocument{}, fmt.Errorf("users: get academic year %s: %w", class.AcademicYearID, err)
	}
	major, err := s.st.GetMajorByID(ctx, class.MajorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountCardsDocument{}, apperr.NotFound()
		}
		return AccountCardsDocument{}, fmt.Errorf("users: get major %s: %w", class.MajorID, err)
	}

	rows, err := s.st.ListVaultEntriesForClass(ctx, classID)
	if err != nil {
		return AccountCardsDocument{}, fmt.Errorf("users: list vault entries for class %s: %w", classID, err)
	}
	if len(rows) == 0 {
		return AccountCardsDocument{}, apperr.Validation(apperr.Field(
			"classId", "empty",
			"Tidak ada siswa yang belum aktivasi dengan sandi awal yang masih berlaku.",
		))
	}

	cards := make([]AccountCard, 0, len(rows))
	for _, row := range rows {
		plaintext, err := s.openVaultEntry(row)
		if err != nil {
			return AccountCardsDocument{}, err
		}
		cards = append(cards, AccountCard{
			DisplayName:       row.DisplayName,
			NISN:              derefOptional(row.Nisn),
			InitialPassword:   plaintext,
			ClassName:         class.Name,
			MajorName:         major.Name,
			AcademicYearLabel: academicYearLabel(year.StartYear),
			ExpiresAt:         row.ExpiresAt.UTC(),
		})
	}

	pdfBytes, err := buildAccountCardsPDF(cards)
	if err != nil {
		return AccountCardsDocument{}, err
	}

	majorID := class.MajorID
	if err := audit.Record(ctx, s.st.Queries, audit.Entry{
		ActorID:      actorID(a),
		ActorIP:      a.IP,
		Action:       "user.export_account_cards",
		TargetType:   "class",
		TargetID:     &classID,
		ScopeMajorID: &majorID,
		Data: map[string]any{
			"class_id": classID,
			"count":    len(cards),
		},
	}); err != nil {
		return AccountCardsDocument{}, fmt.Errorf("users: record audit user.export_account_cards: %w", err)
	}

	return AccountCardsDocument{PDF: pdfBytes, Count: len(cards), ClassSlug: class.Slug}, nil
}

// openVaultEntry decrypts one stored initial password. The user id is the
// associated data, so an entry cannot be moved to another user.
func (s *Service) openVaultEntry(row store.ListVaultEntriesForClassRow) (string, error) {
	if s.box == nil {
		return "", fmt.Errorf("users: initial password vault is not configured")
	}
	plaintext, err := s.box.Open(row.Ciphertext, row.UserID[:])
	if err != nil {
		return "", fmt.Errorf("users: open vault entry for %s: %w", row.UserID, err)
	}
	return string(plaintext), nil
}

// buildAccountCardsPDF lays out A4 pages with 2 x 5 cards and dashed cut lines.
func buildAccountCardsPDF(cards []AccountCard) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	translate := pdf.UnicodeTranslatorFromDescriptor("")

	for i, card := range cards {
		if i > 0 && i%cardsPerPage == 0 {
			pdf.AddPage()
		}
		slot := i % cardsPerPage
		x := cardMarginX + float64(slot%cardColumns)*cardWidth
		y := cardMarginY + float64(slot/cardColumns)*cardHeight
		drawAccountCard(pdf, translate, card, x, y)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("users: render account cards pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func drawAccountCard(pdf *fpdf.Fpdf, translate func(string) string, card AccountCard, x, y float64) {
	lines := cardLines(card)
	innerX := x + 4
	innerWidth := cardWidth - 8

	// Dashed outline marks the cut line of each card.
	pdf.SetDrawColor(150, 150, 150)
	pdf.SetLineWidth(0.2)
	pdf.SetDashPattern([]float64{1.2, 1.2}, 0)
	pdf.Rect(x, y, cardWidth, cardHeight, "D")
	pdf.SetDashPattern(nil, 0)

	pdf.SetTextColor(90, 90, 90)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetXY(innerX, y+3)
	pdf.CellFormat(innerWidth, 4, translate(lines[0]), "", 0, "L", false, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(innerX, y+9)
	pdf.CellFormat(innerWidth, 5, fitText(pdf, translate(lines[1]), innerWidth), "", 0, "L", false, 0, "")

	pdf.SetTextColor(80, 80, 80)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(innerX, y+15)
	pdf.CellFormat(innerWidth, 4, fitText(pdf, translate(lines[2]), innerWidth), "", 0, "L", false, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(innerX, y+21)
	pdf.CellFormat(innerWidth, 4, translate(lines[3]), "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(innerX, y+26)
	pdf.CellFormat(innerWidth, 4, translate(cardGapTitle), "", 0, "L", false, 0, "")

	pdf.SetFont("Courier", "B", 14)
	pdf.SetXY(innerX, y+30)
	pdf.CellFormat(innerWidth, 6, translate(card.InitialPassword), "", 0, "L", false, 0, "")

	pdf.SetTextColor(90, 90, 90)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetXY(innerX, y+37)
	pdf.CellFormat(innerWidth, 4, translate(lines[5]), "", 0, "L", false, 0, "")

	pdf.SetTextColor(110, 110, 110)
	pdf.SetFont("Helvetica", "", 6)
	pdf.SetXY(innerX, y+43)
	pdf.MultiCell(innerWidth, 2.6, translate(cardFooter), "", "L", false)
}

// fitText shortens text with an ellipsis until it fits maxWidth in the current
// font. It never returns an empty string for non-empty input.
func fitText(pdf *fpdf.Fpdf, text string, maxWidth float64) string {
	if pdf.GetStringWidth(text) <= maxWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		trimmed := string(runes) + "..."
		if pdf.GetStringWidth(trimmed) <= maxWidth {
			return trimmed
		}
	}
	return string(runes)
}

func derefOptional(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func actorID(a Actor) *uuid.UUID {
	if a.Principal == nil {
		return nil
	}
	id := a.Principal.UserID
	return &id
}
