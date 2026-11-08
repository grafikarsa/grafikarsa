package api

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

// usersActor builds the service actor for an authenticated admin request.
func (s *Server) usersActor(ctx context.Context) (users.Actor, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return users.Actor{}, err
	}
	return users.Actor{Principal: p, IP: authctx.MetaFrom(ctx).IP}, nil
}

// toAdminClassRef maps a domain class reference to its API representation.
func toAdminClassRef(c *users.ClassRef) *apigen.AdminClassRef {
	if c == nil {
		return nil
	}
	return &apigen.AdminClassRef{
		Id:                c.ID,
		Name:              c.Name,
		MajorCode:         c.MajorCode,
		AcademicYearLabel: c.AcademicYearLabel,
	}
}

// toAdminUserSummary maps a domain list entry to its API representation.
func toAdminUserSummary(u users.UserSummary) apigen.AdminUserSummary {
	return apigen.AdminUserSummary{
		Id:             u.ID,
		AccountType:    apigen.AccountType(u.AccountType),
		Status:         apigen.AccountStatus(u.Status),
		DisplayName:    u.DisplayName,
		Username:       u.Username,
		Nisn:           u.NISN,
		StaffNumber:    u.StaffNumber,
		Activated:      u.Activated,
		ContentBlocked: u.ContentBlocked,
		CurrentClass:   toAdminClassRef(u.CurrentClass),
		LastLoginAt:    u.LastLoginAt,
		CreatedAt:      u.CreatedAt,
	}
}

// toAdminUserDetail maps a domain detail to its API representation.
func toAdminUserDetail(u users.UserDetail) apigen.AdminUserDetail {
	teacherMajorIDs := make([]openapi_types.UUID, len(u.TeacherMajorIDs))
	copy(teacherMajorIDs, u.TeacherMajorIDs)

	roles := make([]apigen.AdminRoleAssignment, 0, len(u.Roles))
	for _, role := range u.Roles {
		roles = append(roles, apigen.AdminRoleAssignment{
			Id:        role.ID,
			RoleSlug:  role.RoleSlug,
			RoleName:  role.RoleName,
			MajorId:   role.MajorID,
			ExpiresAt: role.ExpiresAt,
		})
	}

	enrollments := make([]apigen.AdminEnrollment, 0, len(u.Enrollments))
	for _, enrollment := range u.Enrollments {
		enrollments = append(enrollments, apigen.AdminEnrollment{
			ClassId:           enrollment.ClassID,
			ClassName:         enrollment.ClassName,
			MajorCode:         enrollment.MajorCode,
			AcademicYearLabel: enrollment.AcademicYearLabel,
			Outcome:           enrollment.Outcome,
		})
	}

	return apigen.AdminUserDetail{
		Id:                u.ID,
		AccountType:       apigen.AccountType(u.AccountType),
		Status:            apigen.AccountStatus(u.Status),
		DisplayName:       u.DisplayName,
		Username:          u.Username,
		Nisn:              u.NISN,
		StaffNumber:       u.StaffNumber,
		Activated:         u.Activated,
		ContentBlocked:    u.ContentBlocked,
		CurrentClass:      toAdminClassRef(u.CurrentClass),
		LastLoginAt:       u.LastLoginAt,
		CreatedAt:         u.CreatedAt,
		EntryYear:         toOptionalInt(u.EntryYear),
		GraduationYear:    toOptionalInt(u.GraduationYear),
		TeacherMajorIds:   teacherMajorIDs,
		Roles:             roles,
		ActiveSessions:    u.ActiveSessions,
		PasswordKind:      apigen.AdminUserDetailPasswordKind(u.PasswordKind),
		PasswordExpiresAt: u.PasswordExpiresAt,
		DeactivatedAt:     u.DeactivatedAt,
		Enrollments:       enrollments,
	}
}

// toIssuedPassword maps a freshly issued password to its API representation.
func toIssuedPassword(password string, expiresAt time.Time) apigen.IssuedPassword {
	return apigen.IssuedPassword{Password: password, ExpiresAt: expiresAt}
}

// toOptionalInt converts an optional int16 to the API integer representation.
func toOptionalInt(value *int16) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

// fromOptionalEntryYear converts an optional API integer to int16. Values
// outside the int16 range become zero so the service rejects them as invalid.
func fromOptionalEntryYear(value *int) *int16 {
	if value == nil {
		return nil
	}
	converted := int16(0)
	if *value >= math.MinInt16 && *value <= math.MaxInt16 {
		converted = int16(*value)
	}
	return &converted
}

// fromOpenAPIUUIDs copies a list of API UUIDs into a domain UUID slice.
func fromOpenAPIUUIDs(values []openapi_types.UUID) []uuid.UUID {
	out := make([]uuid.UUID, len(values))
	copy(out, values)
	return out
}

// fromOptionalOpenAPIUUIDs converts an optional API UUID list, preserving the
// "not sent" (nil) vs "sent empty" distinction.
func fromOptionalOpenAPIUUIDs(values *[]openapi_types.UUID) *[]uuid.UUID {
	if values == nil {
		return nil
	}
	converted := fromOpenAPIUUIDs(*values)
	return &converted
}

// GetAdminUsers returns users visible to the authenticated admin.
func (s *Server) GetAdminUsers(ctx context.Context, request apigen.GetAdminUsersRequestObject) (apigen.GetAdminUsersResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	filter := users.ListFilter{
		Query:  derefString(request.Params.Q),
		Cursor: derefString(request.Params.Cursor),
	}
	if request.Params.Limit != nil {
		filter.Limit = *request.Params.Limit
	}
	if request.Params.AccountType != nil {
		filter.AccountType = string(*request.Params.AccountType)
	}
	if request.Params.Status != nil {
		filter.Status = string(*request.Params.Status)
	}
	if request.Params.MajorId != nil {
		id := uuid.UUID(*request.Params.MajorId)
		filter.MajorID = &id
	}
	if request.Params.ClassId != nil {
		id := uuid.UUID(*request.Params.ClassId)
		filter.ClassID = &id
	}

	page, err := s.users.List(ctx, actor, filter)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.AdminUserSummary, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toAdminUserSummary(item))
	}

	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}

	return apigen.GetAdminUsers200JSONResponse{Items: items, NextCursor: nextCursor}, nil
}

// PostAdminUser creates one account and returns its initial password once.
func (s *Server) PostAdminUser(ctx context.Context, request apigen.PostAdminUserRequestObject) (apigen.PostAdminUserResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	in := users.CreateInput{
		AccountType: string(request.Body.AccountType),
		DisplayName: request.Body.DisplayName,
		NISN:        request.Body.Nisn,
		StaffNumber: request.Body.StaffNumber,
		EntryYear:   fromOptionalEntryYear(request.Body.EntryYear),
	}
	if request.Body.ClassId != nil {
		id := uuid.UUID(*request.Body.ClassId)
		in.ClassID = &id
	}
	if request.Body.TeacherMajorIds != nil {
		in.TeacherMajorIDs = fromOpenAPIUUIDs(*request.Body.TeacherMajorIds)
	}

	created, err := s.users.Create(ctx, actor, in)
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminUser201JSONResponse{
		User:            toAdminUserDetail(created.User),
		InitialPassword: created.InitialPassword,
	}, nil
}

// GetAdminUser returns the admin detail of one user.
func (s *Server) GetAdminUser(ctx context.Context, request apigen.GetAdminUserRequestObject) (apigen.GetAdminUserResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	detail, err := s.users.Get(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}

	return apigen.GetAdminUser200JSONResponse(toAdminUserDetail(detail)), nil
}

// PatchAdminUser changes profile fields of one user.
func (s *Server) PatchAdminUser(ctx context.Context, request apigen.PatchAdminUserRequestObject) (apigen.PatchAdminUserResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	in := users.UpdateInput{
		DisplayName:     request.Body.DisplayName,
		NISN:            request.Body.Nisn,
		StaffNumber:     request.Body.StaffNumber,
		TeacherMajorIDs: fromOptionalOpenAPIUUIDs(request.Body.TeacherMajorIds),
	}
	if request.Body.AccountType != nil {
		accountType := string(*request.Body.AccountType)
		in.AccountType = &accountType
	}

	detail, err := s.users.Update(ctx, actor, uuid.UUID(request.Id), in)
	if err != nil {
		return nil, err
	}

	return apigen.PatchAdminUser200JSONResponse(toAdminUserDetail(detail)), nil
}

// PostAdminUserDeactivate marks an account inactive and revokes its sessions.
func (s *Server) PostAdminUserDeactivate(ctx context.Context, request apigen.PostAdminUserDeactivateRequestObject) (apigen.PostAdminUserDeactivateResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.users.Deactivate(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.PostAdminUserDeactivate204Response{}, nil
}

// PostAdminUserReactivate restores an inactive account.
func (s *Server) PostAdminUserReactivate(ctx context.Context, request apigen.PostAdminUserReactivateRequestObject) (apigen.PostAdminUserReactivateResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.users.Reactivate(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.PostAdminUserReactivate204Response{}, nil
}

// PostAdminUserResetPassword issues a temporary password and revokes sessions.
func (s *Server) PostAdminUserResetPassword(ctx context.Context, request apigen.PostAdminUserResetPasswordRequestObject) (apigen.PostAdminUserResetPasswordResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	password, expiresAt, err := s.users.ResetPassword(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminUserResetPassword200JSONResponse(toIssuedPassword(password, expiresAt)), nil
}

// PostAdminUserReissueInitialPassword replaces the password of an account that
// has not been activated yet.
func (s *Server) PostAdminUserReissueInitialPassword(ctx context.Context, request apigen.PostAdminUserReissueInitialPasswordRequestObject) (apigen.PostAdminUserReissueInitialPasswordResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	password, expiresAt, err := s.users.ReissueInitialPassword(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminUserReissueInitialPassword200JSONResponse(toIssuedPassword(password, expiresAt)), nil
}

// PostAdminUserContentBlock revokes content creation for one account.
func (s *Server) PostAdminUserContentBlock(ctx context.Context, request apigen.PostAdminUserContentBlockRequestObject) (apigen.PostAdminUserContentBlockResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.users.BlockContent(ctx, actor, uuid.UUID(request.Id), request.Body.Reason); err != nil {
		return nil, err
	}

	return apigen.PostAdminUserContentBlock204Response{}, nil
}

// PostAdminUserContentUnblock restores content creation for one account.
func (s *Server) PostAdminUserContentUnblock(ctx context.Context, request apigen.PostAdminUserContentUnblockRequestObject) (apigen.PostAdminUserContentUnblockResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.users.UnblockContent(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.PostAdminUserContentUnblock204Response{}, nil
}

// GetAdminAccountCards streams a PDF of account cards for one class.
func (s *Server) GetAdminAccountCards(ctx context.Context, request apigen.GetAdminAccountCardsRequestObject) (apigen.GetAdminAccountCardsResponseObject, error) {
	actor, err := s.usersActor(ctx)
	if err != nil {
		return nil, err
	}

	document, err := s.users.AccountCardsDocument(ctx, actor, uuid.UUID(request.Params.ClassId))
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("kartu-akun-%s.pdf", document.ClassSlug)
	disposition := fmt.Sprintf("attachment; filename=%q", filename)
	return apigen.GetAdminAccountCards200ApplicationpdfResponse{
		Body:          bytes.NewReader(document.PDF),
		Headers:       apigen.GetAdminAccountCards200ResponseHeaders{ContentDisposition: &disposition},
		ContentLength: int64(len(document.PDF)),
	}, nil
}

// derefString returns the pointed-to string, or an empty string for nil.
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
