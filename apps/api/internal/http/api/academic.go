package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/academic"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

const (
	permissionManageMajors  = "academic.manage_majors"
	permissionManageYears   = "academic.manage_years"
	permissionManageClasses = "academic.manage_classes"
	permissionViewUsers     = "user.view"
)

// academicActor builds the service actor for an authenticated admin request.
func (s *Server) academicActor(ctx context.Context) (academic.Actor, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return academic.Actor{}, err
	}
	return academic.Actor{Principal: p, IP: authctx.MetaFrom(ctx).IP}, nil
}

// canViewAcademicAdmin reports whether the principal may see admin academic
// master data. Any of the related admin permissions grants visibility.
func canViewAcademicAdmin(p *auth.Principal) bool {
	return p.Can(permissionManageMajors) ||
		p.Can(permissionManageYears) ||
		p.Can(permissionManageClasses) ||
		p.Can(permissionViewUsers)
}

// requireAcademicAdmin returns the principal when it may see admin academic
// data, and a not-found error otherwise so admin areas stay invisible.
func requireAcademicAdmin(ctx context.Context) (*auth.Principal, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if !canViewAcademicAdmin(p) {
		return nil, apperr.NotFound()
	}
	return p, nil
}

func toMajor(m academic.Major) apigen.Major {
	return apigen.Major{
		Id:        m.ID,
		Code:      m.Code,
		Name:      m.Name,
		ShortName: m.ShortName,
		IsActive:  m.IsActive,
		SortOrder: int(m.SortOrder),
		GroupId:   m.GroupID,
	}
}

func toYear(y academic.AcademicYear) apigen.AcademicYear {
	return apigen.AcademicYear{
		Id:        y.ID,
		StartYear: int(y.StartYear),
		Label:     y.Label,
		Slug:      y.Slug,
		StartsOn:  toOpenAPIDate(y.StartsOn),
		EndsOn:    toOpenAPIDate(y.EndsOn),
		IsCurrent: y.IsCurrent,
	}
}

func toClass(c academic.Class) apigen.Class {
	return apigen.Class{
		Id:              c.ID,
		AcademicYearId:  c.AcademicYearID,
		MajorId:         c.MajorID,
		MajorCode:       c.MajorCode,
		GroupId:         c.GroupID,
		Grade:           int(c.Grade),
		Section:         c.Section,
		Name:            c.Name,
		Slug:            c.Slug,
		EnrollmentCount: c.EnrollmentCount,
	}
}

// toOpenAPIDate converts a domain time to a date at UTC midnight.
func toOpenAPIDate(t time.Time) openapi_types.Date {
	y, m, d := t.UTC().Date()
	return openapi_types.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

// fromOpenAPIDate converts a contract date to a domain time at UTC midnight.
func fromOpenAPIDate(d openapi_types.Date) time.Time {
	y, m, day := d.Date()
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func fromOpenAPIDatePtr(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := fromOpenAPIDate(*d)
	return &t
}

// GetMajors returns active majors. It is a public endpoint.
func (s *Server) GetMajors(ctx context.Context, _ apigen.GetMajorsRequestObject) (apigen.GetMajorsResponseObject, error) {
	majors, err := s.academic.ListMajors(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.Major, 0, len(majors))
	for _, major := range majors {
		if !major.IsActive {
			continue
		}
		items = append(items, toMajor(major))
	}

	return apigen.GetMajors200JSONResponse{Items: items}, nil
}

// GetAdminMajors returns every major, including inactive ones.
func (s *Server) GetAdminMajors(ctx context.Context, _ apigen.GetAdminMajorsRequestObject) (apigen.GetAdminMajorsResponseObject, error) {
	if _, err := requireAcademicAdmin(ctx); err != nil {
		return nil, err
	}

	majors, err := s.academic.ListMajors(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.Major, 0, len(majors))
	for _, major := range majors {
		items = append(items, toMajor(major))
	}

	return apigen.GetAdminMajors200JSONResponse{Items: items}, nil
}

// PostAdminMajor creates a major and its mirror group.
func (s *Server) PostAdminMajor(ctx context.Context, request apigen.PostAdminMajorRequestObject) (apigen.PostAdminMajorResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	in := academic.CreateMajorInput{
		Code:      request.Body.Code,
		Name:      request.Body.Name,
		ShortName: request.Body.ShortName,
	}
	if request.Body.SortOrder != nil {
		in.SortOrder = int16(*request.Body.SortOrder)
	}

	major, err := s.academic.CreateMajor(ctx, actor, in)
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminMajor201JSONResponse(toMajor(major)), nil
}

// PatchAdminMajor updates a major.
func (s *Server) PatchAdminMajor(ctx context.Context, request apigen.PatchAdminMajorRequestObject) (apigen.PatchAdminMajorResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	in := academic.UpdateMajorInput{
		Name:      request.Body.Name,
		ShortName: request.Body.ShortName,
		IsActive:  request.Body.IsActive,
	}
	if request.Body.SortOrder != nil {
		sortOrder := int16(*request.Body.SortOrder)
		in.SortOrder = &sortOrder
	}

	major, err := s.academic.UpdateMajor(ctx, actor, uuid.UUID(request.Id), in)
	if err != nil {
		return nil, err
	}

	return apigen.PatchAdminMajor200JSONResponse(toMajor(major)), nil
}

// GetAdminAcademicYears returns every academic year.
func (s *Server) GetAdminAcademicYears(ctx context.Context, _ apigen.GetAdminAcademicYearsRequestObject) (apigen.GetAdminAcademicYearsResponseObject, error) {
	if _, err := requireAcademicAdmin(ctx); err != nil {
		return nil, err
	}

	years, err := s.academic.ListYears(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.AcademicYear, 0, len(years))
	for _, year := range years {
		items = append(items, toYear(year))
	}

	return apigen.GetAdminAcademicYears200JSONResponse{Items: items}, nil
}

// PostAdminAcademicYear creates an academic year.
func (s *Server) PostAdminAcademicYear(ctx context.Context, request apigen.PostAdminAcademicYearRequestObject) (apigen.PostAdminAcademicYearResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	year, err := s.academic.CreateYear(ctx, actor, academic.CreateYearInput{
		StartYear: int16(request.Body.StartYear),
		StartsOn:  fromOpenAPIDate(request.Body.StartsOn),
		EndsOn:    fromOpenAPIDate(request.Body.EndsOn),
	})
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminAcademicYear201JSONResponse(toYear(year)), nil
}

// PatchAdminAcademicYear updates the dates of an academic year.
func (s *Server) PatchAdminAcademicYear(ctx context.Context, request apigen.PatchAdminAcademicYearRequestObject) (apigen.PatchAdminAcademicYearResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	year, err := s.academic.UpdateYear(ctx, actor, uuid.UUID(request.Id), academic.UpdateYearInput{
		StartsOn: fromOpenAPIDatePtr(request.Body.StartsOn),
		EndsOn:   fromOpenAPIDatePtr(request.Body.EndsOn),
	})
	if err != nil {
		return nil, err
	}

	return apigen.PatchAdminAcademicYear200JSONResponse(toYear(year)), nil
}

// PostAdminAcademicYearSetCurrent marks an academic year as the current one.
func (s *Server) PostAdminAcademicYearSetCurrent(ctx context.Context, request apigen.PostAdminAcademicYearSetCurrentRequestObject) (apigen.PostAdminAcademicYearSetCurrentResponseObject, error) {
	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.academic.SetCurrentYear(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.PostAdminAcademicYearSetCurrent204Response{}, nil
}

// GetAdminClasses returns classes, optionally filtered by year or major.
func (s *Server) GetAdminClasses(ctx context.Context, request apigen.GetAdminClassesRequestObject) (apigen.GetAdminClassesResponseObject, error) {
	if _, err := requireAcademicAdmin(ctx); err != nil {
		return nil, err
	}

	filter := academic.ClassFilter{}
	if request.Params.AcademicYearId != nil {
		id := uuid.UUID(*request.Params.AcademicYearId)
		filter.AcademicYearID = &id
	}
	if request.Params.MajorId != nil {
		id := uuid.UUID(*request.Params.MajorId)
		filter.MajorID = &id
	}

	classes, err := s.academic.ListClasses(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.Class, 0, len(classes))
	for _, class := range classes {
		items = append(items, toClass(class))
	}

	return apigen.GetAdminClasses200JSONResponse{Items: items}, nil
}

// PostAdminClass creates a class and its mirror group.
func (s *Server) PostAdminClass(ctx context.Context, request apigen.PostAdminClassRequestObject) (apigen.PostAdminClassResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	class, err := s.academic.CreateClass(ctx, actor, academic.CreateClassInput{
		AcademicYearID: uuid.UUID(request.Body.AcademicYearId),
		MajorID:        uuid.UUID(request.Body.MajorId),
		Grade:          int16(request.Body.Grade),
		Section:        request.Body.Section,
	})
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminClass201JSONResponse(toClass(class)), nil
}

// PatchAdminClass changes a class section.
func (s *Server) PatchAdminClass(ctx context.Context, request apigen.PatchAdminClassRequestObject) (apigen.PatchAdminClassResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	class, err := s.academic.UpdateClassSection(ctx, actor, uuid.UUID(request.Id), request.Body.Section)
	if err != nil {
		return nil, err
	}

	return apigen.PatchAdminClass200JSONResponse(toClass(class)), nil
}

// DeleteAdminClass removes an empty class.
func (s *Server) DeleteAdminClass(ctx context.Context, request apigen.DeleteAdminClassRequestObject) (apigen.DeleteAdminClassResponseObject, error) {
	actor, err := s.academicActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.academic.DeleteClass(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.DeleteAdminClass204Response{}, nil
}
