package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"waba.local/control/internal/config"
)

type requestKey struct{}
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func requestID(r *http.Request) string { id, _ := r.Context().Value(requestKey{}).(string); return id }
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, ErrorEnvelope{APIError{code, message, requestID(r)}})
}

// Readiness is a function seam for deterministic outage tests; the production probe uses real PostgreSQL.
func Handler(c config.Config, logger *slog.Logger, ready func(context.Context) error, identityHandlers ...http.Handler) http.Handler {
	mux := http.NewServeMux()
	if len(identityHandlers) > 0 {
		mux.Handle("/api/", identityHandlers[0])
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			writeError(w, r, 405, "METHOD_NOT_ALLOWED", "Method not allowed.")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			writeError(w, r, 405, "METHOD_NOT_ALLOWED", "Method not allowed.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := ready(ctx); err != nil {
			writeError(w, r, 503, "DEPENDENCY_UNAVAILABLE", "Service is not ready.")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, 404, "RESOURCE_NOT_FOUND", "Resource not found.")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get("X-Request-ID")
		if !requestIDPattern.MatchString(id) {
			value, err := uuid.NewV7()
			if err != nil {
				http.Error(w, "Request unavailable", 503)
				return
			}
			id = value.String()
		}
		r = r.WithContext(context.WithValue(r.Context(), requestKey{}, id))
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		if c.Environment == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		rec := &response{ResponseWriter: w, status: 200}
		defer func() {
			if recover() != nil {
				if !rec.wrote {
					writeError(rec, r, 500, "INTERNAL_ERROR", "Unexpected server error.")
				}
				logger.Error("request panic", slog.String("request_id", id))
			}
			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			logger.Info("http request", slog.String("request_id", id), slog.String("method", r.Method), slog.String("route", route), slog.Int("status", rec.status), slog.Duration("duration", time.Since(start)))
		}()
		if r.ContentLength > c.MaxBodyBytes {
			writeError(rec, r, 413, "REQUEST_TOO_LARGE", "Request body exceeds the configured limit.")
			return
		}
		r.Body = http.MaxBytesReader(rec, r.Body, c.MaxBodyBytes)
		mux.ServeHTTP(rec, r)
	})
}

type response struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *response) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.status = status
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}
func (w *response) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}

func Server(c config.Config, handler http.Handler) *http.Server {
	return &http.Server{Addr: c.Listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: c.ReadTimeout, WriteTimeout: c.WriteTimeout, IdleTimeout: c.IdleTimeout, MaxHeaderBytes: c.MaxHeaderBytes}
}
func Serve(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		drain, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := server.Shutdown(drain)
		if err != nil {
			_ = server.Close()
		}
		serveErr := <-result
		if err != nil {
			return err
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return nil
	}
}
