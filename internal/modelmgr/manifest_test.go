package modelmgr

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManifestStoreRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "manifest.json")
	store := NewManifestStore(path)

	m := Manifest{
		UpdatedAt: time.Now().UTC(),
		Installed: map[string][]InstalledModel{
			"en": {
				{
					ID:      "model",
					Version: "v1",
					Path:    "/tmp/model",
				},
			},
		},
	}
	if err := store.Save(m); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(got.Installed["en"]) != 1 {
		t.Fatalf("expected one installed model")
	}
}
