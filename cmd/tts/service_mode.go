package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/keith/linux-tts-onnx/internal/config"
	"github.com/keith/linux-tts-onnx/internal/httpapi"
	"github.com/keith/linux-tts-onnx/internal/modelmgr"
	"github.com/keith/linux-tts-onnx/internal/synth"
)

func runService(cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting service", "listen_addr", cfg.ListenAddr, "engine", cfg.Engine)

	store := modelmgr.NewManifestStore(cfg.ModelManifestPath)
	models := modelmgr.New(
		filepath.Join(cfg.DataDir, "models"),
		cfg.TempDir,
		store,
	)

	var engine synth.Engine = &synth.MockEngine{}
	if cfg.Engine == "sherpa-onnx" {
		engine = synth.NewSherpaEngine()
	}
	synthSvc := synth.NewService(
		engine,
		models,
		cfg.MaxConcurrentRequests,
		cfg.SynthTimeout,
		cfg.MaxTextChars,
	)
	defer synthSvc.Stop()

	api := httpapi.New(cfg, logger, models, synthSvc, engine.Name())
	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      api.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

