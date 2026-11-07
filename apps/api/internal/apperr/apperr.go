// Package apperr defines expected application errors that map to RFC 9457
// problem responses. Services return *Error for failures the client can act
// on (validation, not found, conflict, ...). Any other error is treated as an
// internal error by the HTTP layer.
package apperr

import (
	"errors"
	"net/http"
)

// FieldError describes a problem with a single input field.
type FieldError struct {
	Field   string
	Ref     string
	Code    string
	Message string
}

// Error is an expected, client-facing error.
type Error struct {
	Status int
	Code   string
	Title  string
	Detail string
	Fields []FieldError
	// Cause is the underlying error, kept for logs only (never sent to clients).
	Cause error
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return e.Code + ": " + e.Detail
	}
	return e.Code + ": " + e.Title
}

func (e *Error) Unwrap() error { return e.Cause }

// WithDetail returns a copy of e with Detail set.
func (e *Error) WithDetail(detail string) *Error {
	c := *e
	c.Detail = detail
	return &c
}

// WithCause returns a copy of e with Cause set.
func (e *Error) WithCause(err error) *Error {
	c := *e
	c.Cause = err
	return &c
}

// As reports whether err is (or wraps) an *Error and returns it.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// Is reports whether err is an *Error with the given code.
func Is(err error, code string) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

func newErr(status int, code, title string) *Error {
	return &Error{Status: status, Code: code, Title: title}
}

// Validation returns a 422 error listing invalid fields.
func Validation(fields ...FieldError) *Error {
	e := newErr(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Data tidak valid")
	e.Detail = "Beberapa field perlu diperbaiki."
	e.Fields = fields
	return e
}

// Field builds a FieldError.
func Field(field, code, message string) FieldError {
	return FieldError{Field: field, Code: code, Message: message}
}

// Constructors for the error codes defined in docs/revamp/api.md §1.1.
func BadRequest(detail string) *Error {
	return newErr(http.StatusBadRequest, "BAD_REQUEST", "Permintaan tidak valid").WithDetail(detail)
}
func Unauthenticated() *Error {
	return newErr(http.StatusUnauthorized, "UNAUTHENTICATED", "Silakan masuk terlebih dahulu")
}
func InvalidCredentials() *Error {
	return newErr(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Identitas atau sandi salah")
}
func ActivationRequired() *Error {
	return newErr(http.StatusForbidden, "ACTIVATION_REQUIRED", "Akun perlu diaktivasi")
}
func PasswordChangeRequired() *Error {
	return newErr(http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "Sandi sementara harus diganti")
}
func CSRFFailed() *Error {
	return newErr(http.StatusForbidden, "CSRF_FAILED", "Permintaan ditolak (token CSRF tidak valid)")
}
func ContentBlocked() *Error {
	return newErr(http.StatusForbidden, "CONTENT_BLOCKED", "Kamu tidak dapat membuat konten saat ini")
}
func NotFound() *Error {
	return newErr(http.StatusNotFound, "NOT_FOUND", "Tidak ditemukan")
}
func Conflict(detail string) *Error {
	return newErr(http.StatusConflict, "CONFLICT", "Data bentrok").WithDetail(detail)
}
func LockConflict() *Error {
	return newErr(http.StatusConflict, "LOCK_CONFLICT", "Data sudah diubah orang lain")
}
func InvalidTransition(detail string) *Error {
	return newErr(http.StatusConflict, "INVALID_TRANSITION", "Aksi tidak bisa dilakukan pada status ini").WithDetail(detail)
}
func LimitReached(detail string) *Error {
	return newErr(http.StatusConflict, "LIMIT_REACHED", "Batas tercapai").WithDetail(detail)
}
func PayloadTooLarge() *Error {
	return newErr(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Ukuran data terlalu besar")
}
func UnsupportedMediaType() *Error {
	return newErr(http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Tipe file tidak didukung")
}
func RateLimited() *Error {
	return newErr(http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak permintaan, coba lagi nanti")
}
func Unavailable(detail string) *Error {
	return newErr(http.StatusServiceUnavailable, "UNAVAILABLE", "Layanan belum siap").WithDetail(detail)
}

// NotImplemented marks an endpoint that exists in the contract but is not built yet.
func NotImplemented() *Error {
	return newErr(http.StatusNotImplemented, "NOT_IMPLEMENTED", "Fitur belum tersedia")
}
