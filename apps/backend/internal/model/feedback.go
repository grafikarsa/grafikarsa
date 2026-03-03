package model

import (
	"database/sql"
	"time"
)

type Feedback struct {
	ID         uint64         `db:"id"`
	UserID     sql.NullInt64  `db:"user_id"`
	Nama       sql.NullString `db:"nama"`
	Email      sql.NullString `db:"email"`
	Kategori   string         `db:"kategori"`
	Pesan      string         `db:"pesan"`
	Status     string         `db:"status"`
	AdminNotes sql.NullString `db:"admin_notes"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
}

type FeedbackWithUser struct {
	Feedback
	UserNama     sql.NullString `db:"user_nama"`
	UserUsername sql.NullString `db:"user_username"`
	UserAvatar   sql.NullString `db:"user_avatar"`
}
