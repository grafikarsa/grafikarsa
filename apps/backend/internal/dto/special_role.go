package dto

import (
	"time"

	"github.com/google/uuid"
)

// SpecialRoleDTO untuk response
type SpecialRoleDTO struct {
	ID           uuid.UUID `json:"id"`
	Nama         string    `json:"nama"`
	Description  *string   `json:"description,omitempty"`
	Color        string    `json:"color"`
	Capabilities []string  `json:"capabilities"`
	IsActive     bool      `json:"is_active"`
	UserCount    int       `json:"user_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SpecialRoleDetailDTO dengan users
type SpecialRoleDetailDTO struct {
	SpecialRoleDTO
	Users []SpecialRoleUserDTO `json:"users,omitempty"`
}

// SpecialRoleUserDTO untuk user dalam special role
type SpecialRoleUserDTO struct {
	ID         uuid.UUID  `json:"id"`
	Username   string     `json:"username"`
	Nama       string     `json:"nama"`
	AvatarURL  *string    `json:"avatar_url,omitempty"`
	KelasNama  *string    `json:"kelas_nama,omitempty"`
	AssignedAt time.Time  `json:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
}

// CreateSpecialRoleRequest untuk admin create
type CreateSpecialRoleRequest struct {
	Nama         string   `json:"nama" validate:"required,max=100"`
	Description  *string  `json:"description,omitempty"`
	Color        string   `json:"color" validate:"required,hexcolor"`
	Capabilities []string `json:"capabilities" validate:"required,min=1"`
	IsActive     *bool    `json:"is_active,omitempty"`
}

// UpdateSpecialRoleRequest untuk admin update
type UpdateSpecialRoleRequest struct {
	Nama         *string  `json:"nama,omitempty" validate:"omitempty,max=100"`
	Description  *string  `json:"description,omitempty"`
	Color        *string  `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Capabilities []string `json:"capabilities,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
}

// AssignUsersRequest untuk assign users ke role
type AssignUsersRequest struct {
	UserIDs []uuid.UUID `json:"user_ids" validate:"required,min=1"`
}

// UserSpecialRolesRequest untuk update special roles user
type UserSpecialRolesRequest struct {
	SpecialRoleIDs []uuid.UUID `json:"special_role_ids"`
}

// UserCapabilitiesDTO untuk response capabilities user
type UserCapabilitiesDTO struct {
	Capabilities []string         `json:"capabilities"`
	SpecialRoles []SpecialRoleDTO `json:"special_roles"`
}

// Daftar capabilities yang valid
var ValidCapabilities = map[string]string{
	"dashboard":          "Dashboard",
	"portfolios":         "Kelola Portfolios",
	"moderation":         "Moderasi",
	"assessments":        "Penilaian",
	"assessment_metrics": "Metrik Penilaian",
	"tags":               "Kelola Tags",
	"series":             "Kelola Series",
	"users":              "Kelola Users",
	"majors":             "Kelola Jurusan",
	"classes":            "Kelola Kelas",
	"academic_years":     "Tahun Ajaran",
	"feedback":           "Kelola Feedback",
	"special_roles":      "Kelola Special Roles",
}

// CapabilityInfo untuk frontend
type CapabilityInfo struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
}

// GetCapabilitiesList returns list of capabilities with grouping
func GetCapabilitiesList() []CapabilityInfo {
	return []CapabilityInfo{
		{Key: "dashboard", Label: "Dashboard", Group: "Overview"},
		{Key: "portfolios", Label: "Kelola Portfolios", Group: "Konten"},
		{Key: "moderation", Label: "Moderasi", Group: "Konten"},
		{Key: "assessments", Label: "Penilaian", Group: "Konten"},
		{Key: "assessment_metrics", Label: "Metrik Penilaian", Group: "Konten"},
		{Key: "tags", Label: "Kelola Tags", Group: "Konten"},
		{Key: "series", Label: "Kelola Series", Group: "Konten"},
		{Key: "users", Label: "Kelola Users", Group: "Pengguna"},
		{Key: "special_roles", Label: "Kelola Special Roles", Group: "Pengguna"},
		{Key: "majors", Label: "Kelola Jurusan", Group: "Akademik"},
		{Key: "classes", Label: "Kelola Kelas", Group: "Akademik"},
		{Key: "academic_years", Label: "Tahun Ajaran", Group: "Akademik"},
		{Key: "feedback", Label: "Kelola Feedback", Group: "Lainnya"},
	}
}
