package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

const sherpaTTSReleaseAPI = "https://api.github.com/repos/k2-fsa/sherpa-onnx/releases/tags/tts-models"

type githubRelease struct {
	Assets []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type remoteModel struct {
	ID      string
	Lang    string
	Version string
	URL     string
}

func fetchRemoteModels(ctx context.Context) ([]remoteModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sherpaTTSReleaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "linux-tts-onnx/cli")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("remote catalog request failed with status %d", res.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return nil, err
	}
	models := make([]remoteModel, 0, len(rel.Assets))
	for _, a := range rel.Assets {
		if !isTTSArchive(a.Name) {
			continue
		}
		id := trimArchiveSuffix(a.Name)
		models = append(models, remoteModel{
			ID:      id,
			Lang:    inferLang(id),
			Version: inferVersion(id),
			URL:     a.BrowserDownloadURL,
		})
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Lang == models[j].Lang {
			return models[i].ID < models[j].ID
		}
		return models[i].Lang < models[j].Lang
	})
	return models, nil
}

func isTTSArchive(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".tar.bz2") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".zip")
}

func trimArchiveSuffix(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tar.bz2"):
		return name[:len(name)-len(".tar.bz2")]
	case strings.HasSuffix(lower, ".tar.gz"):
		return name[:len(name)-len(".tar.gz")]
	case strings.HasSuffix(lower, ".tgz"):
		return name[:len(name)-len(".tgz")]
	case strings.HasSuffix(lower, ".zip"):
		return name[:len(name)-len(".zip")]
	default:
		return strings.TrimSuffix(name, path.Ext(name))
	}
}

func inferLang(id string) string {
	l := strings.ToLower(id)
	switch {
	case hasLangToken(l, "en"):
		return "en"
	case hasLangToken(l, "ko"):
		return "ko"
	case hasLangToken(l, "ja"):
		return "ja"
	default:
		return "unknown"
	}
}

func hasLangToken(s, lang string) bool {
	for i := 0; i+len(lang) <= len(s); i++ {
		if s[i:i+len(lang)] != lang {
			continue
		}
		leftOK := i == 0 || !isAlphaNum(s[i-1])
		rightIdx := i + len(lang)
		rightOK := rightIdx == len(s) || !isAlphaNum(s[rightIdx])
		if leftOK && rightOK {
			return true
		}
	}
	return false
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func inferVersion(id string) string {
	re := regexp.MustCompile(`v[0-9][a-z0-9._-]*`)
	v := re.FindString(strings.ToLower(id))
	if v == "" {
		return time.Now().UTC().Format("20060102-150405")
	}
	return v
}

