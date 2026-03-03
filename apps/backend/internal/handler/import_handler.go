package handler

import (
	"encoding/csv"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

type ImportHandler struct {
	adminRepo *repository.AdminRepository
	userRepo  *repository.UserRepository
}

func NewImportHandler(adminRepo *repository.AdminRepository, userRepo *repository.UserRepository) *ImportHandler {
	return &ImportHandler{
		adminRepo: adminRepo,
		userRepo:  userRepo,
	}
}

// StudentImportRow represents parsed row data
type studentImportRow struct {
	Row         int
	Tingkat     int
	KodeJurusan string
	Rombel      string
	Nama        string
	NIS         string
}

// ImportStudents handles student import from CSV/XLSX
func (h *ImportHandler) ImportStudents(c *fiber.Ctx) error {
	// Check dry_run parameter
	dryRun := c.FormValue("dry_run") == "true"

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_FILE", "File tidak ditemukan"))
	}

	// Check file size (max 5MB)
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("FILE_TOO_LARGE", "Ukuran file maksimal 5MB"))
	}

	// Detect file type
	filename := strings.ToLower(file.Filename)
	isCSV := strings.HasSuffix(filename, ".csv")
	isXLSX := strings.HasSuffix(filename, ".xlsx")

	if !isCSV && !isXLSX {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_FILE_TYPE", "File harus berformat CSV atau XLSX"))
	}

	// Open file
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuka file"))
	}
	defer f.Close()

	// Parse rows
	var rows []studentImportRow
	var parseErrors []dto.StudentImportError

	if isCSV {
		rows, parseErrors = h.parseCSV(f)
	} else {
		rows, parseErrors = h.parseXLSX(f)
	}

	if len(rows) == 0 && len(parseErrors) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("EMPTY_FILE", "File tidak memiliki data"))
	}

	// Get active tahun ajaran
	tahunAjaran, err := h.adminRepo.GetActiveTahunAjaran()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("NO_ACTIVE_TAHUN_AJARAN", "Tidak ada tahun ajaran aktif"))
	}

	// Validate and process rows
	validationErrors, classesToCreate, studentsToCreate := h.validateAndPrepare(rows, tahunAjaran.ID)

	// Combine parse errors and validation errors
	allErrors := append(parseErrors, validationErrors...)

	if dryRun {
		// Return dry run response
		return c.JSON(dto.SuccessResponse(dto.StudentImportDryRunResponse{
			TotalRows:        len(rows),
			ClassesToCreate:  classesToCreate,
			StudentsToCreate: len(studentsToCreate),
			ValidationErrors: allErrors,
		}, "Dry run selesai"))
	}

	// Actual import
	createdClasses := 0
	createdStudents := 0

	// Create classes first
	classMap := make(map[string]uuid.UUID) // key: "tingkat-jurusan-rombel" -> kelas_id

	for _, cls := range classesToCreate {
		// Find jurusan
		jurusan, err := h.adminRepo.FindJurusanByKode(cls.Jurusan)
		if err != nil {
			continue
		}

		// Check if kelas already exists
		existingKelas, err := h.adminRepo.FindKelasByTingkatJurusanRombel(tahunAjaran.ID, jurusan.ID, cls.Tingkat, cls.Rombel)
		if err == nil && existingKelas != nil {
			// Kelas already exists, use it
			key := h.kelasKey(cls.Tingkat, cls.Jurusan, cls.Rombel)
			classMap[key] = existingKelas.ID
			continue
		}

		// Create new kelas
		newKelas := &domain.Kelas{
			TahunAjaranID: tahunAjaran.ID,
			JurusanID:     jurusan.ID,
			Tingkat:       cls.Tingkat,
			Rombel:        strings.ToUpper(cls.Rombel),
		}

		if err := h.adminRepo.CreateKelas(newKelas); err == nil {
			key := h.kelasKey(cls.Tingkat, cls.Jurusan, cls.Rombel)
			classMap[key] = newKelas.ID
			createdClasses++
		}
	}

	// Also populate classMap with existing classes
	for _, student := range studentsToCreate {
		key := h.kelasKey(student.Tingkat, student.KodeJurusan, student.Rombel)
		if _, exists := classMap[key]; !exists {
			jurusan, err := h.adminRepo.FindJurusanByKode(student.KodeJurusan)
			if err != nil {
				continue
			}
			existingKelas, err := h.adminRepo.FindKelasByTingkatJurusanRombel(tahunAjaran.ID, jurusan.ID, student.Tingkat, student.Rombel)
			if err == nil && existingKelas != nil {
				classMap[key] = existingKelas.ID
			}
		}
	}

	// Create students
	var importErrors []dto.StudentImportError

	for _, student := range studentsToCreate {
		key := h.kelasKey(student.Tingkat, student.KodeJurusan, student.Rombel)
		kelasID, exists := classMap[key]
		if !exists {
			importErrors = append(importErrors, dto.StudentImportError{
				Row:   student.Row,
				NIS:   student.NIS,
				Nama:  student.Nama,
				Error: "Kelas tidak ditemukan",
			})
			continue
		}

		// Generate password hash
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(student.NIS), bcrypt.DefaultCost)
		if err != nil {
			importErrors = append(importErrors, dto.StudentImportError{
				Row:   student.Row,
				NIS:   student.NIS,
				Nama:  student.Nama,
				Error: "Gagal generate password",
			})
			continue
		}

		// Create user
		newUser := &domain.User{
			Username:     student.NIS,
			Email:        student.NIS + "@grafikarsa.com",
			PasswordHash: string(hashedPassword),
			Nama:         student.Nama,
			Role:         domain.RoleStudent,
			NIS:          &student.NIS,
			KelasID:      &kelasID,
			IsActive:     true,
		}

		if err := h.userRepo.Create(newUser); err != nil {
			importErrors = append(importErrors, dto.StudentImportError{
				Row:   student.Row,
				NIS:   student.NIS,
				Nama:  student.Nama,
				Error: "Gagal membuat user: " + err.Error(),
			})
			continue
		}

		createdStudents++
	}

	// Combine all errors
	allErrors = append(allErrors, importErrors...)

	return c.JSON(dto.SuccessResponse(dto.StudentImportResponse{
		TotalRows:       len(rows),
		CreatedClasses:  createdClasses,
		CreatedStudents: createdStudents,
		Skipped:         len(allErrors),
		Errors:          allErrors,
	}, "Import selesai"))
}

func (h *ImportHandler) parseCSV(r io.Reader) ([]studentImportRow, []dto.StudentImportError) {
	var rows []studentImportRow
	var errors []dto.StudentImportError

	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // Allow variable fields

	rowNum := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		rowNum++

		if err != nil {
			errors = append(errors, dto.StudentImportError{
				Row:   rowNum,
				Error: "Gagal membaca baris",
			})
			continue
		}

		// Skip header if detected
		if rowNum == 1 && (strings.ToLower(record[0]) == "tingkat" || strings.ToLower(record[0]) == "kelas") {
			continue
		}

		row, parseErr := h.parseRow(rowNum, record)
		if parseErr != nil {
			errors = append(errors, *parseErr)
			continue
		}

		rows = append(rows, *row)
	}

	return rows, errors
}

func (h *ImportHandler) parseXLSX(r io.Reader) ([]studentImportRow, []dto.StudentImportError) {
	var rows []studentImportRow
	var errors []dto.StudentImportError

	xlsx, err := excelize.OpenReader(r)
	if err != nil {
		errors = append(errors, dto.StudentImportError{
			Row:   0,
			Error: "Gagal membaca file XLSX",
		})
		return rows, errors
	}
	defer xlsx.Close()

	// Get first sheet
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		errors = append(errors, dto.StudentImportError{
			Row:   0,
			Error: "File XLSX tidak memiliki sheet",
		})
		return rows, errors
	}

	xlsxRows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		errors = append(errors, dto.StudentImportError{
			Row:   0,
			Error: "Gagal membaca sheet",
		})
		return rows, errors
	}

	for i, record := range xlsxRows {
		rowNum := i + 1

		// Skip header if detected
		if rowNum == 1 && len(record) > 0 && (strings.ToLower(record[0]) == "tingkat" || strings.ToLower(record[0]) == "kelas") {
			continue
		}

		row, parseErr := h.parseRow(rowNum, record)
		if parseErr != nil {
			errors = append(errors, *parseErr)
			continue
		}

		rows = append(rows, *row)
	}

	return rows, errors
}

func (h *ImportHandler) parseRow(rowNum int, record []string) (*studentImportRow, *dto.StudentImportError) {
	if len(record) < 5 {
		return nil, &dto.StudentImportError{
			Row:   rowNum,
			Error: "Kolom tidak lengkap (butuh 5 kolom)",
		}
	}

	// Parse tingkat
	tingkat, err := strconv.Atoi(strings.TrimSpace(record[0]))
	if err != nil || (tingkat != 10 && tingkat != 11 && tingkat != 12) {
		return nil, &dto.StudentImportError{
			Row:   rowNum,
			Error: "Tingkat harus 10, 11, atau 12",
		}
	}

	// Parse kode jurusan
	kodeJurusan := strings.TrimSpace(strings.ToLower(record[1]))
	if kodeJurusan == "" {
		return nil, &dto.StudentImportError{
			Row:   rowNum,
			Error: "Kode jurusan tidak boleh kosong",
		}
	}

	// Parse rombel
	rombel := strings.TrimSpace(strings.ToUpper(record[2]))
	if !regexp.MustCompile(`^[A-Z]$`).MatchString(rombel) {
		return nil, &dto.StudentImportError{
			Row:   rowNum,
			Error: "Rombel harus huruf A-Z",
		}
	}

	// Parse nama
	nama := strings.TrimSpace(record[3])
	if nama == "" {
		return nil, &dto.StudentImportError{
			Row:   rowNum,
			Error: "Nama tidak boleh kosong",
		}
	}

	// Parse NIS
	nis := strings.TrimSpace(record[4])
	if !regexp.MustCompile(`^\d+$`).MatchString(nis) {
		return nil, &dto.StudentImportError{
			Row:   rowNum,
			NIS:   nis,
			Error: "NIS harus berupa angka",
		}
	}

	return &studentImportRow{
		Row:         rowNum,
		Tingkat:     tingkat,
		KodeJurusan: kodeJurusan,
		Rombel:      rombel,
		Nama:        nama,
		NIS:         nis,
	}, nil
}

func (h *ImportHandler) validateAndPrepare(rows []studentImportRow, tahunAjaranID uuid.UUID) ([]dto.StudentImportError, []dto.ClassToCreate, []studentImportRow) {
	var errors []dto.StudentImportError
	var validStudents []studentImportRow

	// Collect all NIS and usernames to check
	var nisList []string
	var usernameList []string
	nisSet := make(map[string]int) // NIS -> first row number

	for _, row := range rows {
		// Check for duplicate NIS in file
		if firstRow, exists := nisSet[row.NIS]; exists {
			errors = append(errors, dto.StudentImportError{
				Row:   row.Row,
				NIS:   row.NIS,
				Nama:  row.Nama,
				Error: "NIS duplikat dengan baris " + strconv.Itoa(firstRow),
			})
			continue
		}
		nisSet[row.NIS] = row.Row
		nisList = append(nisList, row.NIS)
		usernameList = append(usernameList, row.NIS) // username = NIS
	}

	// Check existing NIS in database
	existingNIS, _ := h.adminRepo.FindExistingNIS(nisList)
	existingNISSet := make(map[string]bool)
	for _, nis := range existingNIS {
		existingNISSet[nis] = true
	}

	// Check existing usernames in database
	existingUsernames, _ := h.adminRepo.FindExistingUsernames(usernameList)
	existingUsernameSet := make(map[string]bool)
	for _, username := range existingUsernames {
		existingUsernameSet[username] = true
	}

	// Collect unique classes to create
	classSet := make(map[string]dto.ClassToCreate)

	// Validate each row
	for _, row := range rows {
		// Skip if already has error (duplicate in file)
		if _, exists := nisSet[row.NIS]; !exists {
			continue
		}

		// Check if NIS already exists in database
		if existingNISSet[row.NIS] {
			errors = append(errors, dto.StudentImportError{
				Row:   row.Row,
				NIS:   row.NIS,
				Nama:  row.Nama,
				Error: "NIS sudah terdaftar",
			})
			continue
		}

		// Check if username already exists in database
		if existingUsernameSet[row.NIS] {
			errors = append(errors, dto.StudentImportError{
				Row:   row.Row,
				NIS:   row.NIS,
				Nama:  row.Nama,
				Error: "Username sudah terdaftar",
			})
			continue
		}

		// Validate jurusan exists
		jurusan, err := h.adminRepo.FindJurusanByKode(row.KodeJurusan)
		if err != nil {
			errors = append(errors, dto.StudentImportError{
				Row:   row.Row,
				NIS:   row.NIS,
				Nama:  row.Nama,
				Error: "Kode jurusan '" + row.KodeJurusan + "' tidak ditemukan",
			})
			continue
		}

		// Check if kelas exists, if not add to classes to create
		_, err = h.adminRepo.FindKelasByTingkatJurusanRombel(tahunAjaranID, jurusan.ID, row.Tingkat, row.Rombel)
		if err != nil {
			// Kelas doesn't exist, add to create list
			key := h.kelasKey(row.Tingkat, row.KodeJurusan, row.Rombel)
			if _, exists := classSet[key]; !exists {
				classSet[key] = dto.ClassToCreate{
					Nama:    h.generateKelasNama(row.Tingkat, row.KodeJurusan, row.Rombel),
					Tingkat: row.Tingkat,
					Jurusan: row.KodeJurusan,
					Rombel:  row.Rombel,
				}
			}
		}

		validStudents = append(validStudents, row)
	}

	// Convert classSet to slice
	var classesToCreate []dto.ClassToCreate
	for _, cls := range classSet {
		classesToCreate = append(classesToCreate, cls)
	}

	return errors, classesToCreate, validStudents
}

func (h *ImportHandler) kelasKey(tingkat int, jurusan, rombel string) string {
	return strconv.Itoa(tingkat) + "-" + strings.ToLower(jurusan) + "-" + strings.ToUpper(rombel)
}

func (h *ImportHandler) generateKelasNama(tingkat int, jurusan, rombel string) string {
	tingkatRomawi := map[int]string{10: "X", 11: "XI", 12: "XII"}
	return tingkatRomawi[tingkat] + "-" + strings.ToUpper(jurusan) + "-" + strings.ToUpper(rombel)
}

// DownloadTemplate returns a sample CSV template
func (h *ImportHandler) DownloadTemplate(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=template_import_siswa.csv")

	template := "tingkat,kode_jurusan,rombel,nama_lengkap,nis\n"
	template += "10,rpl,A,Budi Santoso,25327004990001\n"
	template += "10,rpl,A,Siti Aminah,25327004990002\n"
	template += "11,dkv,B,Ahmad Rizki,24327004990001\n"

	return c.SendString(template)
}
