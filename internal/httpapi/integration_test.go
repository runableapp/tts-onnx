package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keith/linux-tts-onnx/internal/config"
	"github.com/keith/linux-tts-onnx/internal/modelmgr"
	"github.com/keith/linux-tts-onnx/internal/synth"
)

func TestIntegrationHealthAndSpeak(t *testing.T) {
	cfg := config.Default()
	cfg.BearerToken = ""
	cfg.SynthTimeout = 2 * time.Second
	tmp := t.TempDir()
	cfg.DataDir = filepath.Join(tmp, "data")
	cfg.TempDir = filepath.Join(tmp, "tmp")
	cfg.ModelManifestPath = filepath.Join(cfg.DataDir, "models", "manifest.json")
	if err := cfg.EnsureDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}

	store := modelmgr.NewManifestStore(cfg.ModelManifestPath)
	if err := store.Save(modelmgr.Manifest{
		Installed: map[string][]modelmgr.InstalledModel{
			"en": {
				{Version: "v1", Path: filepath.Join(tmp, "model")},
			},
		},
	}); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	models := modelmgr.New(filepath.Join(cfg.DataDir, "models"), cfg.TempDir, store)
	synthSvc := synth.NewService(&synth.MockEngine{}, models, 1, cfg.SynthTimeout, cfg.MaxTextChars)
	t.Cleanup(synthSvc.Stop)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := New(cfg, logger, models, synthSvc, "mock")
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	res, err := http.Get(ts.URL + "/v1/health")
	if err != nil {
		t.Fatalf("health call failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health status: %d", res.StatusCode)
	}

	res, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("root call failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("root status: %d", res.StatusCode)
	}

	body, _ := json.Marshal(map[string]any{
		"text":   "hello",
		"lang":   "en",
		"format": "wav",
	})
	res, err = http.Post(ts.URL+"/v1/speak", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("speak call failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("speak status: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "audio/wav" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
}
