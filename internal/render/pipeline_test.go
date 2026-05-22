package render_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-yaml/parser"
	"github.com/home-operations/yamlls/internal/render"
)

type fakeRenderer struct {
	name    string
	matches bool
	out     []byte
	err     error

	mu    sync.Mutex
	calls int
}

func (f *fakeRenderer) Name() string                        { return f.name }
func (f *fakeRenderer) Matches(*render.SourceDocument) bool { return f.matches }
func (f *fakeRenderer) Render(_ context.Context, doc *render.SourceDocument) (*render.RenderedOutput, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	pf, err := parser.ParseBytes(f.out, 0)
	if err != nil {
		return nil, err
	}
	manifests := make([]render.RenderedManifest, 0, len(pf.Docs))
	for _, d := range pf.Docs {
		manifests = append(manifests, render.RenderedManifest{
			AST:  d,
			GVK:  render.GVK{Group: "", Version: "v1", Kind: "Pod"},
			Name: "test",
		})
	}
	return &render.RenderedOutput{
		Provider:  f.name,
		Raw:       f.out,
		Manifests: manifests,
	}, nil
}

type recordingSink struct {
	mu   sync.Mutex
	got  []*render.RenderedOutput
	errs []error
	uris []string
	done chan struct{}
}

func newRecordingSink() *recordingSink { return &recordingSink{done: make(chan struct{}, 16)} }

func (s *recordingSink) Notify(uri string, out *render.RenderedOutput, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uris = append(s.uris, uri)
	s.got = append(s.got, out)
	s.errs = append(s.errs, err)
	s.done <- struct{}{}
}

func TestPipeline_DebouncedRender(t *testing.T) {
	reg := render.NewRegistry()
	fr := &fakeRenderer{
		name:    "fake",
		matches: true,
		out:     []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n"),
	}
	reg.Register(fr)
	sink := newRecordingSink()
	p := render.NewPipeline(reg, sink)
	p.SetDebounce(20 * time.Millisecond)

	doc := &render.SourceDocument{URI: "file:///tmp/x.yaml", Path: "/tmp/x.yaml", Text: "a"}
	p.Schedule(doc)

	select {
	case <-sink.done:
	case <-time.After(2 * time.Second):
		t.Fatalf("pipeline never delivered: calls=%d", fr.calls)
	}

	if fr.calls != 1 {
		t.Errorf("expected 1 render call, got %d", fr.calls)
	}
	if len(sink.got) != 1 || sink.got[0] == nil || len(sink.got[0].Manifests) != 1 {
		t.Errorf("unexpected sink output: %+v", sink.got)
	}
}

func TestPipeline_NoMatchingProvider(t *testing.T) {
	reg := render.NewRegistry()
	reg.Register(&fakeRenderer{name: "fake", matches: false})
	sink := newRecordingSink()
	p := render.NewPipeline(reg, sink)
	p.SetDebounce(10 * time.Millisecond)

	p.Schedule(&render.SourceDocument{URI: "file:///tmp/y.yaml", Text: "z"})

	select {
	case <-sink.done:
		t.Fatalf("sink should not have been notified")
	case <-time.After(150 * time.Millisecond):
	}
}
