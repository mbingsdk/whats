package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
	"waba.local/control/internal/identity"
)

func (s *Service) stream(w http.ResponseWriter, r *http.Request) {
	var originalOrg uuid.UUID
	fetch := func() (any, error) {
		return s.Auth.DomainTransaction(r, false, func(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID) (any, error) {
			if originalOrg == uuid.Nil {
				originalOrg = *v.OrganizationID
			} else if originalOrg != *v.OrganizationID {
				return nil, identity.ErrUnauthenticated
			}
			return rows(ctx, tx, `SELECT e.id,e.conversation_id,e.event_type FROM app.inbox_events e JOIN app.conversations c ON c.organization_id=e.organization_id AND c.id=e.conversation_id
 WHERE e.created_at>now()-interval '1 minute' AND app.inbox_permission($1,'inbox.view',c.assigned_team_id,c.assigned_member_id)
 ORDER BY e.created_at,e.id LIMIT 101`, member)
		})
	}
	first, e := fetch()
	if e != nil {
		identity.WriteDomainResult(w, r, nil, e)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		identity.WriteDomainResult(w, r, nil, Error("STREAM_UNAVAILABLE"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	emit := func(value any) bool {
		// Bounded reconnect, refreshed authorization and REST invalidation close gaps
		// caused by commit reordering. Event contents never carry message text.
		if e := controller.SetWriteDeadline(time.Now().Add(10 * time.Second)); e != nil && e != http.ErrNotSupported {
			return false
		}
		data, _ := json.Marshal(map[string]any{"events": value, "refresh_required": true})
		if _, e := fmt.Fprintf(w, "event: inbox.invalidate\ndata: %s\n\n", data); e != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	if !emit(first) {
		return
	}
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	end := time.NewTimer(24 * time.Second)
	defer end.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-end.C:
			return
		case <-ticker.C:
			data, e := fetch()
			if e != nil {
				return
			}
			if !emit(data) {
				return
			}
		}
	}
}
