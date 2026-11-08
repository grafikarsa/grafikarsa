package profiles

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateExperience_RequiresEndMonthWhenNotCurrent(t *testing.T) {
	_, err := validateExperience(ExperienceInput{
		Kind:         "job",
		Title:        "Desainer",
		Organization: "Studio",
		StartMonth:   "2024-01",
		IsCurrent:    false,
	})
	if err == nil {
		t.Fatal("validateExperience() = nil, want validation error")
	}
}

func TestValidateExperience_RejectsEndMonthWhenCurrent(t *testing.T) {
	end := "2024-12"
	_, err := validateExperience(ExperienceInput{
		Kind:         "job",
		Title:        "Desainer",
		Organization: "Studio",
		StartMonth:   "2024-01",
		EndMonth:     &end,
		IsCurrent:    true,
	})
	if err == nil {
		t.Fatal("validateExperience() = nil, want validation error")
	}
}

func TestValidateExperience_RejectsEndBeforeStart(t *testing.T) {
	end := "2023-12"
	_, err := validateExperience(ExperienceInput{
		Kind:         "job",
		Title:        "Desainer",
		Organization: "Studio",
		StartMonth:   "2024-01",
		EndMonth:     &end,
	})
	if err == nil {
		t.Fatal("validateExperience() = nil, want validation error")
	}
}

func TestValidateExperience_NormalizesValidInput(t *testing.T) {
	end := "2024-12"
	description := "  Membuat desain  "
	credential := "https://example.com/sertifikat"
	valid, err := validateExperience(ExperienceInput{
		Kind:           " job ",
		Title:          "  Desainer  ",
		Organization:   "  Studio  ",
		Location:       ptr("  Malang  "),
		EmploymentType: ptr("full_time"),
		StartMonth:     "2024-01",
		EndMonth:       &end,
		Description:    &description,
		CredentialURL:  &credential,
	})
	if err != nil {
		t.Fatalf("validateExperience() error = %v", err)
	}
	if valid.Kind != "job" || valid.Title != "Desainer" || valid.Organization != "Studio" {
		t.Fatalf("unexpected normalized values: %+v", valid)
	}
	if valid.Location == nil || *valid.Location != "Malang" {
		t.Fatalf("location = %v, want Malang", valid.Location)
	}
	if !valid.Start.Valid || valid.Start.Time.Format("2006-01") != "2024-01" {
		t.Fatalf("start = %v, want 2024-01", valid.Start)
	}
	if !valid.End.Valid || valid.End.Time.Format("2006-01") != "2024-12" {
		t.Fatalf("end = %v, want 2024-12", valid.End)
	}
}

func TestValidateExperience_RejectsNonHTTPS(t *testing.T) {
	credential := "http://example.com"
	_, err := validateExperience(ExperienceInput{
		Kind:          "certification",
		Title:         "Sertifikat",
		Organization:  "LSP",
		StartMonth:    "2024-01",
		IsCurrent:     true,
		CredentialURL: &credential,
	})
	if err == nil {
		t.Fatal("validateExperience() = nil, want validation error")
	}
}

func TestValidateTeachingSubjects_TrimsAndRejectsLongSubject(t *testing.T) {
	subjects, err := validateTeachingSubjects([]string{"  Matematika  ", "Bahasa Indonesia"})
	if err != nil {
		t.Fatalf("validateTeachingSubjects() error = %v", err)
	}
	if len(subjects) != 2 || subjects[0] != "Matematika" {
		t.Fatalf("subjects = %v", subjects)
	}

	long := make([]byte, 61)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := validateTeachingSubjects([]string{string(long)}); err == nil {
		t.Fatal("validateTeachingSubjects() = nil, want validation error")
	}
}

func TestOptionalText_EmptyBecomesNil(t *testing.T) {
	got, err := optionalText(ptr("   "), 10, "field", "pesan")
	if err != nil {
		t.Fatalf("optionalText() error = %v", err)
	}
	if got != nil {
		t.Fatalf("optionalText() = %v, want nil", *got)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	createdAt := time.Now().UTC().Truncate(time.Millisecond)
	id := uuid.MustParse("0192f000-0000-7000-8000-000000000000")

	timeValue, idValue, err := decodeCursor(encodeCursor(createdAt, id))
	if err != nil {
		t.Fatalf("decodeCursor() error = %v", err)
	}
	if !timeValue.Equal(createdAt) || *idValue != id {
		t.Fatalf("decodeCursor() = %v, %v; want %v, %v", timeValue, idValue, createdAt, id)
	}
}

func TestDecodeCursor_EmptyMeansNoCursor(t *testing.T) {
	timeValue, idValue, err := decodeCursor("")
	if err != nil || timeValue != nil || idValue != nil {
		t.Fatalf("decodeCursor(\"\") = %v, %v, %v", timeValue, idValue, err)
	}
}

func ptr(value string) *string { return &value }
