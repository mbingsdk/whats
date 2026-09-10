package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strings"
)

// Cursors bind position to the authenticated subject, organization and resource.
// Every page repeats authorization and RLS; a cursor never conveys access.
func (s *Service) page(ctx context.Context, tx pgx.Tx, in Input, action string, user, org uuid.UUID, query string, args ...any) (Result, error) {
	binding := user.String() + ":" + org.String() + ":" + action + ":" + in.ID.String() + ":"
	sign := func(position string) string {
		mac := hmac.New(sha256.New, s.Root)
		_, _ = mac.Write([]byte("identity-page:" + binding + position))
		return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	}
	var after *uuid.UUID
	if in.Cursor != "" {
		parts := strings.Split(in.Cursor, ".")
		if len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(sign(parts[0]))) {
			return Result{}, ErrValidation
		}
		value, e := uuid.Parse(parts[0])
		if e != nil {
			return Result{}, ErrValidation
		}
		after = &value
	}
	// Queries and sort columns are internal constants, never user input.
	if index := strings.LastIndex(query, " ORDER BY "); index >= 0 {
		query = query[:index]
	}
	direction, operator := "ASC", ">"
	if action == "org.audit.list" || action == "org.invitations.list" {
		direction, operator = "DESC", "<"
	}
	args = append(args, after)
	n := len(args)
	query = fmt.Sprintf("SELECT * FROM (%s) AS page WHERE ($%d::uuid IS NULL OR id %s $%d) ORDER BY id %s LIMIT 101", query, n, operator, n, direction)
	rows, e := tx.Query(ctx, query, args...)
	if e != nil {
		return Result{}, e
	}
	items, e := pgx.CollectRows(rows, pgx.RowToMap)
	if e != nil {
		return Result{}, e
	}
	items = jsonRows(items)
	more := len(items) > 100
	var cursor any
	if more {
		items = items[:100]
		position := items[len(items)-1]["id"].(string)
		cursor = position + "." + sign(position)
	}
	return stringResult(map[string]any{"items": items, "has_more": more, "next_cursor": cursor}), nil
}
