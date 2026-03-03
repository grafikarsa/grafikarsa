package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateCommentRequest struct {
	PortfolioID uuid.UUID  `json:"portfolio_id" validate:"required"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Content     string     `json:"content" validate:"required,min=1"`
}

type CommentResponse struct {
	ID        uuid.UUID          `json:"id"`
	Content   string             `json:"content"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	User      UserBriefDTO       `json:"user"`
	Children  []*CommentResponse `json:"children,omitempty"`
}
