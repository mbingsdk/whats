// Package inbox owns tenant-scoped conversations and guarded text intents.
package inbox

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"strings"
	"time"
	"waba.local/control/internal/identity"
	"waba.local/control/internal/meta"
)

type Error string

func (e Error) Error() string      { return string(e) }
func (e Error) DomainCode() string { return string(e) }
func newID() uuid.UUID {
	v, e := uuid.NewV7()
	if e != nil {
		panic("entropy unavailable")
	}
	return v
}
func hash(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func encode(v any) []byte  { b, _ := json.Marshal(v); return b }
func token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic("entropy unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

type LiveConfig struct {
	Enabled             bool
	Recipient           string
	DeliveryBound       time.Duration
	AcceptanceNotBefore time.Time
}
type TextSender interface {
	SendText(context.Context, string, string, string, string) meta.SendOutcome
}

type Service struct {
	Auth   *identity.Service
	Meta   *meta.Service
	Live   LiveConfig
	Sender TextSender
}

func New(auth *identity.Service, m *meta.Service, live LiveConfig) *Service {
	s := &Service{Auth: auth, Meta: m, Live: live}
	if m != nil {
		s.Sender = m.Graph
	}
	return s
}
func require(ctx context.Context, tx pgx.Tx, member uuid.UUID, key string, team, subject *uuid.UUID) error {
	var ok bool
	if e := tx.QueryRow(ctx, "SELECT app.inbox_permission($1,$2,$3,$4)", member, key, team, subject).Scan(&ok); e != nil {
		return e
	}
	if !ok {
		return identity.ErrPermission
	}
	return nil
}
func rows(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]map[string]any, error) {
	r, e := tx.Query(ctx, sql, args...)
	if e != nil {
		return nil, e
	}
	v, e := pgx.CollectRows(r, pgx.RowToMap)
	if v == nil {
		v = []map[string]any{}
	}
	for _, item := range v {
		for k, value := range item {
			if id, ok := value.([16]byte); ok {
				item[k] = uuid.UUID(id).String()
			}
		}
	}
	return v, e
}
func audit(ctx context.Context, tx pgx.Tx, org, user, id uuid.UUID, action string) error {
	_, e := tx.Exec(ctx, "INSERT INTO app.audit_log(id,organization_id,actor_user_id,actor_kind,action,resource_id,request_id) VALUES($1,$2,$3,'USER',$4,$5,'inbox')", newID(), org, user, action, id)
	return e
}
func event(ctx context.Context, tx pgx.Tx, org, conversation uuid.UUID, kind string) error {
	_, e := tx.Exec(ctx, "INSERT INTO app.inbox_events(id,organization_id,conversation_id,event_type) VALUES($1,$2,$3,$4)", newID(), org, conversation, kind)
	return e
}

type Conversation struct {
	ID, Org, Phone, Recipient                                                          uuid.UUID
	LastEligible, Team, Member                                                         *uuid.UUID
	Revision, Assignment, HandoffEpoch, InboundSequence                                int64
	Handoff, Status, Identity, PhoneExternal, WABAExternal, Timezone, WindowConfidence string
	Window                                                                             *time.Time
	Suppressed                                                                         bool
}

func conversation(ctx context.Context, tx pgx.Tx, id, member uuid.UUID, key string, lock bool) (Conversation, error) {
	var c Conversation
	suffix := ""
	if lock {
		// Stable recipient and sender anchors precede the conversation lock.
		// Suppression/topology changes must conflict with final authorization.
		if _, e := tx.Exec(ctx, "SELECT r.id FROM app.recipient_identities r JOIN app.conversations c ON c.organization_id=r.organization_id AND c.recipient_id=r.id JOIN app.phone_numbers p ON p.organization_id=c.organization_id AND p.id=c.phone_id JOIN app.wabas w ON w.organization_id=p.organization_id AND w.id=p.waba_id WHERE c.id=$1 AND app.inbox_permission($2,'inbox.view',c.assigned_team_id,c.assigned_member_id) FOR SHARE OF r,p,w", id, member); e != nil {
			return c, e
		}
		suffix = " FOR UPDATE OF c"
	}
	e := tx.QueryRow(ctx, "SELECT c.id,c.organization_id,c.phone_id,c.recipient_id,c.last_eligible_inbound_message_id,c.assigned_team_id,c.assigned_member_id,c.revision,c.assignment_revision,c.handoff_epoch,c.inbound_sequence,c.handoff_state,c.status,r.identity_value,p.external_id,w.external_id,w.timezone_id,c.window_confidence,c.window_expires_at,r.suppressed FROM app.conversations c JOIN app.recipient_identities r ON r.organization_id=c.organization_id AND r.id=c.recipient_id JOIN app.phone_numbers p ON p.organization_id=c.organization_id AND p.id=c.phone_id JOIN app.wabas w ON w.organization_id=p.organization_id AND w.id=p.waba_id WHERE c.id=$1 AND app.inbox_permission($2,'inbox.view',c.assigned_team_id,c.assigned_member_id)"+suffix, id, member).Scan(&c.ID, &c.Org, &c.Phone, &c.Recipient, &c.LastEligible, &c.Team, &c.Member, &c.Revision, &c.Assignment, &c.HandoffEpoch, &c.InboundSequence, &c.Handoff, &c.Status, &c.Identity, &c.PhoneExternal, &c.WABAExternal, &c.Timezone, &c.WindowConfidence, &c.Window, &c.Suppressed)
	if errors.Is(e, pgx.ErrNoRows) {
		e = identity.ErrNotFound
	}
	if e != nil {
		return c, e
	}
	if key != "" {
		e = require(ctx, tx, member, key, c.Team, c.Member)
	}
	return c, e
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 262145))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return identity.ErrValidation
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return identity.ErrValidation
	}
	return nil
}
func pathID(r *http.Request) (uuid.UUID, error) {
	v, e := uuid.Parse(r.PathValue("id"))
	if e != nil {
		return v, identity.ErrValidation
	}
	return v, nil
}
func validText(v string, max int) bool {
	return strings.TrimSpace(v) != "" && len([]rune(v)) <= max && !strings.ContainsRune(v, 0)
}

type route struct {
	Method, Path string
	Sensitive    bool
}

var Routes = []route{
	{"GET", "/inbox", false}, {"GET", "/conversations", false}, {"GET", "/conversations/{id}", false},
	{"PATCH", "/conversations/{id}", false}, {"PUT", "/conversations/{id}/assignment", false},
	{"POST", "/conversations/{id}/read", false}, {"POST", "/conversations/{id}/notes", false},
	{"PATCH", "/notes/{id}", false}, {"POST", "/notes/{id}/redact", true},
	{"POST", "/conversations/{id}/presence", false}, {"GET", "/inbox/events", false},
	{"POST", "/pricing/preflight", false}, {"POST", "/outbound-intents", false}, {"GET", "/outbound-intents", false}, {"GET", "/outbound-intents/{id}", false},
	{"GET", "/pricing", false}, {"POST", "/rate-cards/imports", true}, {"POST", "/rate-cards/imports/{id}/{action}", true},
	{"GET", "/budgets", false}, {"POST", "/budgets", true}, {"PATCH", "/inbox/settings", true},
}

func (s *Service) Handler(fallback http.Handler) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range Routes {
		mux.HandleFunc(rt.Method+" /api/v1"+rt.Path, func(w http.ResponseWriter, r *http.Request) {
			if rt.Path == "/inbox/events" {
				s.stream(w, r)
				return
			}
			data, e := s.Auth.DomainTransaction(r, rt.Sensitive, func(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID) (any, error) {
				return s.operation(ctx, tx, v, member, r, rt)
			})
			identity.WriteDomainResult(w, r, data, e)
		})
	}
	mux.Handle("/", fallback)
	return mux
}
