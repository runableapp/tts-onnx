package synth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/keith/linux-tts-onnx/internal/modelmgr"
)

type Service struct {
	engine Engine
	models *modelmgr.Manager

	maxTextChars int
	timeout      time.Duration

	queue     chan job
	wg        sync.WaitGroup
	inflight  sync.Map
	loaded    map[string]string
	loadedMu  sync.Mutex
	totalReq  uint64
	totalFail uint64
}

type job struct {
	ctx    context.Context
	cancel context.CancelFunc
	req    Request
	respCh chan result
}

type result struct {
	audio Audio
	err   error
}

func NewService(engine Engine, models *modelmgr.Manager, workers int, timeout time.Duration, maxTextChars int) *Service {
	if workers < 1 {
		workers = 1
	}
	s := &Service{
		engine:       engine,
		models:       models,
		maxTextChars: maxTextChars,
		timeout:      timeout,
		queue:        make(chan job, workers*4),
		loaded:       map[string]string{},
	}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	return s
}

func (s *Service) Stop() {
	close(s.queue)
	s.wg.Wait()
}

func (s *Service) Metrics() (total, failed uint64) {
	return atomic.LoadUint64(&s.totalReq), atomic.LoadUint64(&s.totalFail)
}

func (s *Service) Submit(ctx context.Context, req Request) (Audio, error) {
	atomic.AddUint64(&s.totalReq, 1)
	if len(req.Text) == 0 {
		atomic.AddUint64(&s.totalFail, 1)
		return Audio{}, errors.New("text is required")
	}
	if len(req.Text) > s.maxTextChars {
		atomic.AddUint64(&s.totalFail, 1)
		return Audio{}, fmt.Errorf("text too long; max %d chars", s.maxTextChars)
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	respCh := make(chan result, 1)
	j := job{ctx: ctx, cancel: cancel, req: req, respCh: respCh}
	if req.RequestID != "" {
		s.inflight.Store(req.RequestID, cancel)
		defer s.inflight.Delete(req.RequestID)
	}

	select {
	case <-ctx.Done():
		atomic.AddUint64(&s.totalFail, 1)
		return Audio{}, ctx.Err()
	case s.queue <- j:
	}

	select {
	case <-ctx.Done():
		atomic.AddUint64(&s.totalFail, 1)
		return Audio{}, ctx.Err()
	case res := <-respCh:
		if res.err != nil {
			atomic.AddUint64(&s.totalFail, 1)
		}
		return res.audio, res.err
	}
}

func (s *Service) Cancel(requestID string) bool {
	v, ok := s.inflight.Load(requestID)
	if !ok {
		return false
	}
	cancel, ok := v.(context.CancelFunc)
	if !ok {
		return false
	}
	cancel()
	return true
}

func (s *Service) worker() {
	defer s.wg.Done()
	for j := range s.queue {
		audio, err := s.synthesize(j.ctx, j.req)
		j.respCh <- result{audio: audio, err: err}
	}
}

func (s *Service) synthesize(ctx context.Context, req Request) (Audio, error) {
	resolvedLang, modelPath, version, matchedModelSelector, err := s.models.ResolvePathAny(req.Lang, req.Voice)
	if err != nil {
		return Audio{}, fmt.Errorf("installed model required: %w", err)
	}
	req.Lang = resolvedLang

	s.loadedMu.Lock()
	loadedVersion := s.loaded[req.Lang]
	if loadedVersion != version {
		if err := s.engine.LoadModel(req.Lang, modelPath); err != nil {
			s.loadedMu.Unlock()
			return Audio{}, err
		}
		s.loaded[req.Lang] = version
	}
	s.loadedMu.Unlock()
	// If voice selected a concrete installed model (id/version), do not
	// pass it down as speaker selector for the model runtime.
	if matchedModelSelector {
		req.Voice = ""
	}
	return s.engine.Synthesize(ctx, req, modelPath)
}
