package profiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// linkPlatforms mirrors the profile_links.platform CHECK constraint.
var linkPlatforms = map[string]struct{}{
	"github": {}, "gitlab": {}, "behance": {}, "dribbble": {}, "artstation": {},
	"youtube": {}, "instagram": {}, "tiktok": {}, "linkedin": {}, "x": {}, "figma": {},
	"website": {},
}

// resolveVisible loads a user by username for a public interaction, hiding
// missing, unactivated, and inactive accounts behind NotFound.
func (s *Service) resolveVisible(ctx context.Context, username string) (profileUser, error) {
	normalized := auth.NormalizeUsername(username)
	if auth.ValidateUsernameFormat(normalized) != "" {
		return profileUser{}, apperr.NotFound()
	}
	row, err := s.st.GetProfileUserByUsername(ctx, normalized)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profileUser{}, apperr.NotFound()
		}
		return profileUser{}, fmt.Errorf("profiles: resolve user: %w", err)
	}
	if !visible(row.Status, row.Activated) {
		return profileUser{}, apperr.NotFound()
	}
	return profileUserFromUsernameRow(row), nil
}

// Follow makes the viewer follow the target. It is idempotent.
func (s *Service) Follow(ctx context.Context, p *auth.Principal, username string) error {
	target, err := s.followTarget(ctx, p, username)
	if err != nil {
		return err
	}
	if err := s.st.InsertFollow(ctx, store.InsertFollowParams{
		FollowerID: p.UserID,
		FolloweeID: target.ID,
	}); err != nil {
		return fmt.Errorf("profiles: follow: %w", err)
	}
	return nil
}

// Unfollow makes the viewer stop following the target. It is idempotent.
func (s *Service) Unfollow(ctx context.Context, p *auth.Principal, username string) error {
	target, err := s.followTarget(ctx, p, username)
	if err != nil {
		return err
	}
	if err := s.st.DeleteFollow(ctx, store.DeleteFollowParams{
		FollowerID: p.UserID,
		FolloweeID: target.ID,
	}); err != nil {
		return fmt.Errorf("profiles: unfollow: %w", err)
	}
	return nil
}

func (s *Service) followTarget(ctx context.Context, p *auth.Principal, username string) (profileUser, error) {
	if p == nil {
		return profileUser{}, apperr.Unauthenticated()
	}
	target, err := s.resolveVisible(ctx, username)
	if err != nil {
		return profileUser{}, err
	}
	if target.ID == p.UserID {
		return profileUser{}, apperr.Validation(apperr.Field(
			"username", "self", "Tidak bisa mengikuti diri sendiri.",
		))
	}
	return target, nil
}

// Endorse records the viewer's endorsement of a skill on the target profile.
func (s *Service) Endorse(ctx context.Context, p *auth.Principal, username, skillSlug string) error {
	target, err := s.endorsementTarget(ctx, p, username, skillSlug)
	if err != nil {
		return err
	}
	if err := s.st.InsertSkillEndorsement(ctx, store.InsertSkillEndorsementParams{
		EndorserID: p.UserID,
		UserID:     target.ID,
		Slug:       skillSlug,
	}); err != nil {
		return fmt.Errorf("profiles: endorse skill: %w", err)
	}
	return nil
}

// Unendorse removes the viewer's endorsement of a skill on the target profile.
func (s *Service) Unendorse(ctx context.Context, p *auth.Principal, username, skillSlug string) error {
	target, err := s.endorsementTarget(ctx, p, username, skillSlug)
	if err != nil {
		return err
	}
	if err := s.st.DeleteSkillEndorsement(ctx, store.DeleteSkillEndorsementParams{
		EndorserID: p.UserID,
		UserID:     target.ID,
		Slug:       skillSlug,
	}); err != nil {
		return fmt.Errorf("profiles: unendorse skill: %w", err)
	}
	return nil
}

func (s *Service) endorsementTarget(ctx context.Context, p *auth.Principal, username, skillSlug string) (profileUser, error) {
	if p == nil {
		return profileUser{}, apperr.Unauthenticated()
	}
	target, err := s.resolveVisible(ctx, username)
	if err != nil {
		return profileUser{}, err
	}
	if target.ID == p.UserID {
		return profileUser{}, apperr.Validation(apperr.Field(
			"username", "self", "Tidak bisa mengendorse diri sendiri.",
		))
	}
	if _, err := s.st.GetUserSkill(ctx, store.GetUserSkillParams{
		UserID: target.ID,
		Slug:   skillSlug,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profileUser{}, apperr.NotFound()
		}
		return profileUser{}, fmt.Errorf("profiles: load target skill: %w", err)
	}
	return target, nil
}

// ListFollowers returns one page of users following username.
func (s *Service) ListFollowers(ctx context.Context, username, cursor string, limit int) (UserMiniPage, error) {
	target, err := s.resolveVisible(ctx, username)
	if err != nil {
		return UserMiniPage{}, err
	}
	limit = normalizedLimit(limit)

	cursorTime, cursorID, err := decodeCursor(cursor)
	if err != nil {
		return UserMiniPage{}, apperr.BadRequest("cursor tidak valid")
	}

	rows, err := s.st.ListProfileFollowers(ctx, store.ListProfileFollowersParams{
		UserID:     target.ID,
		CursorTime: cursorTime,
		CursorID:   cursorID,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		return UserMiniPage{}, fmt.Errorf("profiles: list followers: %w", err)
	}

	page := UserMiniPage{Items: make([]UserMini, 0, min(len(rows), limit))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	for _, row := range rows {
		page.Items = append(page.Items, UserMini{
			Username:      row.Username,
			DisplayName:   row.DisplayName,
			AccountType:   row.AccountType,
			AvatarMediaID: row.AvatarMediaID,
		})
	}
	return page, nil
}

// ListFollowing returns one page of users username follows.
func (s *Service) ListFollowing(ctx context.Context, username, cursor string, limit int) (UserMiniPage, error) {
	target, err := s.resolveVisible(ctx, username)
	if err != nil {
		return UserMiniPage{}, err
	}
	limit = normalizedLimit(limit)

	cursorTime, cursorID, err := decodeCursor(cursor)
	if err != nil {
		return UserMiniPage{}, apperr.BadRequest("cursor tidak valid")
	}

	rows, err := s.st.ListProfileFollowing(ctx, store.ListProfileFollowingParams{
		UserID:     target.ID,
		CursorTime: cursorTime,
		CursorID:   cursorID,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		return UserMiniPage{}, fmt.Errorf("profiles: list following: %w", err)
	}

	page := UserMiniPage{Items: make([]UserMini, 0, min(len(rows), limit))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	for _, row := range rows {
		page.Items = append(page.Items, UserMini{
			Username:      row.Username,
			DisplayName:   row.DisplayName,
			AccountType:   row.AccountType,
			AvatarMediaID: row.AvatarMediaID,
		})
	}
	return page, nil
}

func normalizedLimit(limit int) int {
	if limit <= 0 {
		return defaultPageLimit
	}
	if limit > maxPageLimit {
		return maxPageLimit
	}
	return limit
}
