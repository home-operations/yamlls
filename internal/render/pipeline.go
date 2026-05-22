package render

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type Pipeline struct {
	registry *Registry
	sink     Sink
	debounce time.Duration

	mu      sync.Mutex
	pending map[string]*pending
	cache   map[string]cacheEntry
}

type pending struct {
	timer  *time.Timer
	cancel context.CancelFunc
}

type cacheEntry struct {
	out *RenderedOutput
	err error
}

// Sink.Notify runs on the pipeline's goroutine — implementations must be
// non-blocking.
type Sink interface {
	Notify(uri string, out *RenderedOutput, err error)
}

func NewPipeline(reg *Registry, sink Sink) *Pipeline {
	return &Pipeline{
		registry: reg,
		sink:     sink,
		debounce: 750 * time.Millisecond,
		pending:  make(map[string]*pending),
		cache:    make(map[string]cacheEntry),
	}
}

func (p *Pipeline) SetDebounce(d time.Duration) { p.debounce = d }

func (p *Pipeline) Schedule(doc *SourceDocument) {
	if doc == nil {
		return
	}
	r := p.registry.For(doc)
	if r == nil {
		return
	}
	key := cacheKey(doc.URI, doc.Text)
	p.mu.Lock()
	if hit, ok := p.cache[key]; ok {
		p.mu.Unlock()
		p.sink.Notify(doc.URI, hit.out, hit.err)
		return
	}
	if old := p.pending[doc.URI]; old != nil {
		old.timer.Stop()
		if old.cancel != nil {
			old.cancel()
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t := time.AfterFunc(p.debounce, func() {
		out, err := r.Render(ctx, doc)
		p.mu.Lock()
		p.cache[key] = cacheEntry{out: out, err: err}
		delete(p.pending, doc.URI)
		p.mu.Unlock()
		p.sink.Notify(doc.URI, out, err)
	})
	p.pending[doc.URI] = &pending{timer: t, cancel: cancel}
	p.mu.Unlock()
}

func (p *Pipeline) Latest(uri, text string) (*RenderedOutput, bool) {
	key := cacheKey(uri, text)
	p.mu.Lock()
	defer p.mu.Unlock()
	if hit, ok := p.cache[key]; ok && hit.err == nil {
		return hit.out, true
	}
	return nil, false
}

func (p *Pipeline) Cancel(uri string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if old := p.pending[uri]; old != nil {
		old.timer.Stop()
		if old.cancel != nil {
			old.cancel()
		}
		delete(p.pending, uri)
	}
}

func cacheKey(uri, text string) string {
	h := sha256.Sum256([]byte(text))
	return uri + "@" + hex.EncodeToString(h[:8])
}
