package synth

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

const leadInSilenceMs = 50

type Request struct {
	Text       string  `json:"text"`
	Lang       string  `json:"lang"`
	Voice      string  `json:"voice"`
	Rate       float64 `json:"rate"`
	Format     string  `json:"format"`
	SampleRate int     `json:"sample_rate"`
	RequestID  string  `json:"request_id"`
}

type Audio struct {
	Data       []byte
	Format     string
	SampleRate int
}

type Engine interface {
	LoadModel(lang, path string) error
	UnloadModel(lang string) error
	Synthesize(ctx context.Context, req Request, modelPath string) (Audio, error)
	Name() string
}

type MockEngine struct{}

func (m *MockEngine) Name() string                      { return "mock" }
func (m *MockEngine) LoadModel(lang, path string) error { return nil }
func (m *MockEngine) UnloadModel(lang string) error     { return nil }

func (m *MockEngine) Synthesize(ctx context.Context, req Request, modelPath string) (Audio, error) {
	if strings.TrimSpace(req.Text) == "" {
		return Audio{}, errors.New("text is required")
	}
	sr := req.SampleRate
	if sr <= 0 {
		sr = 22050
	}
	durationSec := 0.2 + math.Min(float64(len(req.Text))/60.0, 3.0)
	samples := int(durationSec * float64(sr))
	pcm := make([]int16, samples)
	for i := 0; i < samples; i++ {
		select {
		case <-ctx.Done():
			return Audio{}, ctx.Err()
		default:
		}
		v := math.Sin(2 * math.Pi * 440 * float64(i) / float64(sr))
		pcm[i] = int16(v * 16000)
	}
	outFmt := req.Format
	if outFmt == "" || outFmt == "wav" {
		return Audio{Data: PCM16ToWAV(pcm, sr), Format: "wav", SampleRate: sr}, nil
	}
	if outFmt == "pcm_s16le" {
		return Audio{Data: Int16ToBytesLE(pcm), Format: "pcm_s16le", SampleRate: sr}, nil
	}
	return Audio{}, fmt.Errorf("unsupported format: %s", req.Format)
}

type loadedTTS struct {
	tts *sherpa.OfflineTts
}

type SherpaEngine struct {
	mu    sync.RWMutex
	model map[string]loadedTTS
}

func NewSherpaEngine() *SherpaEngine {
	return &SherpaEngine{model: map[string]loadedTTS{}}
}

func (s *SherpaEngine) Name() string { return "sherpa-onnx" }

func (s *SherpaEngine) LoadModel(lang, path string) error {
	cfg, err := buildSherpaTTSConfig(path, lang)
	if err != nil {
		return err
	}
	tts := sherpa.NewOfflineTts(cfg)
	if tts == nil {
		return fmt.Errorf("failed to initialize sherpa offline tts for %s", path)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if cur, ok := s.model[lang]; ok && cur.tts != nil {
		sherpa.DeleteOfflineTts(cur.tts)
	}
	s.model[lang] = loadedTTS{tts: tts}
	return nil
}

func (s *SherpaEngine) UnloadModel(lang string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.model[lang]
	if !ok {
		return nil
	}
	if cur.tts != nil {
		sherpa.DeleteOfflineTts(cur.tts)
	}
	delete(s.model, lang)
	return nil
}

func (s *SherpaEngine) Synthesize(ctx context.Context, req Request, modelPath string) (Audio, error) {
	s.mu.RLock()
	cur, ok := s.model[req.Lang]
	s.mu.RUnlock()
	if !ok || cur.tts == nil {
		return Audio{}, errors.New("model not loaded")
	}
	select {
	case <-ctx.Done():
		return Audio{}, ctx.Err()
	default:
	}

	speed := float32(1.0)
	if req.Rate > 0 {
		speed = float32(req.Rate)
	}
	sid, err := ResolveSpeakerID(modelPath, req.Voice)
	if err != nil {
		return Audio{}, err
	}
	generated := cur.tts.Generate(req.Text, sid, speed)
	if generated == nil || len(generated.Samples) == 0 {
		return Audio{}, errors.New("sherpa returned empty audio")
	}

	sr := generated.SampleRate
	if sr <= 0 {
		sr = 22050
	}
	pcm := float32ToPCM16(generated.Samples)
	pcm = prependSilencePCM16(pcm, sr, leadInSilenceMs)
	outFmt := req.Format
	if outFmt == "" || outFmt == "wav" {
		return Audio{Data: PCM16ToWAV(pcm, sr), Format: "wav", SampleRate: sr}, nil
	}
	if outFmt == "pcm_s16le" {
		return Audio{Data: Int16ToBytesLE(pcm), Format: "pcm_s16le", SampleRate: sr}, nil
	}
	return Audio{}, fmt.Errorf("unsupported format: %s", req.Format)
}

func buildSherpaTTSConfig(modelDir, lang string) (*sherpa.OfflineTtsConfig, error) {
	modelFile := firstExisting(
		filepath.Join(modelDir, "model.fp16.onnx"),
		filepath.Join(modelDir, "model.onnx"),
		findInTree(modelDir, "model.fp16.onnx"),
		findInTree(modelDir, "model.onnx"),
		findFirstONNXInTree(modelDir),
	)
	if modelFile == "" {
		return nil, fmt.Errorf("no supported model file found in %s", modelDir)
	}

	cfg := &sherpa.OfflineTtsConfig{}
	cfg.Model.Provider = "cpu"
	cfg.Model.NumThreads = 2
	cfg.Model.Debug = 0
	cfg.MaxNumSentences = 1
	cfg.SilenceScale = 0.2

	tokens := firstExisting(filepath.Join(modelDir, "tokens.txt"))
	voices := firstExisting(filepath.Join(modelDir, "voices.bin"))
	espeakDir := firstExisting(filepath.Join(modelDir, "espeak-ng-data"))
	lexiconAny := firstExisting(
		filepath.Join(modelDir, "lexicon-us-en.txt"),
		filepath.Join(modelDir, "lexicon-gb-en.txt"),
		filepath.Join(modelDir, "lexicon-zh.txt"),
	)
	if tokens == "" {
		tokens = findInTree(modelDir, "tokens.txt")
	}
	if voices == "" {
		voices = findInTree(modelDir, "voices.bin")
	}
	if espeakDir == "" {
		espeakDir = findDirInTree(modelDir, "espeak-ng-data")
	}
	if lexiconAny == "" {
		lexiconAny = findInTree(modelDir, "lexicon-us-en.txt")
	}

	modelLower := strings.ToLower(modelFile)
	if tokens != "" && voices != "" && (strings.Contains(modelLower, "kokoro") || lexiconAny != "") {
		cfg.Model.Kokoro.Model = modelFile
		cfg.Model.Kokoro.Tokens = tokens
		cfg.Model.Kokoro.Voices = voices
		cfg.Model.Kokoro.DataDir = espeakDir
		cfg.Model.Kokoro.Lexicon = filepath.Dir(tokens)
		cfg.Model.Kokoro.Lang = kokoroLang(lang)
		cfg.Model.Kokoro.LengthScale = 1.0
		return cfg, nil
	}

	if tokens != "" && voices != "" {
		cfg.Model.Kitten.Model = modelFile
		cfg.Model.Kitten.Tokens = tokens
		cfg.Model.Kitten.Voices = voices
		cfg.Model.Kitten.DataDir = espeakDir
		cfg.Model.Kitten.LengthScale = 1.0
		return cfg, nil
	}

	if tokens != "" {
		cfg.Model.Vits.Model = modelFile
		cfg.Model.Vits.Tokens = tokens
		cfg.Model.Vits.DataDir = espeakDir
		cfg.Model.Vits.NoiseScale = 0.667
		cfg.Model.Vits.NoiseScaleW = 0.8
		cfg.Model.Vits.LengthScale = 1.0
		return cfg, nil
	}
	return nil, fmt.Errorf("unable to infer model type in %s", modelDir)
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func findInTree(root, base string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == base {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func findFirstONNXInTree(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(name, ".onnx") && !strings.HasSuffix(name, ".onnx.json") {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func findDirInTree(root, base string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && filepath.Base(path) == base {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func float32ToPCM16(samples []float32) []int16 {
	out := make([]int16, len(samples))
	for i, s := range samples {
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		out[i] = int16(math.Round(float64(s * 32767)))
	}
	return out
}

func prependSilencePCM16(pcm []int16, sampleRate, silenceMs int) []int16 {
	if len(pcm) == 0 || sampleRate <= 0 || silenceMs <= 0 {
		return pcm
	}
	silenceSamples := (sampleRate * silenceMs) / 1000
	if silenceSamples <= 0 {
		return pcm
	}
	out := make([]int16, silenceSamples+len(pcm))
	copy(out[silenceSamples:], pcm)
	return out
}

func kokoroLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en", "en-us":
		return "en-us"
	case "ja":
		return "ja"
	case "ko", "ko-kr":
		return "ko"
	case "zh", "zh-cn":
		return "zh"
	default:
		// Keep compatibility with unknown/new language tags.
		return strings.TrimSpace(lang)
	}
}

// NumSpeakersForModelPath loads model metadata through sherpa runtime and
// returns speaker count for that model.
func NumSpeakersForModelPath(lang, modelPath string) (int, error) {
	cfg, err := buildSherpaTTSConfig(modelPath, lang)
	if err != nil {
		return 0, err
	}
	tts := sherpa.NewOfflineTts(cfg)
	if tts == nil {
		return 0, fmt.Errorf("failed to initialize sherpa offline tts for %s", modelPath)
	}
	defer sherpa.DeleteOfflineTts(tts)
	n := tts.NumSpeakers()
	if n <= 0 {
		return 0, fmt.Errorf("model reports no speakers")
	}
	return n, nil
}
