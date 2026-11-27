package messaging

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// ErrInvalidCursor is returned by List for a malformed cursor or one issued for other filters.
var ErrInvalidCursor = errors.New("invalid cursor")

// ListFilter selects messages for List. Nil fields do not filter.
type ListFilter struct {
	ClientReference *string
	Status          *Status
	AcceptedAfter   *time.Time // inclusive
	AcceptedBefore  *time.Time // exclusive
	// Limit is the page size (the HTTP layer enforces the contract's 1 to 100).
	Limit int
	// Cursor is the opaque next_cursor of a previous page, or "".
	Cursor string
}

// cursor is the position after the last message of a page. It also carries a hash of the filters
// it was issued for, so that it cannot silently be reused with other filters.
type cursor struct {
	AcceptedAt time.Time `json:"a"`
	ID         string    `json:"i"`
	Filters    string    `json:"f"`
}

// List returns one page of tenantID's messages, newest acceptance first, and the cursor of the next
// page ("" when there are no more).
func (s *Service) List(ctx context.Context, tenantID tenancy.ID, f ListFilter) ([]Message, string, error) {
	params := sqlcgen.ListMessagesParams{
		TenantID:        pgUUID(tenantID),
		ClientReference: f.ClientReference,
		AcceptedAfter:   f.AcceptedAfter,
		AcceptedBefore:  f.AcceptedBefore,
		PageLimit:       int32(f.Limit) + 1, //nolint:gosec // G115: the HTTP layer bounds Limit to 1..100.
	}
	if f.Status != nil {
		params.States = []string{} // an unknown status matches nothing
		for _, st := range f.Status.States() {
			params.States = append(params.States, string(st))
		}
	}
	filters := filterHash(f)
	if f.Cursor != "" {
		c, err := decodeCursor(f.Cursor)
		if err != nil || c.Filters != filters {
			return nil, "", ErrInvalidCursor
		}
		id, err := hex.DecodeString(c.ID)
		if err != nil || len(id) != 16 {
			return nil, "", ErrInvalidCursor
		}
		at := c.AcceptedAt
		params.CursorAcceptedAt = &at
		params.CursorID = pgtype.UUID{Bytes: [16]byte(id), Valid: true}
	}

	var rows []sqlcgen.ListMessagesRow
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListMessages(ctx, params)
		return err
	})
	if err != nil {
		return nil, "", fmt.Errorf("list messages: %w", err)
	}
	next := ""
	if len(rows) > f.Limit {
		rows = rows[:f.Limit]
		last := rows[len(rows)-1]
		next = encodeCursor(cursor{AcceptedAt: last.AcceptedAt, ID: hex.EncodeToString(last.ID.Bytes[:]), Filters: filters})
	}
	messages := make([]Message, len(rows))
	for i, r := range rows {
		messages[i] = messageFromRow(sqlcgen.GetMessageRow(r))
	}
	return messages, next, nil
}

func filterHash(f ListFilter) string {
	b, _ := json.Marshal([]any{f.ClientReference, f.Status, f.AcceptedAfter, f.AcceptedBefore}) // cannot fail
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c) // cannot fail
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursor, error) {
	var c cursor
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, err
	}
	return c, nil
}
