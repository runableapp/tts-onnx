package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/keith/linux-tts-onnx/internal/apierrors"
	"github.com/keith/linux-tts-onnx/internal/config"
	"github.com/keith/linux-tts-onnx/internal/modelmgr"
	"github.com/keith/linux-tts-onnx/internal/playback"
	"github.com/keith/linux-tts-onnx/internal/synth"
)

type Server struct {
	cfg      config.Config
	started  time.Time
	logger   *slog.Logger
	models   *modelmgr.Manager
	synthSvc *synth.Service
	runtime  string
}

func New(cfg config.Config, logger *slog.Logger, models *modelmgr.Manager, synthSvc *synth.Service, runtime string) *Server {
	return &Server{
		cfg:      cfg,
		started:  time.Now(),
		logger:   logger,
		models:   models,
		synthSvc: synthSvc,
		runtime:  runtime,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/v1/health", s.handleHealth)
	mux.HandleFunc("/v1/capabilities", s.handleCapabilities)
	mux.HandleFunc("/v1/models", s.withAuth(s.handleModels))
	mux.HandleFunc("/v1/models/install", s.withAuth(s.handleInstallModel))
	mux.HandleFunc("/v1/models/", s.withAuth(s.handleDeleteModel))
	mux.HandleFunc("/v1/speak", s.withAuth(s.handleSpeak))
	mux.HandleFunc("/v1/stop", s.withAuth(s.handleStop))
	mux.HandleFunc("/v1/metrics", s.withAuth(s.handleMetrics))
	return withMaxBytes(mux, s.cfg.MaxPayloadBytes)
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	if s.cfg.BearerToken == "" {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			next(w, r)
			return
		}
		want := "Bearer " + s.cfg.BearerToken
		if r.Header.Get("Authorization") != want {
			writeError(w, apierrors.Unauthorized("missing/invalid bearer token"))
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"runtime": s.runtime,
		"uptime":  time.Since(s.started).String(),
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	manifest, err := s.models.List()
	if err != nil {
		writeError(w, apierrors.Internal(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":              "ok",
		"runtime":             s.runtime,
		"languages_supported": installedLangs(manifest),
		"models_loaded":       firstInstalledByLang(manifest),
		"max_text_chars":      s.cfg.MaxTextChars,
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	manifest, err := s.models.List()
	if err != nil {
		writeError(w, apierrors.Internal(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, manifest)
}

func (s *Server) handleInstallModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	var req modelmgr.InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apierrors.BadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(req.Lang) == "" {
		writeError(w, apierrors.InvalidLanguage("lang is required for install"))
		return
	}
	entry, err := s.models.Install(r.Context(), req)
	if err != nil {
		writeError(w, apierrors.Internal(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/models/"), "/")
	if len(parts) != 2 {
		writeError(w, apierrors.BadRequest("path must be /v1/models/{lang}/{version}"))
		return
	}
	lang := parts[0]
	version := parts[1]
	force := false
	if v := r.URL.Query().Get("force"); v != "" {
		fv, _ := strconv.ParseBool(v)
		force = fv
	}
	if err := s.models.Delete(lang, version, force); err != nil {
		writeError(w, apierrors.ModelMissing(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleSpeak(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	var req synth.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apierrors.BadRequest("invalid request body"))
		return
	}
	audio, err := s.synthSvc.Submit(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			writeError(w, apierrors.SynthTimeout(err.Error()))
		case strings.Contains(err.Error(), "installed model required"):
			writeError(w, apierrors.ModelMissing(err.Error()))
		default:
			writeError(w, apierrors.Internal(err.Error()))
		}
		return
	}
	if s.cfg.PlayOnSpeak {
		go func() {
			if err := playback.PlayBytes(audio.Data, audio.Format, audio.SampleRate); err != nil {
				s.logger.Warn("speaker playback failed", "err", err, "lang", req.Lang)
			}
		}()
	}
	switch audio.Format {
	case "pcm_s16le":
		w.Header().Set("Content-Type", "application/octet-stream")
	default:
		w.Header().Set("Content-Type", "audio/wav")
	}
	w.Header().Set("X-Sample-Rate", fmt.Sprintf("%d", audio.SampleRate))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio.Data)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	var req struct {
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apierrors.BadRequest("invalid request body"))
		return
	}
	if req.RequestID == "" {
		writeError(w, apierrors.BadRequest("request_id is required"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stopped": s.synthSvc.Cancel(req.RequestID)})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, apierrors.BadRequest("method not allowed"))
		return
	}
	total, failed := s.synthSvc.Metrics()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(
		fmt.Sprintf("tts_requests_total %d\ntts_requests_failed_total %d\n", total, failed),
	))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err apierrors.APIError) {
	writeJSON(w, err.Status, err)
}

func withMaxBytes(next http.Handler, max int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, max)
		}
		next.ServeHTTP(w, r)
	})
}

func firstInstalledByLang(manifest modelmgr.Manifest) map[string]string {
	loaded := make(map[string]string, len(manifest.Installed))
	for lang, installed := range manifest.Installed {
		if len(installed) == 0 {
			continue
		}
		loaded[lang] = installed[0].Version
	}
	return loaded
}

func installedLangs(manifest modelmgr.Manifest) []string {
	langs := make([]string, 0, len(manifest.Installed))
	for lang, installed := range manifest.Installed {
		if len(installed) == 0 {
			continue
		}
		langs = append(langs, lang)
	}
	return langs
}
