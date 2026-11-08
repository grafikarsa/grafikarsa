package api

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/profiles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

// toProfileLinks maps a domain link list to its API representation.
func toProfileLinks(links []profiles.Link) []apigen.ProfileLink {
	out := make([]apigen.ProfileLink, 0, len(links))
	for _, link := range links {
		out = append(out, apigen.ProfileLink{
			Platform: apigen.ProfileLinkPlatform(link.Platform),
			Url:      link.URL,
		})
	}
	return out
}

// toProfileSkills maps a domain skill list to its API representation.
func toProfileSkills(skills []profiles.Skill) []apigen.ProfileSkill {
	out := make([]apigen.ProfileSkill, 0, len(skills))
	for _, skill := range skills {
		out = append(out, apigen.ProfileSkill{
			Slug:             skill.Slug,
			Name:             skill.Name,
			Category:         apigen.SkillCategory(skill.Category),
			EndorsementCount: skill.EndorsementCount,
			EndorsedByViewer: skill.EndorsedByViewer,
		})
	}
	return out
}

// toProfileBadges maps a domain badge list to its API representation.
func toProfileBadges(badges []profiles.Badge) []apigen.ProfileBadge {
	out := make([]apigen.ProfileBadge, 0, len(badges))
	for _, badge := range badges {
		out = append(out, apigen.ProfileBadge{
			Slug:        badge.Slug,
			Name:        badge.Name,
			Description: badge.Description,
			IconKey:     badge.IconKey,
			Note:        badge.Note,
			AwardedAt:   badge.AwardedAt,
		})
	}
	return out
}

// toProfileClass maps the latest class to its API representation.
func toProfileClass(class *profiles.ClassRef) *apigen.ProfileClassRef {
	if class == nil {
		return nil
	}
	return &apigen.ProfileClassRef{Name: class.Name, MajorCode: class.MajorCode}
}

// toProfileMajors maps teacher majors to their API representation.
func toProfileMajors(majors []profiles.MajorRef) []apigen.ProfileMajorRef {
	out := make([]apigen.ProfileMajorRef, 0, len(majors))
	for _, major := range majors {
		out = append(out, apigen.ProfileMajorRef{Code: major.Code, Name: major.Name})
	}
	return out
}

// fromOptionalOpenAPIUUID converts an optional API UUID to the domain type.
func fromOptionalOpenAPIUUID(id *openapi_types.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	converted := uuid.UUID(*id)
	return &converted
}

// toPublicProfile maps a domain public profile to its API representation.
func toPublicProfile(profile *profiles.Profile) apigen.PublicProfile {
	return apigen.PublicProfile{
		Id:               profile.ID,
		Username:         profile.Username,
		DisplayName:      profile.DisplayName,
		AccountType:      apigen.AccountType(profile.AccountType),
		Status:           apigen.ProfileStatus(profile.Status),
		Headline:         profile.Headline,
		Bio:              profile.Bio,
		City:             profile.City,
		AvatarMediaId:    toOptionalOpenAPIUUID(profile.AvatarMediaID),
		AvatarUrl:        nil,
		BannerMediaId:    toOptionalOpenAPIUUID(profile.BannerMediaID),
		BannerUrl:        nil,
		Links:            toProfileLinks(profile.Links),
		Skills:           toProfileSkills(profile.Skills),
		Badges:           toProfileBadges(profile.Badges),
		CurrentClass:     toProfileClass(profile.CurrentClass),
		TeacherMajors:    toProfileMajors(profile.TeacherMajors),
		TeachingSubjects: profile.TeachingSubjects,
		Stats: apigen.ProfileStats{
			Followers: profile.Stats.Followers,
			Following: profile.Stats.Following,
		},
		Viewer: apigen.ProfileViewer{
			Following: profile.Viewer.Following,
			IsSelf:    profile.Viewer.IsSelf,
		},
	}
}

// toMyProfile maps a domain own profile to its API representation.
func toMyProfile(profile *profiles.MyProfile) apigen.MyProfile {
	public := toPublicProfile(&profile.Profile)
	return apigen.MyProfile{
		Id:                        public.Id,
		Username:                  public.Username,
		DisplayName:               public.DisplayName,
		AccountType:               public.AccountType,
		Status:                    public.Status,
		Headline:                  public.Headline,
		Bio:                       public.Bio,
		City:                      public.City,
		AvatarMediaId:             public.AvatarMediaId,
		AvatarUrl:                 public.AvatarUrl,
		BannerMediaId:             public.BannerMediaId,
		BannerUrl:                 public.BannerUrl,
		Links:                     public.Links,
		Skills:                    public.Skills,
		Badges:                    public.Badges,
		CurrentClass:              public.CurrentClass,
		TeacherMajors:             public.TeacherMajors,
		TeachingSubjects:          public.TeachingSubjects,
		Stats:                     public.Stats,
		Viewer:                    public.Viewer,
		UsernameChangeAvailableAt: profile.UsernameChangeAvailableAt,
	}
}

// toExperience maps a domain experience to its API representation.
func toExperience(experience profiles.Experience) apigen.Experience {
	skills := make([]apigen.ExperienceSkill, 0, len(experience.Skills))
	for _, skill := range experience.Skills {
		skills = append(skills, apigen.ExperienceSkill{Slug: skill.Slug, Name: skill.Name})
	}
	var employmentType *apigen.EmploymentType
	if experience.EmploymentType != nil {
		converted := apigen.EmploymentType(*experience.EmploymentType)
		employmentType = &converted
	}
	return apigen.Experience{
		Id:             experience.ID,
		Kind:           apigen.ExperienceKind(experience.Kind),
		Title:          experience.Title,
		Organization:   experience.Organization,
		Location:       experience.Location,
		EmploymentType: employmentType,
		StartMonth:     experience.StartMonth,
		EndMonth:       experience.EndMonth,
		IsCurrent:      experience.IsCurrent,
		Description:    experience.Description,
		CredentialUrl:  experience.CredentialURL,
		IsSystem:       experience.IsSystem,
		Skills:         skills,
	}
}

// toExperienceList maps domain experiences to their API representation.
func toExperienceList(items []profiles.Experience) apigen.ExperienceList {
	out := make([]apigen.Experience, 0, len(items))
	for _, item := range items {
		out = append(out, toExperience(item))
	}
	return apigen.ExperienceList{Items: out}
}

// fromExperienceInput maps an API experience payload to the domain type.
func fromExperienceInput(body *apigen.ExperienceInput) profiles.ExperienceInput {
	in := profiles.ExperienceInput{
		Kind:           string(body.Kind),
		Title:          body.Title,
		Organization:   body.Organization,
		Location:       body.Location,
		StartMonth:     body.StartMonth,
		EndMonth:       body.EndMonth,
		IsCurrent:      body.IsCurrent,
		Description:    body.Description,
		CredentialURL:  body.CredentialUrl,
		EmploymentType: nil,
	}
	if body.EmploymentType != nil {
		value := string(*body.EmploymentType)
		in.EmploymentType = &value
	}
	if body.Skills != nil {
		in.Skills = append([]string{}, (*body.Skills)...)
	}
	return in
}

// toUserMiniPage maps a domain user card page to its API representation.
func toUserMiniPage(page profiles.UserMiniPage) apigen.UserMiniPage {
	items := make([]apigen.UserMini, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, apigen.UserMini{
			Username:      item.Username,
			DisplayName:   item.DisplayName,
			AccountType:   apigen.AccountType(item.AccountType),
			AvatarMediaId: toOptionalOpenAPIUUID(item.AvatarMediaID),
		})
	}
	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}
	return apigen.UserMiniPage{Items: items, NextCursor: nextCursor}
}

// GetMe returns the authenticated user's own profile.
func (s *Server) GetMe(ctx context.Context, _ apigen.GetMeRequestObject) (apigen.GetMeResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.GetMe(ctx, p)
	if err != nil {
		return nil, err
	}
	return apigen.GetMe200JSONResponse(toMyProfile(profile)), nil
}

// PatchMeProfile updates the authenticated user's profile.
func (s *Server) PatchMeProfile(ctx context.Context, request apigen.PatchMeProfileRequestObject) (apigen.PatchMeProfileResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.UpdateProfile(ctx, p, profiles.UpdateInput{
		DisplayName:      request.Body.DisplayName,
		Headline:         request.Body.Headline,
		Bio:              request.Body.Bio,
		City:             request.Body.City,
		TeachingSubjects: request.Body.TeachingSubjects,
	})
	if err != nil {
		return nil, err
	}
	return apigen.PatchMeProfile200JSONResponse(toMyProfile(profile)), nil
}

// PutMeAvatar sets or clears the authenticated user's avatar.
func (s *Server) PutMeAvatar(ctx context.Context, request apigen.PutMeAvatarRequestObject) (apigen.PutMeAvatarResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.SetAvatar(ctx, p, fromOptionalOpenAPIUUID(request.Body.MediaId))
	if err != nil {
		return nil, err
	}
	return apigen.PutMeAvatar200JSONResponse(toMyProfile(profile)), nil
}

// PutMeBanner sets or clears the authenticated user's banner.
func (s *Server) PutMeBanner(ctx context.Context, request apigen.PutMeBannerRequestObject) (apigen.PutMeBannerResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.SetBanner(ctx, p, fromOptionalOpenAPIUUID(request.Body.MediaId))
	if err != nil {
		return nil, err
	}
	return apigen.PutMeBanner200JSONResponse(toMyProfile(profile)), nil
}

// PutMeLinks replaces the authenticated user's ordered link list.
func (s *Server) PutMeLinks(ctx context.Context, request apigen.PutMeLinksRequestObject) (apigen.PutMeLinksResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]profiles.Link, 0, len(request.Body.Items))
	for _, item := range request.Body.Items {
		items = append(items, profiles.Link{Platform: string(item.Platform), URL: item.Url})
	}
	profile, err := s.profiles.ReplaceLinks(ctx, p, items)
	if err != nil {
		return nil, err
	}
	return apigen.PutMeLinks200JSONResponse(toMyProfile(profile)), nil
}

// PutMeSkills replaces the authenticated user's ordered skill list.
func (s *Server) PutMeSkills(ctx context.Context, request apigen.PutMeSkillsRequestObject) (apigen.PutMeSkillsResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.ReplaceSkills(ctx, p, request.Body.Skills)
	if err != nil {
		return nil, err
	}
	return apigen.PutMeSkills200JSONResponse(toMyProfile(profile)), nil
}

// PatchMeUsername changes the authenticated user's username.
func (s *Server) PatchMeUsername(ctx context.Context, request apigen.PatchMeUsernameRequestObject) (apigen.PatchMeUsernameResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.ChangeUsername(ctx, p, request.Body.Username)
	if err != nil {
		return nil, err
	}
	return apigen.PatchMeUsername200JSONResponse(toMyProfile(profile)), nil
}

// GetMeExperiences lists the authenticated user's experiences.
func (s *Server) GetMeExperiences(ctx context.Context, _ apigen.GetMeExperiencesRequestObject) (apigen.GetMeExperiencesResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.profiles.ListMyExperiences(ctx, p)
	if err != nil {
		return nil, err
	}
	return apigen.GetMeExperiences200JSONResponse(toExperienceList(items)), nil
}

// PostMeExperience creates an experience for the authenticated user.
func (s *Server) PostMeExperience(ctx context.Context, request apigen.PostMeExperienceRequestObject) (apigen.PostMeExperienceResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	experience, err := s.profiles.CreateExperience(ctx, p, fromExperienceInput(request.Body))
	if err != nil {
		return nil, err
	}
	return apigen.PostMeExperience201JSONResponse(toExperience(*experience)), nil
}

// PatchMeExperience updates one of the authenticated user's experiences.
func (s *Server) PatchMeExperience(ctx context.Context, request apigen.PatchMeExperienceRequestObject) (apigen.PatchMeExperienceResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	experience, err := s.profiles.UpdateExperience(ctx, p, uuid.UUID(request.Id), fromExperienceInput(request.Body))
	if err != nil {
		return nil, err
	}
	return apigen.PatchMeExperience200JSONResponse(toExperience(*experience)), nil
}

// DeleteMeExperience deletes one of the authenticated user's experiences.
func (s *Server) DeleteMeExperience(ctx context.Context, request apigen.DeleteMeExperienceRequestObject) (apigen.DeleteMeExperienceResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.profiles.DeleteExperience(ctx, p, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}
	return apigen.DeleteMeExperience204Response{}, nil
}

// GetUser returns a public profile, or redirects a renamed username.
func (s *Server) GetUser(ctx context.Context, request apigen.GetUserRequestObject) (apigen.GetUserResponseObject, error) {
	viewer := authctx.Principal(ctx)
	result, err := s.profiles.GetPublicProfile(ctx, viewer, request.Username)
	if err != nil {
		return nil, err
	}
	if result.RedirectUsername != "" {
		return apigen.GetUser301Response{
			Headers: apigen.GetUser301ResponseHeaders{Location: "/@" + result.RedirectUsername},
		}, nil
	}
	return apigen.GetUser200JSONResponse(toPublicProfile(result.Profile)), nil
}

// GetUserExperiences lists a public user's experiences.
func (s *Server) GetUserExperiences(ctx context.Context, request apigen.GetUserExperiencesRequestObject) (apigen.GetUserExperiencesResponseObject, error) {
	items, err := s.profiles.ListExperiences(ctx, request.Username)
	if err != nil {
		return nil, err
	}
	return apigen.GetUserExperiences200JSONResponse(toExperienceList(items)), nil
}

// GetUserFollowers lists a public user's followers.
func (s *Server) GetUserFollowers(ctx context.Context, request apigen.GetUserFollowersRequestObject) (apigen.GetUserFollowersResponseObject, error) {
	cursor := ""
	if request.Params.Cursor != nil {
		cursor = *request.Params.Cursor
	}
	limit := 0
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}
	page, err := s.profiles.ListFollowers(ctx, request.Username, cursor, limit)
	if err != nil {
		return nil, err
	}
	return apigen.GetUserFollowers200JSONResponse(toUserMiniPage(page)), nil
}

// GetUserFollowing lists users a public user follows.
func (s *Server) GetUserFollowing(ctx context.Context, request apigen.GetUserFollowingRequestObject) (apigen.GetUserFollowingResponseObject, error) {
	cursor := ""
	if request.Params.Cursor != nil {
		cursor = *request.Params.Cursor
	}
	limit := 0
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}
	page, err := s.profiles.ListFollowing(ctx, request.Username, cursor, limit)
	if err != nil {
		return nil, err
	}
	return apigen.GetUserFollowing200JSONResponse(toUserMiniPage(page)), nil
}

// PutUserFollow makes the authenticated user follow another user.
func (s *Server) PutUserFollow(ctx context.Context, request apigen.PutUserFollowRequestObject) (apigen.PutUserFollowResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.profiles.Follow(ctx, p, request.Username); err != nil {
		return nil, err
	}
	return apigen.PutUserFollow204Response{}, nil
}

// DeleteUserFollow makes the authenticated user unfollow another user.
func (s *Server) DeleteUserFollow(ctx context.Context, request apigen.DeleteUserFollowRequestObject) (apigen.DeleteUserFollowResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.profiles.Unfollow(ctx, p, request.Username); err != nil {
		return nil, err
	}
	return apigen.DeleteUserFollow204Response{}, nil
}

// PutUserSkillEndorsement records an endorsement of another user's skill.
func (s *Server) PutUserSkillEndorsement(ctx context.Context, request apigen.PutUserSkillEndorsementRequestObject) (apigen.PutUserSkillEndorsementResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.profiles.Endorse(ctx, p, request.Username, request.SkillSlug); err != nil {
		return nil, err
	}
	return apigen.PutUserSkillEndorsement204Response{}, nil
}

// DeleteUserSkillEndorsement removes an endorsement of another user's skill.
func (s *Server) DeleteUserSkillEndorsement(ctx context.Context, request apigen.DeleteUserSkillEndorsementRequestObject) (apigen.DeleteUserSkillEndorsementResponseObject, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.profiles.Unendorse(ctx, p, request.Username, request.SkillSlug); err != nil {
		return nil, err
	}
	return apigen.DeleteUserSkillEndorsement204Response{}, nil
}

// GetTaxonomySkills searches the active skill catalog.
func (s *Server) GetTaxonomySkills(ctx context.Context, request apigen.GetTaxonomySkillsRequestObject) (apigen.GetTaxonomySkillsResponseObject, error) {
	query := ""
	if request.Params.Q != nil {
		query = *request.Params.Q
	}
	items, err := s.profiles.SearchSkills(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]apigen.SkillOption, 0, len(items))
	for _, item := range items {
		out = append(out, apigen.SkillOption{
			Slug:     item.Slug,
			Name:     item.Name,
			Category: apigen.SkillCategory(item.Category),
		})
	}
	return apigen.GetTaxonomySkills200JSONResponse(apigen.SkillOptionList{Items: out}), nil
}
