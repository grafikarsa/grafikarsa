package profiles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// UsernameChecker is the subset of *auth.Service needed by ChangeUsername.
type UsernameChecker interface {
	CheckUsername(ctx context.Context, p *auth.Principal, raw string) (auth.UsernameCheck, error)
	InvalidateUser(ctx context.Context, userID uuid.UUID) error
}

// Service implements profile reads and writes.
type Service struct {
	st   *store.Store
	auth UsernameChecker
	now  func() time.Time
}

// NewService builds a profiles Service. authSvc supplies username checks and
// principal invalidation; it may be nil in unit tests that never change
// usernames.
func NewService(st *store.Store, authSvc UsernameChecker) *Service {
	return &Service{st: st, auth: authSvc, now: time.Now}
}

// profileUser is the user-plus-profile row shared by public and own reads.
type profileUser struct {
	ID                uuid.UUID
	Username          string
	DisplayName       string
	AccountType       string
	Status            string
	Activated         bool
	UsernameChangedAt *time.Time
	Headline          *string
	Bio               *string
	City              *string
	AvatarMediaID     *uuid.UUID
	BannerMediaID     *uuid.UUID
	TeachingSubjects  []string
}

func profileUserFromIDRow(row store.GetProfileUserByIDRow) profileUser {
	return profileUser{
		ID:                row.ID,
		Username:          row.Username,
		DisplayName:       row.DisplayName,
		AccountType:       row.AccountType,
		Status:            row.Status,
		Activated:         row.Activated,
		UsernameChangedAt: row.UsernameChangedAt,
		Headline:          row.Headline,
		Bio:               row.Bio,
		City:              row.City,
		AvatarMediaID:     row.AvatarMediaID,
		BannerMediaID:     row.BannerMediaID,
		TeachingSubjects:  row.TeachingSubjects,
	}
}

func profileUserFromUsernameRow(row store.GetProfileUserByUsernameRow) profileUser {
	return profileUser{
		ID:                row.ID,
		Username:          row.Username,
		DisplayName:       row.DisplayName,
		AccountType:       row.AccountType,
		Status:            row.Status,
		Activated:         row.Activated,
		UsernameChangedAt: row.UsernameChangedAt,
		Headline:          row.Headline,
		Bio:               row.Bio,
		City:              row.City,
		AvatarMediaID:     row.AvatarMediaID,
		BannerMediaID:     row.BannerMediaID,
		TeachingSubjects:  row.TeachingSubjects,
	}
}

// visible reports whether a user may be exposed through public endpoints.
func visible(status string, activated bool) bool {
	return activated && (status == statusActive || status == statusAlumni)
}

func (s *Service) settingInt(ctx context.Context, key string, def int) int {
	return settings.Int(ctx, s.st.Queries, key, def)
}

// GetMe returns the authenticated user's full profile.
func (s *Service) GetMe(ctx context.Context, p *auth.Principal) (*MyProfile, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	row, err := s.st.GetProfileUserByID(ctx, p.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("profiles: get own profile: %w", err)
	}
	user := profileUserFromIDRow(row)

	profile, err := s.loadProfile(ctx, s.st.Queries, &p.UserID, user)
	if err != nil {
		return nil, err
	}

	result := &MyProfile{Profile: *profile}
	if user.UsernameChangedAt != nil {
		days := s.settingInt(ctx, "profile.username_change_days", defaultUsernameChangeDays)
		available := user.UsernameChangedAt.Add(time.Duration(days) * 24 * time.Hour)
		if available.After(s.now().UTC()) {
			result.UsernameChangeAvailableAt = &available
		}
	}
	return result, nil
}

// GetPublicProfile returns a public profile by username, or a redirect to the
// new username when the old one is still inside its redirect window.
func (s *Service) GetPublicProfile(ctx context.Context, viewer *auth.Principal, username string) (*PublicProfileResult, error) {
	normalized := auth.NormalizeUsername(username)
	if auth.ValidateUsernameFormat(normalized) != "" {
		return nil, apperr.NotFound()
	}

	row, err := s.st.GetProfileUserByUsername(ctx, normalized)
	if err == nil {
		if !visible(row.Status, row.Activated) {
			return nil, apperr.NotFound()
		}
		var viewerID *uuid.UUID
		if viewer != nil {
			viewerID = &viewer.UserID
		}
		profile, err := s.loadProfile(ctx, s.st.Queries, viewerID, profileUserFromUsernameRow(row))
		if err != nil {
			return nil, err
		}
		return &PublicProfileResult{Profile: profile}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("profiles: get profile by username: %w", err)
	}

	redirect, err := s.st.GetUsernameRedirect(ctx, normalized)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("profiles: get username redirect: %w", err)
	}

	target, err := s.st.GetProfileUserByUsername(ctx, redirect)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("profiles: load redirect target: %w", err)
	}
	if !visible(target.Status, target.Activated) {
		return nil, apperr.NotFound()
	}
	return &PublicProfileResult{RedirectUsername: redirect}, nil
}

// loadProfile fills a profile from the user row and its related tables.
func (s *Service) loadProfile(ctx context.Context, q *store.Queries, viewerID *uuid.UUID, user profileUser) (*Profile, error) {
	profile := &Profile{
		ID:               user.ID,
		Username:         user.Username,
		DisplayName:      user.DisplayName,
		AccountType:      user.AccountType,
		Status:           user.Status,
		Headline:         user.Headline,
		Bio:              user.Bio,
		City:             user.City,
		AvatarMediaID:    user.AvatarMediaID,
		BannerMediaID:    user.BannerMediaID,
		TeachingSubjects: append([]string{}, user.TeachingSubjects...),
		Viewer:           Viewer{IsSelf: viewerID != nil && *viewerID == user.ID},
	}

	links, err := q.ListProfileLinks(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("profiles: list links: %w", err)
	}
	profile.Links = make([]Link, 0, len(links))
	for _, link := range links {
		profile.Links = append(profile.Links, Link{Platform: link.Platform, URL: link.Url})
	}

	skills, err := q.ListProfileSkills(ctx, store.ListProfileSkillsParams{
		ViewerID: viewerID,
		UserID:   user.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("profiles: list skills: %w", err)
	}
	profile.Skills = make([]Skill, 0, len(skills))
	for _, skill := range skills {
		profile.Skills = append(profile.Skills, Skill{
			Slug:             skill.Slug,
			Name:             skill.Name,
			Category:         skill.Category,
			EndorsementCount: skill.EndorsementCount,
			EndorsedByViewer: skill.EndorsedByViewer,
		})
	}

	badges, err := q.ListProfileBadges(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("profiles: list badges: %w", err)
	}
	profile.Badges = make([]Badge, 0, len(badges))
	for _, badge := range badges {
		profile.Badges = append(profile.Badges, Badge{
			Slug:        badge.Slug,
			Name:        badge.Name,
			Description: badge.Description,
			IconKey:     badge.IconKey,
			Note:        badge.Note,
			AwardedAt:   badge.AwardedAt,
		})
	}

	profile.TeacherMajors = make([]MajorRef, 0)
	if user.AccountType == accountTypeStudent {
		class, err := q.GetProfileLatestClass(ctx, user.ID)
		switch {
		case err == nil:
			profile.CurrentClass = &ClassRef{Name: class.ClassName, MajorCode: class.MajorCode}
		case errors.Is(err, pgx.ErrNoRows):
			// No enrollment yet; leave the class empty.
		default:
			return nil, fmt.Errorf("profiles: get latest class: %w", err)
		}
	} else if user.AccountType == accountTypeTeacher {
		majors, err := q.ListProfileTeacherMajors(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("profiles: list teacher majors: %w", err)
		}
		profile.TeacherMajors = make([]MajorRef, 0, len(majors))
		for _, major := range majors {
			profile.TeacherMajors = append(profile.TeacherMajors, MajorRef{Code: major.Code, Name: major.Name})
		}
	}

	relations, err := q.GetProfileRelations(ctx, store.GetProfileRelationsParams{
		UserID:   user.ID,
		ViewerID: viewerID,
	})
	if err != nil {
		return nil, fmt.Errorf("profiles: get relations: %w", err)
	}
	profile.Stats = Stats{Followers: relations.Followers, Following: relations.Following}
	profile.Viewer.Following = relations.ViewerFollowing

	return profile, nil
}

// UpdateProfile applies optional changes and returns the updated own profile.
func (s *Service) UpdateProfile(ctx context.Context, p *auth.Principal, in UpdateInput) (*MyProfile, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}

	var displayName *string
	if in.DisplayName != nil {
		name := strings.TrimSpace(*in.DisplayName)
		if length := utf8.RuneCountInString(name); length < 1 || length > 100 {
			return nil, apperr.Validation(apperr.Field(
				"displayName", "length", "Nama tampilan 1–100 karakter.",
			))
		}
		displayName = &name
	}

	headline, err := optionalText(in.Headline, 120, "headline", "Headline maksimal 120 karakter.")
	if err != nil {
		return nil, err
	}
	bio, err := optionalText(in.Bio, 1000, "bio", "Bio maksimal 1000 karakter.")
	if err != nil {
		return nil, err
	}
	city, err := optionalText(in.City, 80, "city", "Kota maksimal 80 karakter.")
	if err != nil {
		return nil, err
	}

	var teachingSubjects []string
	if in.TeachingSubjects != nil {
		row, err := s.st.GetProfileUserByID(ctx, p.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apperr.NotFound()
			}
			return nil, fmt.Errorf("profiles: load account type: %w", err)
		}
		if row.AccountType != accountTypeTeacher {
			return nil, apperr.Validation(apperr.Field(
				"teachingSubjects", "not_allowed", "Hanya guru yang dapat mengisi mata pelajaran.",
			))
		}
		subjects, err := validateTeachingSubjects(*in.TeachingSubjects)
		if err != nil {
			return nil, err
		}
		teachingSubjects = subjects
	}

	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		if displayName != nil {
			if err := q.UpdateUserDisplayName(ctx, store.UpdateUserDisplayNameParams{
				DisplayName: *displayName,
				UserID:      p.UserID,
			}); err != nil {
				return fmt.Errorf("profiles: update display name: %w", err)
			}
		}
		if in.Headline != nil || in.Bio != nil || in.City != nil || in.TeachingSubjects != nil {
			if err := q.EnsureProfile(ctx, p.UserID); err != nil {
				return fmt.Errorf("profiles: ensure profile: %w", err)
			}
			if err := q.UpdateProfileFields(ctx, store.UpdateProfileFieldsParams{
				SetHeadline:         in.Headline != nil,
				Headline:            headline,
				SetBio:              in.Bio != nil,
				Bio:                 bio,
				SetCity:             in.City != nil,
				City:                city,
				SetTeachingSubjects: in.TeachingSubjects != nil,
				TeachingSubjects:    teachingSubjects,
				UserID:              p.UserID,
			}); err != nil {
				return fmt.Errorf("profiles: update profile: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetMe(ctx, p)
}

// SetAvatar sets or clears the profile avatar.
func (s *Service) SetAvatar(ctx context.Context, p *auth.Principal, mediaID *uuid.UUID) (*MyProfile, error) {
	if err := s.setMedia(ctx, p, mediaID, avatarPurpose); err != nil {
		return nil, err
	}
	return s.GetMe(ctx, p)
}

// SetBanner sets or clears the profile banner.
func (s *Service) SetBanner(ctx context.Context, p *auth.Principal, mediaID *uuid.UUID) (*MyProfile, error) {
	if err := s.setMedia(ctx, p, mediaID, bannerPurpose); err != nil {
		return nil, err
	}
	return s.GetMe(ctx, p)
}

func (s *Service) setMedia(ctx context.Context, p *auth.Principal, mediaID *uuid.UUID, purpose string) error {
	if p == nil {
		return apperr.Unauthenticated()
	}
	if mediaID != nil {
		_, err := s.st.GetOwnedMedia(ctx, store.GetOwnedMediaParams{
			MediaID: *mediaID,
			UserID:  p.UserID,
			Purpose: purpose,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.Validation(apperr.Field("mediaId", "invalid", "Gambar tidak valid."))
			}
			return fmt.Errorf("profiles: check media: %w", err)
		}
	}

	var err error
	switch purpose {
	case avatarPurpose:
		err = s.st.SetProfileAvatar(ctx, store.SetProfileAvatarParams{UserID: p.UserID, MediaID: mediaID})
	case bannerPurpose:
		err = s.st.SetProfileBanner(ctx, store.SetProfileBannerParams{UserID: p.UserID, MediaID: mediaID})
	}
	if err != nil {
		return fmt.Errorf("profiles: set %s: %w", purpose, err)
	}
	return nil
}

// ReplaceLinks replaces the whole ordered link list.
func (s *Service) ReplaceLinks(ctx context.Context, p *auth.Principal, items []Link) (*MyProfile, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	maxLinks := s.settingInt(ctx, "profile.max_links", defaultMaxLinks)
	if len(items) > maxLinks {
		return nil, apperr.Validation(apperr.Field(
			"items", "limit", fmt.Sprintf("Maksimal %d tautan.", maxLinks),
		))
	}

	seen := make(map[string]struct{}, len(items))
	for i, item := range items {
		platform := strings.TrimSpace(item.Platform)
		if _, ok := linkPlatforms[platform]; !ok {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("items[%d].platform", i), "invalid", "Platform tautan tidak dikenal.",
			))
		}
		url := strings.TrimSpace(item.URL)
		if !strings.HasPrefix(url, "https://") || utf8.RuneCountInString(url) > maxLinkURLLength {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("items[%d].url", i), "invalid", "Tautan harus diawali https:// dan maksimal 500 karakter.",
			))
		}
		key := platform + "\n" + url
		if _, duplicate := seen[key]; duplicate {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("items[%d]", i), "duplicate", "Tautan ganda tidak diperbolehkan.",
			))
		}
		seen[key] = struct{}{}
		items[i].Platform = platform
		items[i].URL = url
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		if err := q.DeleteProfileLinks(ctx, p.UserID); err != nil {
			return fmt.Errorf("profiles: delete links: %w", err)
		}
		for i, item := range items {
			if err := q.InsertProfileLink(ctx, store.InsertProfileLinkParams{
				UserID:   p.UserID,
				Platform: item.Platform,
				Url:      item.URL,
				Position: int16(i),
			}); err != nil {
				return fmt.Errorf("profiles: insert link: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetMe(ctx, p)
}

// ReplaceSkills replaces the whole ordered skill list.
func (s *Service) ReplaceSkills(ctx context.Context, p *auth.Principal, rawSlugs []string) (*MyProfile, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	maxSkills := s.settingInt(ctx, "profile.max_skills", defaultMaxSkills)
	if len(rawSlugs) > maxSkills {
		return nil, apperr.Validation(apperr.Field(
			"skills", "limit", fmt.Sprintf("Maksimal %d skill.", maxSkills),
		))
	}

	slugs := make([]string, len(rawSlugs))
	seen := make(map[string]struct{}, len(rawSlugs))
	for i, raw := range rawSlugs {
		slug := strings.ToLower(strings.TrimSpace(raw))
		if slug == "" {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("skills[%d]", i), "invalid", "Slug skill tidak valid.",
			))
		}
		if _, duplicate := seen[slug]; duplicate {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("skills[%d]", i), "duplicate", "Skill ganda tidak diperbolehkan.",
			))
		}
		seen[slug] = struct{}{}
		slugs[i] = slug
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		if len(slugs) > 0 {
			found, err := q.ListSkillsBySlugs(ctx, slugs)
			if err != nil {
				return fmt.Errorf("profiles: load skills: %w", err)
			}
			if len(found) != len(slugs) {
				existing := make(map[string]struct{}, len(found))
				for _, skill := range found {
					existing[skill.Slug] = struct{}{}
				}
				for i, slug := range slugs {
					if _, ok := existing[slug]; !ok {
						return apperr.Validation(apperr.Field(
							fmt.Sprintf("skills[%d]", i), "unknown", "Skill tidak ditemukan.",
						))
					}
				}
			}
		}
		if err := q.DeleteUserSkills(ctx, p.UserID); err != nil {
			return fmt.Errorf("profiles: delete skills: %w", err)
		}
		for i, slug := range slugs {
			if err := q.InsertUserSkill(ctx, store.InsertUserSkillParams{
				UserID:   p.UserID,
				Position: int16(i),
				Slug:     slug,
			}); err != nil {
				return fmt.Errorf("profiles: insert skill: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetMe(ctx, p)
}

// SearchSkills returns up to 20 active skills matching q.
func (s *Service) SearchSkills(ctx context.Context, q string) ([]SkillOption, error) {
	query := strings.TrimSpace(q)
	if utf8.RuneCountInString(query) > maxSkillQueryLength {
		return nil, apperr.Validation(apperr.Field(
			"q", "length", "Kata kunci maksimal 100 karakter.",
		))
	}
	var param *string
	if query != "" {
		param = &query
	}
	rows, err := s.st.SearchSkills(ctx, param)
	if err != nil {
		return nil, fmt.Errorf("profiles: search skills: %w", err)
	}
	items := make([]SkillOption, 0, len(rows))
	for _, row := range rows {
		items = append(items, SkillOption{Slug: row.Slug, Name: row.Name, Category: row.Category})
	}
	return items, nil
}

// optionalText trims an optional string and maps an empty value to nil.
func optionalText(value *string, max int, field, message string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > max {
		return nil, apperr.Validation(apperr.Field(field, "too_long", message))
	}
	return &trimmed, nil
}

// validateTeachingSubjects trims and validates a teacher's subject list.
func validateTeachingSubjects(raw []string) ([]string, error) {
	if len(raw) > maxTeachingSubjects {
		return nil, apperr.Validation(apperr.Field(
			"teachingSubjects", "limit", "Maksimal 10 mata pelajaran.",
		))
	}
	subjects := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for i, subject := range raw {
		trimmed := strings.TrimSpace(subject)
		if trimmed == "" {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("teachingSubjects[%d]", i), "invalid", "Nama mata pelajaran tidak boleh kosong.",
			))
		}
		if utf8.RuneCountInString(trimmed) > maxTeachingSubjectLength {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("teachingSubjects[%d]", i), "too_long", "Nama mata pelajaran maksimal 60 karakter.",
			))
		}
		if _, duplicate := seen[trimmed]; duplicate {
			continue
		}
		seen[trimmed] = struct{}{}
		subjects = append(subjects, trimmed)
	}
	return subjects, nil
}
