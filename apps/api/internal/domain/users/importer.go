package users

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	permissionImport = "academic.import"

	importKindStudents = "students"
	importKindTeachers = "teachers"
	importKindStaff    = "staff"

	importStatusValidating = "validating"
	importStatusReady      = "ready"
	importStatusInvalid    = "invalid"
	importStatusCommitting = "committing"
	importStatusCommitted  = "committed"
	importStatusFailed     = "failed"
	importStatusCancelled  = "cancelled"

	maxImportRows    = 5000
	maxImportErrors  = 500
	importBatchSize  = 100
	maxImportList    = 20
	targetTypeImport = "import_job"
)

// ImportRowError is one validation error on a CSV row. Row 0 means the whole
// file is invalid.
type ImportRowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ImportSkippedRow is one row skipped while the import ran.
type ImportSkippedRow struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportJob is one CSV import and its validation or execution result.
type ImportJob struct {
	ID              uuid.UUID
	Kind            string
	Status          string
	TotalRows       *int
	ValidRows       *int
	ErrorCountTotal int
	Errors          []ImportRowError
	ByClass         map[string]int
	Created         *int
	Skipped         []ImportSkippedRow
	CreatedAt       time.Time
	CommittedAt     *time.Time
	CreatedBy       uuid.UUID
}

// ImportJobInserter inserts River jobs inside a caller-owned transaction. The
// concrete production implementation is *river.Client[pgx.Tx].
type ImportJobInserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Importer implements CSV import of students, teachers, and staff.
type Importer struct {
	users *Service
	jobs  ImportJobInserter
}

// NewImporter builds an Importer on top of the user administration service.
// jobs may be nil for worker-only use; committing from the HTTP API requires it.
func NewImporter(users *Service, jobs ImportJobInserter) *Importer {
	return &Importer{users: users, jobs: jobs}
}

// storedImportRow is the normalized, validated row kept in summary.rows so the
// worker can create accounts without parsing the CSV again.
type storedImportRow struct {
	Row             int         `json:"row"`
	AccountType     string      `json:"accountType"`
	NISN            *string     `json:"nisn,omitempty"`
	StaffNumber     *string     `json:"staffNumber,omitempty"`
	DisplayName     string      `json:"displayName"`
	ClassID         *uuid.UUID  `json:"classId,omitempty"`
	ClassName       string      `json:"className,omitempty"`
	EntryYear       *int16      `json:"entryYear,omitempty"`
	TeacherMajorIDs []uuid.UUID `json:"teacherMajorIds,omitempty"`
}

type importSummary struct {
	Kind            string             `json:"kind"`
	Rows            []storedImportRow  `json:"rows"`
	ByClass         map[string]int     `json:"byClass"`
	ErrorCountTotal int                `json:"errorCountTotal"`
	Created         *int               `json:"created,omitempty"`
	Skipped         []ImportSkippedRow `json:"skipped,omitempty"`
	Failure         string             `json:"failure,omitempty"`
}

type importRawRow struct {
	Number int
	Values map[string]string
}

type importClass struct {
	ID        uuid.UUID
	Name      string
	MajorID   uuid.UUID
	StartYear int16
	Grade     int16
}

// Create parses and validates the uploaded CSV, then stores an import job with
// status ready or invalid. Validation never stops at the first bad row.
func (s *Importer) Create(ctx context.Context, a Actor, kind string, data []byte) (ImportJob, error) {
	scope, ok := a.permissionScope(permissionImport)
	if !ok {
		return ImportJob{}, apperr.NotFound()
	}

	kind = strings.ToLower(strings.TrimSpace(kind))
	if !validImportKind(kind) {
		return ImportJob{}, apperr.Validation(apperr.Field(
			"kind", "invalid", "Jenis impor tidak valid.",
		))
	}
	if kind != importKindStudents && !scope.global {
		return ImportJob{}, apperr.NotFound()
	}

	rawRows, parseErr := parseImportFile(kind, data)

	var (
		validRows []storedImportRow
		rowErrors []ImportRowError
		byClass   = map[string]int{}
		totalRows int
	)

	if parseErr != nil {
		rowErrors = []ImportRowError{*parseErr}
	} else {
		totalRows = len(rawRows)
		var validationErr error
		validRows, rowErrors, byClass, validationErr = s.validateImportRows(ctx, scope, kind, rawRows)
		if validationErr != nil {
			return ImportJob{}, validationErr
		}
		if len(rawRows) == 0 {
			rowErrors = append(rowErrors, ImportRowError{
				Row:     0,
				Field:   "file",
				Message: "File tidak memiliki baris data.",
			})
		}
	}

	status := importStatusReady
	if len(rowErrors) > 0 {
		status = importStatusInvalid
	}

	total := int32(totalRows)
	valid := int32(len(validRows))
	summary := importSummary{
		Kind:            kind,
		Rows:            validRows,
		ByClass:         byClass,
		ErrorCountTotal: len(rowErrors),
	}
	errorsJSON, err := json.Marshal(capImportErrors(rowErrors))
	if err != nil {
		return ImportJob{}, fmt.Errorf("users: encode import errors: %w", err)
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return ImportJob{}, fmt.Errorf("users: encode import summary: %w", err)
	}

	var job ImportJob
	err = s.users.st.WithTx(ctx, func(q *store.Queries) error {
		row, err := q.InsertImportJob(ctx, store.InsertImportJobParams{
			Kind:      kind,
			Status:    status,
			TotalRows: &total,
			ValidRows: &valid,
			Errors:    errorsJSON,
			Summary:   summaryJSON,
			CreatedBy: a.Principal.UserID,
		})
		if err != nil {
			return fmt.Errorf("users: insert import job: %w", err)
		}

		if err := audit.Record(ctx, q, audit.Entry{
			ActorID:    &a.Principal.UserID,
			ActorIP:    a.IP,
			Action:     "user.import_upload",
			TargetType: targetTypeImport,
			TargetID:   &row.ID,
			Data: map[string]any{
				"kind":       kind,
				"total_rows": totalRows,
				"valid_rows": len(validRows),
				"status":     status,
			},
		}); err != nil {
			return fmt.Errorf("users: record audit user.import_upload: %w", err)
		}

		job = importJobFromRecord(importJobRecord{
			ID:          row.ID,
			Kind:        row.Kind,
			Status:      row.Status,
			TotalRows:   row.TotalRows,
			ValidRows:   row.ValidRows,
			Errors:      row.Errors,
			Summary:     row.Summary,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   row.CreatedAt,
			CommittedAt: row.CommittedAt,
		})
		return nil
	})
	if err != nil {
		return ImportJob{}, err
	}
	return job, nil
}

// Get returns one import job visible to the actor.
func (s *Importer) Get(ctx context.Context, a Actor, id uuid.UUID) (ImportJob, error) {
	scope, ok := a.permissionScope(permissionImport)
	if !ok {
		return ImportJob{}, apperr.NotFound()
	}

	row, err := s.users.st.GetImportJob(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ImportJob{}, apperr.NotFound()
		}
		return ImportJob{}, fmt.Errorf("users: get import job %s: %w", id, err)
	}
	if !scope.global && row.CreatedBy != a.Principal.UserID {
		return ImportJob{}, apperr.NotFound()
	}
	return importJobFromRecord(importJobRecord{
		ID:          row.ID,
		Kind:        row.Kind,
		Status:      row.Status,
		TotalRows:   row.TotalRows,
		ValidRows:   row.ValidRows,
		Errors:      row.Errors,
		Summary:     row.Summary,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
		CommittedAt: row.CommittedAt,
	}), nil
}

// List returns the 20 newest import jobs visible to the actor.
func (s *Importer) List(ctx context.Context, a Actor) ([]ImportJob, error) {
	scope, ok := a.permissionScope(permissionImport)
	if !ok {
		return nil, apperr.NotFound()
	}

	var createdBy *uuid.UUID
	if !scope.global {
		userID := a.Principal.UserID
		createdBy = &userID
	}

	rows, err := s.users.st.ListImportJobs(ctx, store.ListImportJobsParams{
		CreatedBy:  createdBy,
		LimitCount: maxImportList,
	})
	if err != nil {
		return nil, fmt.Errorf("users: list import jobs: %w", err)
	}

	items := make([]ImportJob, 0, len(rows))
	for _, row := range rows {
		items = append(items, importJobFromRecord(importJobRecord{
			ID:          row.ID,
			Kind:        row.Kind,
			Status:      row.Status,
			TotalRows:   row.TotalRows,
			ValidRows:   row.ValidRows,
			Errors:      row.Errors,
			Summary:     row.Summary,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   row.CreatedAt,
			CommittedAt: row.CommittedAt,
		}))
	}
	return items, nil
}

// Commit moves a ready import to committing and enqueues the worker job in the
// same transaction.
func (s *Importer) Commit(ctx context.Context, a Actor, id uuid.UUID) (ImportJob, error) {
	scope, ok := a.permissionScope(permissionImport)
	if !ok {
		return ImportJob{}, apperr.NotFound()
	}
	if a.Principal == nil {
		return ImportJob{}, apperr.NotFound()
	}

	var job ImportJob
	err := s.users.st.WithTxFunc(ctx, func(q *store.Queries, tx pgx.Tx) error {
		row, err := q.GetImportJob(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("users: get import job %s: %w", id, err)
		}
		if !scope.global && row.CreatedBy != a.Principal.UserID {
			return apperr.NotFound()
		}
		if row.Status != importStatusReady {
			return apperr.InvalidTransition("Impor hanya bisa dijalankan saat status siap.")
		}

		affected, err := q.MarkImportJobCommitting(ctx, id)
		if err != nil {
			return fmt.Errorf("users: mark import job committing: %w", err)
		}
		if affected == 0 {
			return apperr.InvalidTransition("Impor hanya bisa dijalankan saat status siap.")
		}

		if s.jobs == nil {
			return fmt.Errorf("users: import job inserter is not configured")
		}
		if _, err := s.jobs.InsertTx(ctx, tx, ImportCommitArgs{
			ImportJobID: id,
			ActorID:     a.Principal.UserID,
			ActorIP:     a.IP,
		}, nil); err != nil {
			return fmt.Errorf("users: enqueue import commit: %w", err)
		}

		row.Status = importStatusCommitting
		job = importJobFromRecord(importJobRecord{
			ID:          row.ID,
			Kind:        row.Kind,
			Status:      row.Status,
			TotalRows:   row.TotalRows,
			ValidRows:   row.ValidRows,
			Errors:      row.Errors,
			Summary:     row.Summary,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   row.CreatedAt,
			CommittedAt: row.CommittedAt,
		})
		return nil
	})
	if err != nil {
		return ImportJob{}, err
	}
	return job, nil
}

// validateImportRows applies every per-row rule and resolves classes and
// majors. It returns only rows without errors.
func (s *Importer) validateImportRows(
	ctx context.Context,
	scope permissionScope,
	kind string,
	rawRows []importRawRow,
) ([]storedImportRow, []ImportRowError, map[string]int, error) {
	classes := map[string]importClass{}
	majors := map[string]uuid.UUID{}

	if kind == importKindStudents {
		resolved, err := s.importClasses(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		classes = resolved
	} else {
		resolved, err := s.importMajors(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		majors = resolved
	}

	existingNISN, existingNIP, err := s.existingIdentifiers(ctx, kind, rawRows)
	if err != nil {
		return nil, nil, nil, err
	}

	validRows := make([]storedImportRow, 0, len(rawRows))
	rowErrors := make([]ImportRowError, 0, len(rawRows))
	byClass := map[string]int{}
	seenNISN := map[string]int{}
	seenNIP := map[string]int{}

	for _, raw := range rawRows {
		var rowErrs []ImportRowError
		addError := func(field, message string) {
			rowErrs = append(rowErrs, ImportRowError{
				Row:     raw.Number,
				Field:   field,
				Message: message,
			})
		}

		displayName := strings.TrimSpace(raw.Values["nama"])
		if length := utf8.RuneCountInString(displayName); length < minDisplayNameLength || length > maxDisplayNameLength {
			addError("nama", "Nama harus 1–100 karakter.")
		}

		row := storedImportRow{
			Row:         raw.Number,
			DisplayName: displayName,
		}

		switch kind {
		case importKindStudents:
			row.AccountType = accountTypeStudent
			nisn := strings.TrimSpace(raw.Values["nisn"])
			switch {
			case nisn == "":
				addError("nisn", "NISN wajib diisi.")
			case !nisnPattern.MatchString(nisn):
				addError("nisn", "NISN harus 10 digit angka.")
			default:
				if first, exists := seenNISN[nisn]; exists {
					addError("nisn", fmt.Sprintf("NISN duplikat di dalam file (baris %d).", first))
				} else {
					seenNISN[nisn] = raw.Number
				}
				if existingNISN[nisn] {
					addError("nisn", "NISN sudah terdaftar.")
				}
				value := nisn
				row.NISN = &value
			}

			className := normalizeImportClassName(raw.Values["kelas"])
			class, found := classes[className]
			if !found {
				addError("kelas", "Kelas tidak ditemukan pada tahun ajaran berjalan.")
			} else if !scope.allows([]uuid.UUID{class.MajorID}) {
				addError("kelas", "Kamu tidak berwenang mengimpor ke kelas ini.")
			} else {
				classID := class.ID
				row.ClassID = &classID
				row.ClassName = class.Name
				entryYear := class.StartYear - (class.Grade - 10)
				row.EntryYear = &entryYear
			}

			if value := strings.TrimSpace(raw.Values["tahun_masuk"]); value != "" {
				year, err := strconv.Atoi(value)
				if err != nil {
					addError("tahun_masuk", "Tahun masuk harus berupa angka.")
				} else if year < 2000 || year > 2100 {
					addError("tahun_masuk", "Tahun masuk harus antara 2000 dan 2100.")
				} else {
					entryYear := int16(year)
					row.EntryYear = &entryYear
				}
			}

		case importKindTeachers:
			row.AccountType = accountTypeTeacher
			nip := strings.TrimSpace(raw.Values["nip"])
			if nip == "" {
				addError("nip", "NIP wajib diisi.")
			} else {
				violated := false
				if length := utf8.RuneCountInString(nip); length < minStaffNumberLength || length > maxStaffNumberLength {
					addError("nip", "NIP harus 3–30 karakter.")
					violated = true
				}
				if first, exists := seenNIP[nip]; exists {
					addError("nip", fmt.Sprintf("NIP duplikat di dalam file (baris %d).", first))
				} else {
					seenNIP[nip] = raw.Number
				}
				if existingNIP[nip] {
					addError("nip", "NIP sudah terdaftar.")
				}
				if !violated {
					value := nip
					row.StaffNumber = &value
				}
			}

			majorCodes, majorErrs := resolveImportMajors(raw.Values["jurusan"], majors)
			for _, message := range majorErrs {
				addError("jurusan", message)
			}
			row.TeacherMajorIDs = majorCodes

		case importKindStaff:
			row.AccountType = accountTypeStaff
			nip := strings.TrimSpace(raw.Values["nip"])
			if nip != "" {
				if length := utf8.RuneCountInString(nip); length < minStaffNumberLength || length > maxStaffNumberLength {
					addError("nip", "NIP harus 3–30 karakter.")
				} else {
					if first, exists := seenNIP[nip]; exists {
						addError("nip", fmt.Sprintf("NIP duplikat di dalam file (baris %d).", first))
					} else {
						seenNIP[nip] = raw.Number
					}
					if existingNIP[nip] {
						addError("nip", "NIP sudah terdaftar.")
					}
					value := nip
					row.StaffNumber = &value
				}
			}
		}

		if len(rowErrs) > 0 {
			rowErrors = append(rowErrors, rowErrs...)
			continue
		}

		validRows = append(validRows, row)
		if kind == importKindStudents {
			byClass[row.ClassName]++
		}
	}

	return validRows, rowErrors, byClass, nil
}

func (s *Importer) importClasses(ctx context.Context) (map[string]importClass, error) {
	classes := map[string]importClass{}

	year, err := s.users.st.GetCurrentAcademicYear(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("users: get current academic year: %w", err)
	}
	if err != nil {
		return classes, nil
	}

	rows, err := s.users.st.ListClasses(ctx, store.ListClassesParams{
		AcademicYearID: &year.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("users: list current classes: %w", err)
	}
	for _, row := range rows {
		classes[normalizeImportClassName(row.Name)] = importClass{
			ID:        row.ID,
			Name:      row.Name,
			MajorID:   row.MajorID,
			StartYear: year.StartYear,
			Grade:     row.Grade,
		}
	}
	return classes, nil
}

func (s *Importer) importMajors(ctx context.Context) (map[string]uuid.UUID, error) {
	rows, err := s.users.st.ListMajors(ctx)
	if err != nil {
		return nil, fmt.Errorf("users: list majors: %w", err)
	}
	majors := make(map[string]uuid.UUID, len(rows))
	for _, row := range rows {
		majors[strings.ToLower(strings.TrimSpace(row.Code))] = row.ID
	}
	return majors, nil
}

func (s *Importer) existingIdentifiers(
	ctx context.Context,
	kind string,
	rawRows []importRawRow,
) (map[string]bool, map[string]bool, error) {
	nisns := make([]string, 0, len(rawRows))
	nips := make([]string, 0, len(rawRows))
	seenNISN := map[string]bool{}
	seenNIP := map[string]bool{}

	for _, raw := range rawRows {
		switch kind {
		case importKindStudents:
			nisn := strings.TrimSpace(raw.Values["nisn"])
			if nisnPattern.MatchString(nisn) && !seenNISN[nisn] {
				seenNISN[nisn] = true
				nisns = append(nisns, nisn)
			}
		case importKindTeachers, importKindStaff:
			nip := strings.TrimSpace(raw.Values["nip"])
			if nip != "" && !seenNIP[nip] {
				seenNIP[nip] = true
				nips = append(nips, nip)
			}
		}
	}

	existingNISN := map[string]bool{}
	existingNIP := map[string]bool{}
	if len(nisns) == 0 && len(nips) == 0 {
		return existingNISN, existingNIP, nil
	}

	rows, err := s.users.st.ListExistingUserIdentifiers(ctx, store.ListExistingUserIdentifiersParams{
		Nisns:        nisns,
		StaffNumbers: nips,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("users: list existing identifiers: %w", err)
	}
	for _, row := range rows {
		if row.Nisn != nil {
			existingNISN[*row.Nisn] = true
		}
		if row.StaffNumber != nil {
			existingNIP[*row.StaffNumber] = true
		}
	}
	return existingNISN, existingNIP, nil
}

// parseImportFile reads UTF-8 CSV with an optional BOM, auto-detects comma or
// semicolon from the header line, and checks required headers.
func parseImportFile(kind string, data []byte) ([]importRawRow, *ImportRowError) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if len(data) == 0 {
		return nil, &ImportRowError{Row: 0, Field: "header", Message: "Berkas kosong."}
	}
	if !utf8.Valid(data) {
		return nil, &ImportRowError{Row: 0, Field: "file", Message: "Berkas harus berformat UTF-8."}
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = detectImportSeparator(data)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, &ImportRowError{Row: 0, Field: "file", Message: "Berkas CSV tidak valid."}
	}
	if len(records) == 0 {
		return nil, &ImportRowError{Row: 0, Field: "header", Message: "Berkas kosong."}
	}

	headers := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		name := normalizeImportHeader(header)
		if _, exists := headers[name]; !exists {
			headers[name] = index
		}
	}

	missing := missingImportHeaders(kind, headers)
	if len(missing) > 0 {
		return nil, &ImportRowError{
			Row:     0,
			Field:   "header",
			Message: "Kolom wajib tidak ada: " + strings.Join(missing, ", "),
		}
	}

	if len(records)-1 > maxImportRows {
		return nil, &ImportRowError{Row: 0, Field: "file", Message: "Maksimal 5000 baris data."}
	}

	rows := make([]importRawRow, 0, len(records)-1)
	for index, record := range records[1:] {
		values := make(map[string]string, len(headers))
		for name, column := range headers {
			if column < len(record) {
				values[name] = record[column]
			} else {
				values[name] = ""
			}
		}
		rows = append(rows, importRawRow{Number: index + 1, Values: values})
	}
	return rows, nil
}

func detectImportSeparator(data []byte) rune {
	line := data
	if index := bytes.IndexByte(data, '\n'); index >= 0 {
		line = data[:index]
	}
	if bytes.Count(line, []byte(";")) > bytes.Count(line, []byte(",")) {
		return ';'
	}
	return ','
}

func normalizeImportHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeImportClassName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func importedHeaderRequired(kind string) []string {
	switch kind {
	case importKindStudents:
		return []string{"nisn", "nama", "kelas"}
	case importKindTeachers:
		return []string{"nip", "nama"}
	case importKindStaff:
		return []string{"nama"}
	default:
		return nil
	}
}

func missingImportHeaders(kind string, headers map[string]int) []string {
	missing := make([]string, 0, 4)
	for _, name := range importedHeaderRequired(kind) {
		if _, exists := headers[name]; !exists {
			missing = append(missing, name)
		}
	}
	return missing
}

func resolveImportMajors(value string, majors map[string]uuid.UUID) ([]uuid.UUID, []string) {
	ids := make([]uuid.UUID, 0, 4)
	errors := make([]string, 0, 1)
	unknown := make([]string, 0, 4)
	seen := map[uuid.UUID]bool{}

	for _, part := range strings.Split(value, "|") {
		code := strings.ToLower(strings.TrimSpace(part))
		if code == "" {
			continue
		}
		id, found := majors[code]
		if !found {
			unknown = append(unknown, code)
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(unknown) > 0 {
		errors = append(errors, "Kode jurusan tidak ditemukan: "+strings.Join(unknown, ", ")+".")
	}
	return ids, errors
}

func validImportKind(kind string) bool {
	switch kind {
	case importKindStudents, importKindTeachers, importKindStaff:
		return true
	default:
		return false
	}
}

func capImportErrors(errors []ImportRowError) []ImportRowError {
	if len(errors) > maxImportErrors {
		return errors[:maxImportErrors]
	}
	return errors
}

type importJobRecord struct {
	ID          uuid.UUID
	Kind        string
	Status      string
	TotalRows   *int32
	ValidRows   *int32
	Errors      json.RawMessage
	Summary     []byte
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	CommittedAt *time.Time
}

func importJobFromRecord(record importJobRecord) ImportJob {
	job := ImportJob{
		ID:              record.ID,
		Kind:            record.Kind,
		Status:          record.Status,
		ErrorCountTotal: 0,
		Errors:          []ImportRowError{},
		ByClass:         map[string]int{},
		Skipped:         []ImportSkippedRow{},
		CreatedAt:       record.CreatedAt,
		CommittedAt:     record.CommittedAt,
		CreatedBy:       record.CreatedBy,
	}
	if record.TotalRows != nil {
		value := int(*record.TotalRows)
		job.TotalRows = &value
	}
	if record.ValidRows != nil {
		value := int(*record.ValidRows)
		job.ValidRows = &value
	}
	if len(record.Errors) > 0 {
		_ = json.Unmarshal(record.Errors, &job.Errors)
	}
	var summary importSummary
	if len(record.Summary) > 0 {
		_ = json.Unmarshal(record.Summary, &summary)
	}
	if summary.ByClass != nil {
		job.ByClass = summary.ByClass
	}
	if summary.Skipped != nil {
		job.Skipped = summary.Skipped
	}
	job.Created = summary.Created
	job.ErrorCountTotal = summary.ErrorCountTotal
	if job.ErrorCountTotal == 0 {
		job.ErrorCountTotal = len(job.Errors)
	}
	return job
}
