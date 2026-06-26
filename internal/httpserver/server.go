package httpserver

import (
	"log/slog"
	"net/http"

	"echo-mcp/internal/state"
)

// Server owns the local HTTP data-plane handler.
type Server struct {
	handler http.Handler
	logger  *slog.Logger
	store   *state.Store
}

// New creates the local HTTP data-plane server skeleton.
func New(store *state.Store, logger *slog.Logger) *Server {
	if store == nil {
		store = state.New()
	}
	if logger == nil {
		logger = slog.Default()
	}

	server := &Server{
		logger: logger,
		store:  store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleDataPlane)
	server.handler = mux

	return server
}

// Handler returns the HTTP handler for local serving and tests.
func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) handleDataPlane(w http.ResponseWriter, r *http.Request) {
	s.logger.Info(
		"data-plane request received",
		"method", r.Method,
		"path", r.URL.Path,
		"state_generation", s.store.Generation(),
	)

	if rule, ok := s.store.MatchResponseRule(r.Method, r.URL.Path); ok {
		if rule.ContentType != "" {
			w.Header().Set("Content-Type", rule.ContentType)
		}
		w.WriteHeader(rule.StatusCode)
		if _, err := w.Write([]byte(rule.Body)); err != nil {
			s.logger.Error("write configured response", "error", err)
		}
		s.store.RecordObservation(state.Observation{
			RequestMethod:     r.Method,
			RequestPath:       r.URL.Path,
			MatchedRuleID:     rule.ID,
			OutcomeStatusCode: rule.StatusCode,
		})
		return
	}

	http.Error(w, "data-plane behavior matching is not implemented", http.StatusNotImplemented)
}
