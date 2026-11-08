package profiles

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type cursorPayload struct {
	Time string `json:"t"`
	ID   string `json:"id"`
}

// encodeCursor builds an opaque pagination cursor from a follow timestamp and
// the user ID that breaks ties.
func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw, err := json.Marshal(cursorPayload{
		Time: createdAt.UTC().Format(time.RFC3339Nano),
		ID:   id.String(),
	})
	if err != nil {
		// Marshalling a struct of strings cannot fail.
		panic("profiles: encode cursor: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// decodeCursor returns nil values for an empty cursor.
func decodeCursor(value string) (*time.Time, *uuid.UUID, error) {
	if value == "" {
		return nil, nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, nil, err
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.Time)
	if err != nil {
		return nil, nil, err
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil || id == uuid.Nil {
		return nil, nil, fmt.Errorf("profiles: invalid cursor id")
	}
	return &createdAt, &id, nil
}
