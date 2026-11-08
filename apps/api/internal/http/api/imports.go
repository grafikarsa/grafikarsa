package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// maxImportFileBytes is the largest CSV body accepted by the upload endpoint.
const maxImportFileBytes = 2 << 20

// PostAdminImport validates an uploaded CSV synchronously and stores the job.
func (s *Server) PostAdminImport(ctx context.Context, request apigen.PostAdminImportRequestObject) (apigen.PostAdminImportResponseObject, error) {
	if s.imports == nil {
		return nil, apperr.NotFound()
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	kind, data, err := readImportUpload(request.Body)
	if err != nil {
		return nil, err
	}

	job, err := s.imports.Create(ctx, actor, kind, data)
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminImport201JSONResponse(toImportJob(job)), nil
}

// GetAdminImports lists the newest imports visible to the actor.
func (s *Server) GetAdminImports(ctx context.Context, _ apigen.GetAdminImportsRequestObject) (apigen.GetAdminImportsResponseObject, error) {
	if s.imports == nil {
		return nil, apperr.NotFound()
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	jobs, err := s.imports.List(ctx, actor)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.ImportJob, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, toImportJob(job))
	}
	return apigen.GetAdminImports200JSONResponse{Items: items}, nil
}

// GetAdminImport returns one import job visible to the actor.
func (s *Server) GetAdminImport(ctx context.Context, request apigen.GetAdminImportRequestObject) (apigen.GetAdminImportResponseObject, error) {
	if s.imports == nil {
		return nil, apperr.NotFound()
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.imports.Get(ctx, actor, request.Id)
	if err != nil {
		return nil, err
	}
	return apigen.GetAdminImport200JSONResponse(toImportJob(job)), nil
}

// PostAdminImportCommit enqueues the worker job that creates the accounts.
func (s *Server) PostAdminImportCommit(ctx context.Context, request apigen.PostAdminImportCommitRequestObject) (apigen.PostAdminImportCommitResponseObject, error) {
	if s.imports == nil {
		return nil, apperr.NotFound()
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.imports.Commit(ctx, actor, request.Id)
	if err != nil {
		return nil, err
	}
	return apigen.PostAdminImportCommit202JSONResponse(toImportJob(job)), nil
}

// readImportUpload streams the multipart body, enforcing the 2 MB file limit
// and rejecting non-text content without buffering the whole request.
func readImportUpload(reader *multipart.Reader) (string, []byte, error) {
	var (
		kind     string
		fileData []byte
		sawFile  bool
	)

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", nil, apperr.BadRequest("Formulir unggahan tidak valid")
		}

		switch part.FormName() {
		case "kind":
			raw, err := io.ReadAll(io.LimitReader(part, 64))
			if err != nil {
				part.Close()
				return "", nil, apperr.BadRequest("Formulir unggahan tidak valid")
			}
			kind = strings.TrimSpace(string(raw))
		case "file":
			sawFile = true
			content, err := io.ReadAll(io.LimitReader(part, maxImportFileBytes+1))
			if err != nil {
				part.Close()
				return "", nil, apperr.BadRequest("Berkas tidak dapat dibaca")
			}
			if len(content) > maxImportFileBytes {
				part.Close()
				return "", nil, apperr.PayloadTooLarge()
			}
			if err := validateImportContent(part.Header.Get("Content-Type"), content); err != nil {
				part.Close()
				return "", nil, err
			}
			fileData = content
		}
		part.Close()
	}

	if kind == "" {
		return "", nil, apperr.Validation(apperr.Field(
			"kind", "required", "Jenis impor wajib dipilih.",
		))
	}
	if !sawFile {
		return "", nil, apperr.Validation(apperr.Field(
			"file", "required", "Berkas CSV wajib diunggah.",
		))
	}
	return kind, fileData, nil
}

// validateImportContent rejects declared non-text media types and binary
// content. CSV uploads from Excel often arrive as application/vnd.ms-excel.
func validateImportContent(declared string, data []byte) error {
	mediaType := strings.ToLower(strings.TrimSpace(declared))
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	switch {
	case mediaType == "":
	case strings.HasPrefix(mediaType, "text/"):
	case mediaType == "application/csv", mediaType == "application/vnd.ms-excel":
	default:
		return apperr.UnsupportedMediaType()
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return apperr.UnsupportedMediaType()
	}
	return nil
}

// toImportJob maps an import job to its API representation.
func toImportJob(job users.ImportJob) apigen.ImportJob {
	rowErrors := make([]apigen.ImportRowError, 0, len(job.Errors))
	for _, rowError := range job.Errors {
		rowErrors = append(rowErrors, apigen.ImportRowError{
			Row:     rowError.Row,
			Field:   rowError.Field,
			Message: rowError.Message,
		})
	}

	skipped := make([]apigen.ImportSkippedRow, 0, len(job.Skipped))
	for _, row := range job.Skipped {
		skipped = append(skipped, apigen.ImportSkippedRow{
			Row:    row.Row,
			Reason: row.Reason,
		})
	}

	byClass := job.ByClass
	if byClass == nil {
		byClass = map[string]int{}
	}

	return apigen.ImportJob{
		Id:              job.ID,
		Kind:            apigen.ImportKind(job.Kind),
		Status:          apigen.ImportStatus(job.Status),
		TotalRows:       job.TotalRows,
		ValidRows:       job.ValidRows,
		ErrorCountTotal: job.ErrorCountTotal,
		Errors:          rowErrors,
		ByClass:         byClass,
		Created:         job.Created,
		Skipped:         skipped,
		CreatedAt:       job.CreatedAt,
		CommittedAt:     job.CommittedAt,
		CreatedBy:       job.CreatedBy,
	}
}
