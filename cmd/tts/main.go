package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/keith/linux-tts-onnx/internal/config"
	"github.com/keith/linux-tts-onnx/internal/modelmgr"
	"github.com/keith/linux-tts-onnx/internal/playback"
	"github.com/keith/linux-tts-onnx/internal/synth"
)

func main() {
	var (
		lang            string
		format          string
		outFile         string
		cfgPath         string
		serviceMode     bool
		rate            float64
		sampleHz        int
		requestID       string
		noPlay          bool
		voiceList       bool
		remoteModels    bool
		installRemoteID string
		menu            bool
		voice           string
		allowInstall    bool
	)

	flag.StringVar(&lang, "lang", "", "language (optional for synthesis; inferred from installed model when omitted)")
	flag.StringVar(&voice, "voice", "", "voice name or speaker id for supported multi-voice models")
	flag.StringVar(&format, "format", "wav", "audio format: wav|pcm_s16le")
	flag.StringVar(&outFile, "out", "", "output file path (optional)")
	flag.StringVar(&cfgPath, "config", "./config/config.sherpa.yaml", "config file path")
	flag.BoolVar(&serviceMode, "service", false, "run HTTP service mode")
	flag.Float64Var(&rate, "rate", 1.0, "speech rate multiplier")
	flag.IntVar(&sampleHz, "sample-rate", 0, "optional sample rate override")
	flag.StringVar(&requestID, "request-id", "", "optional request id")
	flag.BoolVar(&noPlay, "no-play", false, "disable immediate speaker playback")
	flag.BoolVar(&voiceList, "voice-list", false, "list voices for installed models and exit")
	flag.BoolVar(&remoteModels, "remote-models", false, "list online TTS model packages from sherpa-onnx release")
	flag.StringVar(&installRemoteID, "install-remote-id", "", "install model by remote ID from online sherpa-onnx release")
	flag.BoolVar(&menu, "menu", false, "interactive menu: choose language/model, auto-install, and select voice")
	flag.BoolVar(&allowInstall, "auto-install", true, "allow automatic model install when selecting from menu")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [options] <text>\n\nOptions:\n", filepath.Base(os.Args[0]))
		flag.VisitAll(func(f *flag.Flag) {
			if f.DefValue == "false" {
				fmt.Fprintf(os.Stdout, "  --%s\n      %s\n", f.Name, f.Usage)
				return
			}
			fmt.Fprintf(os.Stdout, "  --%s (default %q)\n      %s\n", f.Name, f.DefValue, f.Usage)
		})
	}
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	for _, arg := range os.Args[1:] {
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "--") || !strings.HasPrefix(arg, "-") {
			continue
		}
		exitf("invalid flag %q: use double-dash format (example: --voice-list)", arg)
	}
	flag.Parse()

	if serviceMode {
		if err := runService(cfgPath); err != nil {
			exitf("service failed: %v", err)
		}
		return
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		exitf("load config: %v", err)
	}
	if cfg.Engine == "" || cfg.Engine == "mock" {
		cfg.Engine = "sherpa-onnx"
	}
	if err := cfg.EnsureDirs(); err != nil {
		exitf("ensure dirs: %v", err)
	}

	store := modelmgr.NewManifestStore(cfg.ModelManifestPath)
	models := modelmgr.New(filepath.Join(cfg.DataDir, "models"), cfg.TempDir, store)
	manifest, err := models.List()
	if err != nil {
		exitf("load manifest: %v", err)
	}

	if remoteModels || strings.TrimSpace(installRemoteID) != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		remote, err := fetchRemoteModels(ctx)
		if err != nil {
			exitf("load remote models: %v", err)
		}
		if remoteModels {
			for _, m := range remote {
				if lang != "" && m.Lang != "unknown" && lang != m.Lang {
					continue
				}
				fmt.Printf("- lang=%s id=%s version=%s url=%s\n", m.Lang, m.ID, m.Version, m.URL)
			}
		}
		if strings.TrimSpace(installRemoteID) != "" {
			targetID := strings.TrimSpace(installRemoteID)
			var chosen *remoteModel
			for i := range remote {
				if remote[i].ID == targetID {
					chosen = &remote[i]
					break
				}
			}
			if chosen == nil {
				exitf("remote model id not found: %s", targetID)
			}
			resolvedLang := chosen.Lang
			if resolvedLang == "unknown" {
				// Allow explicit override for multi-language or ambiguous IDs.
				if lang != "" {
					resolvedLang = lang
				} else {
					exitf("cannot infer language for remote model %q from remote metadata (pass --lang <language>)", targetID)
				}
			}
			if existing, ok := findInstalledByIDVersion(manifest, resolvedLang, chosen.ID, chosen.Version); ok {
				fmt.Printf("model already installed: lang=%s id=%s version=%s path=%s\n", resolvedLang, existing.ID, existing.Version, existing.Path)
			} else {
				prevProgress := os.Getenv("TTS_DOWNLOAD_PROGRESS")
				_ = os.Setenv("TTS_DOWNLOAD_PROGRESS", "1")
				defer func() {
					_ = os.Setenv("TTS_DOWNLOAD_PROGRESS", prevProgress)
				}()
				entry, err := models.Install(context.Background(), modelmgr.InstallRequest{
					Lang:    resolvedLang,
					ModelID: chosen.ID,
					URL:     chosen.URL,
					Version: chosen.Version,
				})
				if err != nil {
					exitf("install remote model: %v", err)
				}
				fmt.Printf("installed remote model: lang=%s id=%s version=%s\n", resolvedLang, entry.ID, entry.Version)
				manifest, _ = models.List()
			}
		}
		if strings.TrimSpace(strings.Join(flag.Args(), " ")) == "" {
			return
		}
	}

	if voiceList {
		langs := make([]string, 0, len(manifest.Installed))
		for l := range manifest.Installed {
			langs = append(langs, l)
		}
		sort.Strings(langs)
		if lang != "" {
			langs = []string{lang}
		}
		printed := false
		for _, currentLang := range langs {
			installed := manifest.Installed[currentLang]
			if len(installed) == 0 {
				continue
			}
			fmt.Printf("lang=%s\n", currentLang)
			for _, im := range installed {
				modelName := strings.TrimSpace(im.ID)
				if modelName == "" {
					modelName = im.Version
				}
				fmt.Printf("model: %s (version=%s)\n", modelName, im.Version)
				voices := synth.KnownVoicesForModelPath(im.Path)
				if len(voices) == 0 {
					n, err := synth.NumSpeakersForModelPath(currentLang, im.Path)
					if err != nil {
						fmt.Println("  voices: <numeric sid available but count unknown>")
						printed = true
						continue
					}
					if n == 1 {
						fmt.Println("  voices: sid 0")
						printed = true
						continue
					}
					fmt.Printf("  voices: sid 0..%d\n", n-1)
					printed = true
					continue
				}
				for i, v := range voices {
					fmt.Printf("  %d\t%s\n", i, v)
				}
				printed = true
			}
		}
		if !printed {
			exitf("no installed models found for voice listing")
		}
		return
	}

	text := strings.TrimSpace(strings.Join(flag.Args(), " "))
	for _, a := range flag.Args() {
		if strings.HasPrefix(a, "-") && a != "-" {
			exitf("unexpected flag-like token %q after text; place all flags before text", a)
		}
	}
	if format != "wav" && format != "pcm_s16le" {
		exitf("invalid --format %q (must be wav|pcm_s16le)", format)
	}

	if menu {
		lang, voice, text = runInteractiveMenu(manifest, models, allowInstall, text)
	}

	if text == "" {
		exitf("missing text argument, usage: tts [--lang <language>] \"hello world\"")
	}

	var engine synth.Engine
	if cfg.Engine == "sherpa-onnx" {
		engine = synth.NewSherpaEngine()
	} else {
		engine = &synth.MockEngine{}
	}
	svc := synth.NewService(engine, models, 1, cfg.SynthTimeout, cfg.MaxTextChars)
	defer svc.Stop()

	req := synth.Request{
		Text:       text,
		Lang:       lang,
		Voice:      voice,
		Rate:       rate,
		Format:     format,
		SampleRate: sampleHz,
		RequestID:  requestID,
	}
	audio, err := svc.Submit(context.Background(), req)
	if err != nil {
		exitf("synthesize failed: %v\nhint: install a model first, or pass --lang to select language explicitly", err)
	}

	if strings.TrimSpace(outFile) != "" {
		if err := os.WriteFile(outFile, audio.Data, 0o644); err != nil {
			exitf("write output: %v", err)
		}
		fmt.Printf("ok: wrote %s (%d bytes, sample_rate=%d, format=%s)\n", outFile, len(audio.Data), audio.SampleRate, audio.Format)
		if !noPlay {
			if err := playback.PlayFile(outFile, audio.Format, audio.SampleRate); err != nil {
				fmt.Fprintf(os.Stderr, "warning: playback failed: %v\n", err)
			}
		}
		return
	}
	if !noPlay {
		if err := playback.PlayBytes(audio.Data, audio.Format, audio.SampleRate); err != nil {
			fmt.Fprintf(os.Stderr, "warning: playback failed: %v\n", err)
		}
	}
	fmt.Printf("ok: synthesized audio (%d bytes, sample_rate=%d, format=%s)\n", len(audio.Data), audio.SampleRate, audio.Format)
}

func exitf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

type menuModel struct {
	Lang      string
	ID        string
	Version   string
	URL       string
	Installed bool
}

func runInteractiveMenu(manifest modelmgr.Manifest, models *modelmgr.Manager, allowInstall bool, initialText string) (lang, voice, text string) {
	reader := bufio.NewReader(os.Stdin)
	langs := make([]string, 0, len(manifest.Installed))
	for l, installed := range manifest.Installed {
		if len(installed) > 0 {
			langs = append(langs, l)
		}
	}
	sort.Strings(langs)
	if len(langs) == 0 {
		langs = []string{"en", "ko", "ja"}
	}
	fmt.Println("Select language:")
	for i, l := range langs {
		fmt.Printf("  %d) %s\n", i+1, l)
	}
	lang = langs[promptSelect(reader, len(langs))-1]

	langModels := make([]menuModel, 0)
	seenVersions := map[string]bool{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	remote, remoteErr := fetchRemoteModels(ctx)
	cancel()
	if remoteErr == nil {
		for _, rm := range remote {
			if rm.Lang != lang {
				continue
			}
			installed := isInstalledVersion(manifest, lang, rm.Version)
			langModels = append(langModels, menuModel{
				Lang:      lang,
				ID:        rm.ID,
				Version:   rm.Version,
				URL:       rm.URL,
				Installed: installed,
			})
			seenVersions[rm.Version] = true
		}
	}
	for _, im := range manifest.Installed[lang] {
		if seenVersions[im.Version] {
			continue
		}
		id := strings.TrimSpace(im.ID)
		if id == "" {
			id = im.Version
		}
		langModels = append(langModels, menuModel{
			Lang:      lang,
			ID:        id,
			Version:   im.Version,
			URL:       "",
			Installed: true,
		})
		seenVersions[im.Version] = true
	}
	if len(langModels) == 0 {
		exitf("no installed or remote models found for %s", lang)
	}
	sort.Slice(langModels, func(i, j int) bool { return langModels[i].ID < langModels[j].ID })
	fmt.Printf("Select model for %s:\n", lang)
	if remoteErr != nil {
		fmt.Printf("  (warning: remote list unavailable: %v)\n", remoteErr)
	}
	for i, m := range langModels {
		installed := "not-installed"
		if m.Installed {
			installed = "installed"
		}
		fmt.Printf("  %d) %s (%s, %s)\n", i+1, m.ID, m.Version, installed)
	}
	chosen := langModels[promptSelect(reader, len(langModels))-1]

	if !chosen.Installed {
		if !allowInstall {
			exitf("model %s/%s not installed and --auto-install=false", lang, chosen.Version)
		}
		if strings.TrimSpace(chosen.URL) == "" {
			exitf("model %s/%s has no remote URL for install", lang, chosen.Version)
		}
		fmt.Printf("Installing %s for %s...\n", chosen.ID, lang)
		prevProgress := os.Getenv("TTS_DOWNLOAD_PROGRESS")
		_ = os.Setenv("TTS_DOWNLOAD_PROGRESS", "1")
		if _, err := models.Install(context.Background(), modelmgr.InstallRequest{
			Lang:    lang,
			ModelID: chosen.ID,
			URL:     chosen.URL,
			Version: chosen.Version,
		}); err != nil {
			_ = os.Setenv("TTS_DOWNLOAD_PROGRESS", prevProgress)
			exitf("install failed: %v", err)
		}
		_ = os.Setenv("TTS_DOWNLOAD_PROGRESS", prevProgress)
		manifest, _ = models.List()
	}
	modelPath, err := installedPathByVersion(manifest, lang, chosen.Version)
	if err != nil {
		exitf("installed model path not found: %v", err)
	}

	voices := synth.KnownVoicesForModelPath(modelPath)
	if len(voices) > 0 {
		fmt.Println("Select voice:")
		for i, v := range voices {
			fmt.Printf("  %d) %s\n", i+1, v)
			if i >= 24 {
				fmt.Println("  ... (truncated display; enter index manually for higher voices)")
				break
			}
		}
		fmt.Print("Voice index or name (Enter for default): ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			if idx, err := strconv.Atoi(line); err == nil && idx >= 0 && idx < len(voices) {
				voice = voices[idx]
			} else {
				voice = line
			}
		}
	}

	text = strings.TrimSpace(initialText)
	if text == "" {
		fmt.Print("Enter text: ")
		line, _ := reader.ReadString('\n')
		text = strings.TrimSpace(line)
	}
	return lang, voice, text
}

func promptSelect(reader *bufio.Reader, max int) int {
	for {
		fmt.Printf("Choose [1-%d]: ", max)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		n, err := strconv.Atoi(line)
		if err == nil && n >= 1 && n <= max {
			return n
		}
		fmt.Println("Invalid selection")
	}
}

func isInstalledVersion(m modelmgr.Manifest, lang, version string) bool {
	for _, im := range m.Installed[lang] {
		if im.Version == version {
			return true
		}
	}
	return false
}

func installedPathByVersion(m modelmgr.Manifest, lang, version string) (string, error) {
	for _, im := range m.Installed[lang] {
		if im.Version == version {
			return im.Path, nil
		}
	}
	return "", fmt.Errorf("model not installed for %s/%s", lang, version)
}

func findInstalledByIDVersion(m modelmgr.Manifest, lang, id, version string) (modelmgr.InstalledModel, bool) {
	for _, im := range m.Installed[lang] {
		if im.ID == id && im.Version == version {
			return im, true
		}
	}
	return modelmgr.InstalledModel{}, false
}
