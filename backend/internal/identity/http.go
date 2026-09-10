package identity

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Route struct{ Method, Path, Action string }

var Routes = []Route{
	{"POST", "/auth/login", "public.login"}, {"POST", "/auth/logout", "logout"}, {"GET", "/auth/session", "session"},
	{"POST", "/auth/mfa/verify", "public.mfa"}, {"POST", "/auth/reauthenticate", "reauth"},
	{"POST", "/auth/password-reset/request", "public.reset.request"}, {"POST", "/auth/password-reset/complete", "public.reset.complete"},
	{"POST", "/auth/email-verification/request", "public.verify.request"}, {"POST", "/auth/email-verification/complete", "public.verify.complete"},
	{"POST", "/auth/organization-session", "organization.select"},
	{"POST", "/auth/mfa/enrollment", "mfa.enroll"}, {"POST", "/auth/mfa/enrollment/confirm", "mfa.confirm"},
	{"POST", "/auth/mfa/disable", "mfa.disable"}, {"POST", "/auth/mfa/recovery-codes/regenerate", "mfa.recovery"},
	{"GET", "/security/sessions", "sessions"}, {"POST", "/security/sessions/{id}/revoke", "session.revoke"}, {"POST", "/security/sessions/revoke-others", "sessions.revoke"},
	{"POST", "/invitations/accept", "public.invitation.accept"}, {"GET", "/organization", "org.get"},
	{"GET", "/organizations/{org}/members", "org.members.list"}, {"GET", "/organizations/{org}/members/{id}", "org.member.get"},
	{"POST", "/organizations/{org}/members/{id}/deactivate", "org.member.deactivate"}, {"POST", "/organizations/{org}/members/{id}/reactivate", "org.member.reactivate"},
	{"PUT", "/organizations/{org}/members/{id}/role-grants", "org.member.roles"},
	{"GET", "/organizations/{org}/invitations", "org.invitations.list"}, {"POST", "/organizations/{org}/invitations", "org.invite"},
	{"POST", "/organizations/{org}/invitations/{id}/revoke", "org.invite.revoke"}, {"POST", "/organizations/{org}/invitations/{id}/resend", "org.invite.resend"},
	{"GET", "/organizations/{org}/teams", "org.teams.list"}, {"POST", "/organizations/{org}/teams", "org.team.create"},
	{"PATCH", "/organizations/{org}/teams/{id}", "org.team.update"}, {"POST", "/organizations/{org}/teams/{id}/archive", "org.team.archive"},
	{"GET", "/organizations/{org}/teams/{id}/members", "org.team.members.list"},
	{"POST", "/organizations/{org}/teams/{id}/members", "org.team.member.add"}, {"DELETE", "/organizations/{org}/teams/{id}/members/{member}", "org.team.member.remove"},
	{"PUT", "/organizations/{org}/teams/{id}/role-grants", "org.team.roles"},
	{"GET", "/organizations/{org}/roles", "org.roles.list"}, {"POST", "/organizations/{org}/roles", "org.role.create"}, {"PATCH", "/organizations/{org}/roles/{id}", "org.role.update"},
	{"GET", "/organizations/{org}/audit", "org.audit.list"},
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, r *http.Request, e error) {
	code, status := "DEPENDENCY_UNAVAILABLE", 503
	pairs := []struct {
		e      error
		status int
	}{{ErrUnauthenticated, 401}, {ErrAuthentication, 401}, {ErrPermission, 403}, {ErrReauth, 403}, {ErrToken, 400}, {ErrValidation, 422}, {ErrConflict, 409}, {ErrOwner, 409}, {ErrNotFound, 404}, {ErrRate, 429}}
	for _, p := range pairs {
		if errors.Is(e, p.e) {
			code, status = p.e.Error(), p.status
			break
		}
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && (pg.Code == "23505" || pg.Code == "23503") {
		code, status = "RESOURCE_CONFLICT", 409
	}
	if status == 429 {
		w.Header().Set("Retry-After", "900")
	}
	jsonResponse(w, status, map[string]any{"error": map[string]any{"code": code, "message": strings.ReplaceAll(strings.ToLower(code), "_", " "), "request_id": w.Header().Get("X-Request-ID")}})
}
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/auth/csrf", func(w http.ResponseWriter, r *http.Request) {
		value := ""
		if c, e := r.Cookie("waba_csrf"); e == nil && len(c.Value) == 43 {
			value = c.Value
		}
		if value == "" {
			value = randomToken()
		}
		s.setCookie(w, "waba_csrf", value, false)
		jsonResponse(w, 200, map[string]string{"csrf_token": value})
	})
	for _, route := range Routes {
		mux.HandleFunc(route.Method+" /api/v1"+route.Path, func(w http.ResponseWriter, r *http.Request) {
			var in Input
			in.Cursor = r.URL.Query().Get("cursor")
			if len(in.Cursor) > 128 {
				fail(w, r, ErrValidation)
				return
			}
			if r.Method != "GET" {
				csrf := r.Header.Get("X-CSRF-Token")
				cookie, e := r.Cookie("waba_csrf")
				if r.Header.Get("Origin") != s.Origin || e != nil || len(csrf) != 43 || !csrfEqual(csrf, cookie.Value) {
					fail(w, r, ErrPermission)
					return
				}
				if r.Body != nil && r.ContentLength != 0 {
					decoder := json.NewDecoder(r.Body)
					decoder.DisallowUnknownFields()
					if e := decoder.Decode(&in); e != nil {
						var tooLarge *http.MaxBytesError
						if errors.As(e, &tooLarge) {
							jsonResponse(w, 413, map[string]any{"error": map[string]string{"code": "REQUEST_TOO_LARGE", "message": "Request body exceeds the limit.", "request_id": w.Header().Get("X-Request-ID")}})
							return
						}
						fail(w, r, ErrValidation)
						return
					}
					var extra any
					if decoder.Decode(&extra) != io.EOF {
						fail(w, r, ErrValidation)
						return
					}
				}
			}
			for _, path := range []struct {
				name string
				dst  *uuid.UUID
			}{{"org", &in.OrganizationID}, {"id", &in.ID}, {"member", &in.MemberID}} {
				if value := r.PathValue(path.name); value != "" {
					parsed, e := uuid.Parse(value)
					if e != nil {
						fail(w, r, ErrNotFound)
						return
					}
					*path.dst = parsed
				}
			}
			if in.ID != uuid.Nil && r.Method != "GET" && strings.HasPrefix(route.Action, "org.") {
				value := strings.Trim(r.Header.Get("If-Match"), "\"")
				n, e := strconv.ParseInt(value, 10, 64)
				if e != nil || n < 1 {
					jsonResponse(w, 428, map[string]any{"error": map[string]string{"code": "PRECONDITION_REQUIRED", "message": "Supply the current revision.", "request_id": w.Header().Get("X-Request-ID")}})
					return
				}
				in.Revision = n
			}
			token := ""
			if cookie, e := r.Cookie("waba_session"); e == nil {
				token = cookie.Value
			}
			ip, _, _ := net.SplitHostPort(r.RemoteAddr) // Forwarded headers are deliberately not trusted.
			result, e := s.Run(r.Context(), route.Action, token, in, Meta{RequestID: w.Header().Get("X-Request-ID"), IP: ip, CSRF: r.Header.Get("X-CSRF-Token")})
			if e != nil {
				fail(w, r, e)
				return
			}
			if result.Cookie == "clear" {
				http.SetCookie(w, &http.Cookie{Name: "waba_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.Production, SameSite: http.SameSiteLaxMode})
			} else if result.Cookie != "" {
				s.setCookie(w, "waba_session", result.Cookie, true)
				s.setCookie(w, "waba_csrf", result.CSRF, false)
			}
			status := 200
			if route.Action == "public.reset.request" || route.Action == "public.verify.request" || route.Action == "org.invite" {
				status = 202
			}
			if route.Action == "org.team.create" || route.Action == "org.role.create" {
				status = 201
			}
			jsonResponse(w, status, result.Data)
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fail(w, r, ErrNotFound) })
	return mux
}
func (s *Service) setCookie(w http.ResponseWriter, name, value string, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: httpOnly, Secure: s.Production, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 60 * 60})
}
