package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// PostMedia accepts one multipart upload and streams the file part directly to
// the media service.
func (s *Server) PostMedia(ctx context.Context, request apigen.PostMediaRequestObject) (apigen.PostMediaResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body multipart kosong.")
	}

	var purpose string
	for {
		part, err := request.Body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, mediaPartError(err)
		}

		switch part.FormName() {
		case "purpose":
			value, err := io.ReadAll(io.LimitReader(part, 128))
			if err != nil {
				return nil, mediaPartError(err)
			}
			purpose = strings.TrimSpace(string(value))
		case "file":
			if purpose == "" {
				return nil, apperr.BadRequest("Field purpose harus dikirim sebelum file.")
			}

			asset, err := s.media.Upload(ctx, principal, media.Purpose(purpose), part.FileName(), part)
			if err != nil {
				return nil, err
			}
			return apigen.PostMedia202JSONResponse(toMediaAsset(asset)), nil
		default:
			if _, err := io.Copy(io.Discard, part); err != nil {
				return nil, mediaPartError(err)
			}
		}
	}

	return nil, apperr.BadRequest("File wajib diunggah.")
}

// GetMedia returns the status and variants of one owned asset.
func (s *Server) GetMedia(ctx context.Context, request apigen.GetMediaRequestObject) (apigen.GetMediaResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	asset, err := s.media.Get(ctx, principal, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}
	return apigen.GetMedia200JSONResponse(toMediaAsset(asset)), nil
}

// GetMediaVariant streams the original or a processed variant.
func (s *Server) GetMediaVariant(ctx context.Context, request apigen.GetMediaVariantRequestObject) (apigen.GetMediaVariantResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	reader, info, err := s.media.OpenVariant(ctx, principal, uuid.UUID(request.Id), request.Variant)
	if err != nil {
		return nil, err
	}

	cacheControl := "private, max-age=300"
	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return apigen.GetMediaVariant200ApplicationoctetStreamResponse{
		Body: reader,
		Headers: apigen.GetMediaVariant200ResponseHeaders{
			CacheControl: &cacheControl,
			ContentType:  &contentType,
		},
		ContentLength: info.Size,
	}, nil
}

// DeleteMedia soft-deletes an owned asset that is no longer referenced.
func (s *Server) DeleteMedia(ctx context.Context, request apigen.DeleteMediaRequestObject) (apigen.DeleteMediaResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.media.Delete(ctx, principal, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}
	return apigen.DeleteMedia204Response{}, nil
}

// mediaPartError maps multipart decoding failures to the right client error.
func mediaPartError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.PayloadTooLarge()
	}
	return apperr.BadRequest("Berkas tidak dapat dibaca.")
}

// toMediaAsset maps a domain asset to the API representation. Object keys stay
// on the server.
func toMediaAsset(asset media.Asset) apigen.MediaAsset {
	variants := make(map[string]apigen.MediaVariant, len(asset.Variants))
	for name, variant := range asset.Variants {
		variants[name] = apigen.MediaVariant{
			Format: variant.Format,
			W:      variant.W,
			H:      variant.H,
			Bytes:  variant.Bytes,
		}
	}

	return apigen.MediaAsset{
		Id:               asset.ID,
		Kind:             apigen.MediaKind(asset.Kind),
		Purpose:          apigen.MediaPurpose(asset.Purpose),
		MimeType:         asset.MimeType,
		SizeBytes:        asset.SizeBytes,
		Width:            toOptionalInt32(asset.Width),
		Height:           toOptionalInt32(asset.Height),
		PageCount:        toOptionalInt32(asset.PageCount),
		Status:           apigen.MediaStatus(asset.Status),
		Variants:         variants,
		IsPublic:         asset.IsPublic,
		Blurhash:         asset.Blurhash,
		DominantColor:    asset.DominantColor,
		OriginalFilename: asset.OriginalFilename,
		CreatedAt:        asset.CreatedAt,
	}
}

func toOptionalInt32(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}
