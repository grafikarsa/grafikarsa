package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"grafikarsa/internal/domain"
)

var (
	validate *validator.Validate

	// Reserved usernames that cannot be used
	reservedUsernames = map[string]bool{
		"admin": true, "administrator": true, "root": true, "system": true,
		"api": true, "auth": true, "login": true, "logout": true, "register": true,
		"signup": true, "signin": true, "signout": true, "password": true,
		"me": true, "profile": true, "settings": true, "account": true,
		"user": true, "users": true, "portfolio": true, "portfolios": true,
		"search": true, "feed": true, "explore": true, "discover": true,
		"help": true, "support": true, "contact": true, "about": true,
		"terms": true, "privacy": true, "legal": true, "dmca": true,
		"blog": true, "news": true, "press": true, "jobs": true,
		"static": true, "assets": true, "public": true, "private": true,
		"uploads": true, "files": true, "images": true, "media": true,
		"grafikarsa": true, "smkn4": true, "smkn4malang": true,
		"null": true, "undefined": true, "true": true, "false": true,
	}

	// Username regex: 3-50 chars, lowercase alphanumeric and underscores only
	usernameRegex = regexp.MustCompile(`^[a-z0-9_]{3,50}$`)

	// Slug regex: lowercase alphanumeric and hyphens
	slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

	// Valid social platforms
	validSocialPlatforms = map[string]bool{
		"facebook": true, "instagram": true, "github": true, "linkedin": true,
		"twitter": true, "website": true, "tiktok": true, "youtube": true,
		"behance": true, "dribbble": true, "threads": true, "bluesky": true,
		"medium": true, "gitlab": true,
	}
)

func init() {
	validate = validator.New()

	// Use JSON tag names in validation errors
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Custom validations
	validate.RegisterValidation("username", validateUsername)
	validate.RegisterValidation("not_reserved_username", validateNotReservedUsername)
	validate.RegisterValidation("slug", validateSlug)
	validate.RegisterValidation("social_platform", validateSocialPlatform)
	validate.RegisterValidation("grade_level", validateGradeLevel)
	validate.RegisterValidation("group_letter", validateGroupLetter)
	validate.RegisterValidation("block_type", validateBlockType)
	validate.RegisterValidation("upload_type", validateUploadType)
	validate.RegisterValidation("portfolio_status", validatePortfolioStatus)
	validate.RegisterValidation("user_role", validateUserRole)
	validate.RegisterValidation("user_status", validateUserStatus)
}

// Validate validates a struct and returns formatted error details
func Validate(s interface{}) []ErrorDetail {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var details []ErrorDetail
	for _, err := range err.(validator.ValidationErrors) {
		details = append(details, ErrorDetail{
			Field:   err.Field(),
			Message: formatValidationError(err),
		})
	}
	return details
}

// formatValidationError returns a human-readable validation error message
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", err.Field())
	case "email":
		return "Format email tidak valid"
	case "min":
		if err.Kind() == reflect.String {
			return fmt.Sprintf("%s minimal %s karakter", err.Field(), err.Param())
		}
		return fmt.Sprintf("%s minimal %s", err.Field(), err.Param())
	case "max":
		if err.Kind() == reflect.String {
			return fmt.Sprintf("%s maksimal %s karakter", err.Field(), err.Param())
		}
		return fmt.Sprintf("%s maksimal %s", err.Field(), err.Param())
	case "username":
		return "Username hanya boleh berisi huruf kecil, angka, dan underscore (3-50 karakter)"
	case "not_reserved_username":
		return "Username tidak boleh menggunakan reserved words"
	case "slug":
		return "Slug hanya boleh berisi huruf kecil, angka, dan tanda hubung"
	case "uuid":
		return "Format UUID tidak valid"
	case "url":
		return "Format URL tidak valid"
	case "oneof":
		return fmt.Sprintf("Nilai harus salah satu dari: %s", err.Param())
	case "eqfield":
		return fmt.Sprintf("Harus sama dengan %s", err.Param())
	case "grade_level":
		return "Tingkat kelas harus 10, 11, atau 12"
	case "group_letter":
		return "Rombel hanya boleh satu huruf A-Z"
	case "block_type":
		return "Block type tidak valid"
	case "upload_type":
		return "Upload type tidak valid"
	case "portfolio_status":
		return "Portfolio status tidak valid"
	case "user_role":
		return "User role tidak valid"
	case "user_status":
		return "User status tidak valid"
	case "social_platform":
		return "Social platform tidak valid"
	case "gte":
		return fmt.Sprintf("%s harus lebih besar atau sama dengan %s", err.Field(), err.Param())
	case "lte":
		return fmt.Sprintf("%s harus lebih kecil atau sama dengan %s", err.Field(), err.Param())
	case "gt":
		return fmt.Sprintf("%s harus lebih besar dari %s", err.Field(), err.Param())
	case "lt":
		return fmt.Sprintf("%s harus lebih kecil dari %s", err.Field(), err.Param())
	default:
		return fmt.Sprintf("Validasi %s gagal", err.Field())
	}
}

// Custom validators

func validateUsername(fl validator.FieldLevel) bool {
	return usernameRegex.MatchString(fl.Field().String())
}

func validateNotReservedUsername(fl validator.FieldLevel) bool {
	return !reservedUsernames[strings.ToLower(fl.Field().String())]
}

func validateSlug(fl validator.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}

func validateSocialPlatform(fl validator.FieldLevel) bool {
	return validSocialPlatforms[strings.ToLower(fl.Field().String())]
}

func validateGradeLevel(fl validator.FieldLevel) bool {
	level := fl.Field().String()
	return level == "10" || level == "11" || level == "12"
}

func validateGroupLetter(fl validator.FieldLevel) bool {
	letter := fl.Field().String()
	return len(letter) == 1 && letter >= "A" && letter <= "Z"
}

func validateBlockType(fl validator.FieldLevel) bool {
	t := fl.Field().String()
	validTypes := map[string]bool{
		"text": true, "image": true, "table": true, "youtube": true, "button": true,
	}
	return validTypes[t]
}

func validateUploadType(fl validator.FieldLevel) bool {
	t := fl.Field().String()
	validTypes := map[string]bool{
		"avatar": true, "banner": true, "thumbnail": true, "portfolio_image": true,
	}
	return validTypes[t]
}

func validatePortfolioStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := map[string]bool{
		"draft": true, "pending_review": true, "rejected": true, "published": true, "archived": true,
	}
	return validStatuses[status]
}

func validateUserRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	validRoles := map[string]bool{
		"admin": true, "student": true, "alumni": true,
	}
	return validRoles[role]
}

func validateUserStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := map[string]bool{
		"active": true, "graduated": true, "dropped_out": true, "inactive": true,
	}
	return validStatuses[status]
}

// IsReservedUsername checks if a username is reserved
func IsReservedUsername(username string) bool {
	return reservedUsernames[strings.ToLower(username)]
}

// IsValidUsername checks if username matches the required format
func IsValidUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

// IsValidSocialPlatform checks if a platform is valid
func IsValidSocialPlatform(platform string) bool {
	return validSocialPlatforms[strings.ToLower(platform)]
}

// ParseUUID parses a string into a UUID
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ToUserRole converts a string to UserRole
func ToUserRole(s string) domain.UserRole {
	switch s {
	case "admin":
		return domain.RoleAdmin
	case "student":
		return domain.RoleStudent
	case "alumni":
		return domain.RoleAlumni
	default:
		return domain.RoleStudent
	}
}

// ToUserStatus converts a string to UserStatus
func ToUserStatus(s string) domain.UserStatus {
	switch s {
	case "active":
		return domain.StatusActive
	case "graduated":
		return domain.StatusGraduated
	case "dropped_out":
		return domain.StatusDroppedOut
	case "inactive":
		return domain.StatusInactive
	default:
		return domain.StatusActive
	}
}
