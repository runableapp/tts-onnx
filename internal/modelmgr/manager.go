package modelmgr

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Manager struct {
	baseDir string
	tmpDir  string
	store   *ManifestStore
	client  *http.Client
}

type InstallRequest struct {
	Lang     string `json:"lang"`
	ModelID  string `json:"model_id"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
	Version  string `json:"version"`
}

func New(baseDir, tmpDir string, store *ManifestStore) *Manager {
	return &Manager{
		baseDir: baseDir,
		tmpDir:  tmpDir,
		store:   store,
		client: &http.Client{
			Timeout: 20 * time.Minute,
		},
	}
}

func (m *Manager) List() (Manifest, error) {
	return m.store.Load()
}

func (m *Manager) Install(ctx context.Context, req InstallRequest) (InstalledModel, error) {
	if req.Lang == "" {
		return InstalledModel{}, errors.New("lang is required")
	}
	url := req.URL
	checksum := strings.ToLower(strings.TrimSpace(req.Checksum))
	modelID := req.ModelID
	version := req.Version
	if url == "" {
		return InstalledModel{}, errors.New("url is required")
	}
	if version == "" {
		version = time.Now().UTC().Format("20060102-150405")
	}

	if err := os.MkdirAll(m.tmpDir, 0o755); err != nil {
		return InstalledModel{}, err
	}
	downloadPath := filepath.Join(m.tmpDir, fmt.Sprintf("%s-%s%s", req.Lang, version, archiveSuffix(url)))
	if err := m.download(ctx, url, downloadPath); err != nil {
		return InstalledModel{}, err
	}
	defer os.Remove(downloadPath)

	actualSHA, err := fileSHA256(downloadPath)
	if err != nil {
		return InstalledModel{}, err
	}
	if checksum != "" && actualSHA != checksum {
		return InstalledModel{}, fmt.Errorf("checksum mismatch: want %s, got %s", checksum, actualSHA)
	}

	destDir := filepath.Join(m.baseDir, req.Lang, version)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return InstalledModel{}, err
	}
	if err := extractArtifact(downloadPath, destDir); err != nil {
		return InstalledModel{}, err
	}

	entry := InstalledModel{
		ID:          modelID,
		Version:     version,
		SHA256:      actualSHA,
		License:     "",
		Attribution: "",
		InstalledAt: time.Now().UTC(),
		Path:        destDir,
	}
	manifest, err := m.store.Load()
	if err != nil {
		return InstalledModel{}, err
	}
	if strings.TrimSpace(modelID) != "" {
		// Keep only the newest install for the same model id within a language.
		// Older versions are removed from manifest and disk.
		kept := make([]InstalledModel, 0, len(manifest.Installed[req.Lang]))
		for _, existing := range manifest.Installed[req.Lang] {
			if existing.ID == modelID {
				// If reinstalling the same version/path, keep extracted files.
				// Also skip deletion when existing path is inside destDir,
				// since model archives often extract nested directories there.
				if filepath.Clean(existing.Path) != filepath.Clean(destDir) && !isPathWithin(existing.Path, destDir) {
					_ = os.RemoveAll(existing.Path)
				}
				continue
			}
			kept = append(kept, existing)
		}
		manifest.Installed[req.Lang] = kept
	}
	manifest.Installed[req.Lang] = append(manifest.Installed[req.Lang], entry)
	if err := m.store.Save(manifest); err != nil {
		return InstalledModel{}, err
	}
	return entry, nil
}

func (m *Manager) FirstInstalledPath(lang string) (path, version string, err error) {
	manifest, err := m.store.Load()
	if err != nil {
		return "", "", err
	}
	installed := manifest.Installed[lang]
	if len(installed) == 0 {
		return "", "", fmt.Errorf("no installed model for %s", lang)
	}
	return installed[0].Path, installed[0].Version, nil
}

// ResolvePath resolves the model path for a request.
// If selector matches an installed model version/id for the language,
// it selects that model directly and returns matched=true.
// Otherwise it returns the first installed model and matched=false.
func (m *Manager) ResolvePath(lang, selector string) (path, version string, matched bool, err error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		p, v, e := m.FirstInstalledPath(lang)
		return p, v, false, e
	}
	manifest, err := m.store.Load()
	if err != nil {
		return "", "", false, err
	}
	for _, e := range manifest.Installed[lang] {
		if e.Version == selector || e.ID == selector {
			return e.Path, e.Version, true, nil
		}
	}
	p, v, e := m.FirstInstalledPath(lang)
	return p, v, false, e
}

// ResolvePathAny resolves model path without requiring an explicit language.
// If lang is provided, it behaves like ResolvePath and returns that language.
// If lang is empty:
//   - with selector: it searches all installed languages for id/version match
//   - without selector: it picks the first installed model in sorted language order
func (m *Manager) ResolvePathAny(lang, selector string) (resolvedLang, path, version string, matched bool, err error) {
	selector = strings.TrimSpace(selector)
	if strings.TrimSpace(lang) != "" {
		p, v, matched, err := m.ResolvePath(lang, selector)
		return lang, p, v, matched, err
	}

	manifest, err := m.store.Load()
	if err != nil {
		return "", "", "", false, err
	}
	langs := make([]string, 0, len(manifest.Installed))
	for l := range manifest.Installed {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	if selector != "" {
		for _, l := range langs {
			for _, e := range manifest.Installed[l] {
				if e.Version == selector || e.ID == selector {
					return l, e.Path, e.Version, true, nil
				}
			}
		}
	}

	for _, l := range langs {
		installed := manifest.Installed[l]
		if len(installed) == 0 {
			continue
		}
		return l, installed[0].Path, installed[0].Version, false, nil
	}

	if selector != "" {
		return "", "", "", false, fmt.Errorf("no installed model matching selector %q", selector)
	}
	return "", "", "", false, errors.New("no installed model")
}

func (m *Manager) Delete(lang, version string, force bool) error {
	manifest, err := m.store.Load()
	if err != nil {
		return err
	}
	models := manifest.Installed[lang]
	next := make([]InstalledModel, 0, len(models))
	var deleted *InstalledModel
	for _, e := range models {
		if e.Version == version {
			copied := e
			deleted = &copied
			continue
		}
		next = append(next, e)
	}
	if deleted == nil {
		return fmt.Errorf("model version not found: %s/%s", lang, version)
	}
	manifest.Installed[lang] = next
	if err := m.store.Save(manifest); err != nil {
		return err
	}
	return os.RemoveAll(deleted.Path)
}

func (m *Manager) download(ctx context.Context, url, out string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("download failed with status %d", res.StatusCode)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if os.Getenv("TTS_DOWNLOAD_PROGRESS") != "1" {
		_, err = io.Copy(f, res.Body)
		return err
	}
	name := filepath.Base(stripURLQuery(url))
	if name == "" || name == "." || name == "/" {
		name = "model"
	}
	_, err = copyWithProgress(f, res.Body, res.ContentLength, name)
	return err
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64, label string) (int64, error) {
	const (
		barWidth      = 28
		updateEvery   = 200 * time.Millisecond
		readBufferLen = 32 * 1024
	)
	buf := make([]byte, readBufferLen)
	var written int64
	lastDraw := time.Now().Add(-updateEvery)

	draw := func(force bool) {
		if !force && time.Since(lastDraw) < updateEvery {
			return
		}
		lastDraw = time.Now()
		if total > 0 {
			pct := float64(written) / float64(total)
			if pct > 1 {
				pct = 1
			}
			filled := int(pct * barWidth)
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
			fmt.Fprintf(os.Stderr, "\rDownloading %-40s [%s] %6.2f%% (%s/%s)", label, bar, pct*100, humanBytes(written), humanBytes(total))
			return
		}
		fmt.Fprintf(os.Stderr, "\rDownloading %-40s %s", label, humanBytes(written))
	}

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
			draw(false)
		}
		if er != nil {
			if er == io.EOF {
				draw(true)
				fmt.Fprintln(os.Stderr)
				return written, nil
			}
			return written, er
		}
	}
}

func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(n)
	u := 0
	for size >= 1024 && u < len(units)-1 {
		size /= 1024
		u++
	}
	return fmt.Sprintf("%.1f%s", size, units[u])
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractArtifact(srcPath, destDir string) error {
	lower := strings.ToLower(srcPath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(srcPath, destDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(srcPath, destDir)
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz2"):
		return extractTarBz2(srcPath, destDir)
	default:
		dst := filepath.Join(destDir, filepath.Base(srcPath))
		return copyFile(srcPath, dst)
	}
}

func archiveSuffix(url string) string {
	lower := strings.ToLower(stripURLQuery(url))
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return ".tar.gz"
	case strings.HasSuffix(lower, ".tgz"):
		return ".tgz"
	case strings.HasSuffix(lower, ".tar.bz2"):
		return ".tar.bz2"
	case strings.HasSuffix(lower, ".tbz2"):
		return ".tbz2"
	case strings.HasSuffix(lower, ".zip"):
		return ".zip"
	default:
		return ".pkg"
	}
}

func stripURLQuery(s string) string {
	if i := strings.IndexByte(s, '?'); i >= 0 {
		return s[:i]
	}
	return s
}

func extractZip(srcPath, destDir string) error {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("zip path traversal blocked: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return err
		}
		in.Close()
		out.Close()
	}
	return nil
}

func extractTarGz(srcPath, destDir string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return untar(gz, destDir)
}

func extractTarBz2(srcPath, destDir string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	bz := bzip2.NewReader(f)
	return untar(bz, destDir)
}

func untar(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("tar path traversal blocked: %q", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func isPathWithin(path, parent string) bool {
	path = filepath.Clean(path)
	parent = filepath.Clean(parent)
	if path == parent {
		return true
	}
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
