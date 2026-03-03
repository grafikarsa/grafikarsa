package dto

// StudentImportRow represents a single row from import file
type StudentImportRow struct {
	Row         int    `json:"row"`
	Tingkat     int    `json:"tingkat"`
	KodeJurusan string `json:"kode_jurusan"`
	Rombel      string `json:"rombel"`
	Nama        string `json:"nama"`
	NIS         string `json:"nis"`
}

// StudentImportError represents an error for a specific row
type StudentImportError struct {
	Row   int    `json:"row"`
	NIS   string `json:"nis,omitempty"`
	Nama  string `json:"nama,omitempty"`
	Error string `json:"error"`
}

// ClassToCreate represents a class that will be created
type ClassToCreate struct {
	Nama    string `json:"nama"`
	Tingkat int    `json:"tingkat"`
	Jurusan string `json:"jurusan"`
	Rombel  string `json:"rombel"`
}

// StudentImportDryRunResponse is the response for dry run
type StudentImportDryRunResponse struct {
	TotalRows        int                  `json:"total_rows"`
	ClassesToCreate  []ClassToCreate      `json:"classes_to_create"`
	StudentsToCreate int                  `json:"students_to_create"`
	ValidationErrors []StudentImportError `json:"validation_errors"`
}

// StudentImportResponse is the response for actual import
type StudentImportResponse struct {
	TotalRows       int                  `json:"total_rows"`
	CreatedClasses  int                  `json:"created_classes"`
	CreatedStudents int                  `json:"created_students"`
	Skipped         int                  `json:"skipped"`
	Errors          []StudentImportError `json:"errors"`
}
