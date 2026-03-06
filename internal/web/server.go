package web

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dbehnke/trindex/internal/auth"
	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	embedclient "github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/memory"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

var (
	//go:embed all:dist
	distFS embed.FS
)

// Server provides HTTP interface for Trindex
type Server struct {
	cfg        *config.Config
	store      *memory.Store
	auth       *auth.Service
	httpServer *http.Server
	router     chi.Router
}

// NewServer creates a new web server
func NewServer(cfg *config.Config, database *db.DB, embedClient *embedclient.Client) *Server {
	store := memory.NewStore(database, embedClient, cfg)
	authSvc := auth.NewService(database)

	s := &Server{
		cfg:   cfg,
		store: store,
		auth:  authSvc,
	}

	s.setupRouter()

	s.httpServer = &http.Server{
		Addr:    cfg.HTTPHost + ":" + cfg.HTTPPort,
		Handler: s.router,
	}

	return s
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			slog.Debug("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
		})
	})

	r.Get("/health", s.handleHealth)

	r.Route("/api", func(r chi.Router) {
		r.Use(s.apiKeyMiddleware)

		r.Route("/memories", func(r chi.Router) {
			r.Get("/", s.handleListMemories)
			r.Post("/", s.handleCreateMemory)
			r.Get("/{id}", s.handleGetMemory)
			r.Delete("/{id}", s.handleDeleteMemory)
		})

		r.Route("/keys", func(r chi.Router) {
			r.Get("/", s.handleListKeys)
			r.Post("/", s.handleCreateKey)
			r.Delete("/{id}", s.handleRevokeKey)
		})

		r.Post("/search", s.handleSearch)
		r.Get("/stats", s.handleStats)

		r.Get("/export", s.handleExport)
		r.Post("/import", s.handleImport)
		r.Get("/duplicates", s.handleFindDuplicates)
		r.Post("/duplicates/merge", s.handleMergeDuplicates)

		r.Get("/mcp/tools", s.handleMCPTools)
		r.Post("/mcp/call", s.handleMCPCall)
	})

	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		slog.Error("failed to create static filesystem", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(staticFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			filepath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
			if filepath == "" {
				filepath = "index.html"
			}

			_, err := staticFS.Open(filepath)
			if err != nil {
				r.URL.Path = "/"
			}

			fileServer.ServeHTTP(w, r)
		})
	}

	s.router = r
}

// Run starts the HTTP server
func (s *Server) Run(ctx context.Context) error {
	slog.Info("starting web server", "addr", s.httpServer.Addr)

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down web server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	order := r.URL.Query().Get("order")
	if order == "" {
		order = "desc"
	}

	params := memory.ListParams{
		Namespace: r.URL.Query().Get("namespace"),
		Limit:     limit,
		Offset:    offset,
		Order:     order,
	}

	memories, err := s.store.List(r.Context(), params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, memories)
}

func (s *Server) handleGetMemory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "memory ID required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memory ID")
		return
	}

	mem, err := s.store.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, mem)
}

func (s *Server) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content   string                 `json:"content"`
		Namespace string                 `json:"namespace"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Content == "" {
		respondError(w, http.StatusBadRequest, "content is required")
		return
	}

	mem, err := s.store.Create(r.Context(), req.Content, req.Namespace, req.Metadata)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.auth.LogAction(getAPIKeyID(r.Context()), "MEMORY_CREATE", req.Namespace, map[string]interface{}{"memory_id": mem.ID.String()})

	respondJSON(w, http.StatusCreated, mem)
}

func (s *Server) handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "memory ID required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memory ID")
		return
	}

	if err := s.store.DeleteByID(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// For deletion, namespace isn't explicitly passed in the URL, so we log as wild or rely on the UI sending it.
	s.auth.LogAction(getAPIKeyID(r.Context()), "MEMORY_DELETE", "unknown", map[string]interface{}{"memory_id": id.String()})

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string        `json:"query"`
		Namespaces []string      `json:"namespaces"`
		TopK       int           `json:"top_k"`
		Threshold  float64       `json:"threshold"`
		Filter     memory.Filter `json:"filter"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Query == "" {
		respondError(w, http.StatusBadRequest, "query is required")
		return
	}

	if req.TopK == 0 {
		req.TopK = s.cfg.DefaultTopK
	}
	if req.Threshold == 0 {
		req.Threshold = s.cfg.DefaultSimilarityThreshold
	}

	params := memory.RecallParams{
		Query:      req.Query,
		Namespaces: req.Namespaces,
		TopK:       req.TopK,
		Threshold:  req.Threshold,
		Filter:     req.Filter,
	}

	results, err := s.store.Recall(r.Context(), params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.auth.LogAction(getAPIKeyID(r.Context()), "SEARCH", strings.Join(req.Namespaces, ","), map[string]interface{}{
		"query": req.Query,
		"top_k": req.TopK,
		"found": len(results),
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results":             results,
		"total":               len(results),
		"namespaces_searched": append(req.Namespaces, "global"),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	stats, err := s.store.GetStats(r.Context(), namespace)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	var since, until *time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			since = &t
		}
	}
	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		t, err := time.Parse(time.RFC3339, untilStr)
		if err == nil {
			until = &t
		}
	}

	w.Header().Set("Content-Type", "application/jsonl")
	w.Header().Set("Content-Disposition", "attachment; filename=memories.jsonl")

	result, err := s.store.Export(r.Context(), namespace, since, until, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("export completed", "count", result.Count, "namespace", result.Namespace)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Options memory.ImportOptions `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := s.store.Import(r.Context(), r.Body, req.Options)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleFindDuplicates(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	threshold, _ := strconv.ParseFloat(r.URL.Query().Get("threshold"), 64)
	if threshold == 0 {
		threshold = 0.95
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	candidates, err := s.store.FindDuplicates(r.Context(), namespace, threshold, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"candidates": candidates,
		"total":      len(candidates),
	})
}

func (s *Server) handleMergeDuplicates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		KeepID   string `json:"keep_id"`
		RemoveID string `json:"remove_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	keepID, err := uuid.Parse(req.KeepID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid keep_id")
		return
	}

	removeID, err := uuid.Parse(req.RemoveID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid remove_id")
		return
	}

	if err := s.store.MergeDuplicates(r.Context(), keepID, removeID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "merged"})
}

// contextKey is used for strongly typed context values
type contextKey string

const (
	apiKeyIDContextKey contextKey = "api_key_id"
)

func (s *Server) apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no master key is set and there are no keys in DB context,
		// we still want to block unauthorized access, but we'll let
		// the DB validation handle it below if a key is provided.

		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			respondError(w, http.StatusUnauthorized, "missing API key")
			return
		}

		ctx := r.Context()

		// First check against Master Key
		if s.cfg.HTTPAPIKey != "" && apiKey == s.cfg.HTTPAPIKey {
			// Master key used
			next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, apiKeyIDContextKey, nil)))
			return
		}

		// Fallback to database API Keys
		keyID, valid, err := s.auth.ValidateKey(ctx, apiKey)
		if err != nil {
			slog.Error("failed to validate api key", "error", err)
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if !valid {
			respondError(w, http.StatusUnauthorized, "invalid or revoked API key")
			return
		}

		// Inject key ID into context for auditing
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, apiKeyIDContextKey, keyID)))
	})
}

// getAPIKeyID extracts the authenticated Key ID from the context (nil if Master Key)
func getAPIKeyID(ctx context.Context) *uuid.UUID {
	val := ctx.Value(apiKeyIDContextKey)
	if id, ok := val.(*uuid.UUID); ok {
		return id
	}
	return nil
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.auth.ListKeys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	key, rawSecret, err := s.auth.CreateKey(r.Context(), req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.auth.LogAction(getAPIKeyID(r.Context()), "API_KEY_CREATE", "system", map[string]interface{}{"created_key_id": key.ID.String()})

	// Only endpoint that returns the raw secret
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"key":    key,
		"secret": rawSecret,
	})
}

func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "key ID required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid key ID")
		return
	}

	if err := s.auth.RevokeKey(r.Context(), id); err != nil {
		if err.Error() == "API key not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.auth.LogAction(getAPIKeyID(r.Context()), "API_KEY_REVOKE", "system", map[string]interface{}{"revoked_key_id": id.String()})

	respondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
