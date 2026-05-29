package schema

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/home-operations/yamlls/internal/uri"
	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
)

// negativeTTL keeps CRDs missing from the configured mirror from causing
// one network round-trip per keystroke.
const negativeTTL = 5 * time.Minute

type Store struct {
	mu       sync.Mutex
	compiled map[string]*jsonschema.Schema
	failures map[string]failure
	inflight map[string]*inflightCompile
}

type failure struct {
	err error
	at  time.Time
}

// inflightCompile lets concurrent Get callers for the same key share a
// single compile (and its network fetch) instead of racing to fetch the
// same schema in parallel.
type inflightCompile struct {
	done chan struct{}
	sch  *jsonschema.Schema
	err  error
}

func NewStore() *Store {
	InstallDiskLoader()
	installEmbeddedLoader()
	return &Store{
		compiled: make(map[string]*jsonschema.Schema),
		failures: make(map[string]failure),
		inflight: make(map[string]*inflightCompile),
	}
}

func (s *Store) Get(ref, docPath string) (*jsonschema.Schema, error) {
	key, err := absRef(ref, docPath)
	if err != nil {
		return nil, fmt.Errorf("resolve schema ref %q: %w", ref, err)
	}
	s.mu.Lock()
	if sch, ok := s.compiled[key]; ok {
		s.mu.Unlock()
		return sch, nil
	}
	if f, ok := s.failures[key]; ok && time.Since(f.at) < negativeTTL {
		s.mu.Unlock()
		return nil, f.err
	}
	// Coalesce concurrent compiles of the same key: a single fetch serves
	// every caller, so a 806-file run doesn't fire one network round-trip
	// per file that shares a schema.
	if call, ok := s.inflight[key]; ok {
		s.mu.Unlock()
		<-call.done
		return call.sch, call.err
	}
	call := &inflightCompile{done: make(chan struct{})}
	s.inflight[key] = call
	s.mu.Unlock()

	// Compile outside the lock: it may fetch the schema over the network,
	// and holding the mutex would stall every other schema lookup behind a
	// single slow host.
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	c.ExtractAnnotations = true
	sch, err := c.Compile(key)

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inflight, key)
	if err != nil {
		call.err = fmt.Errorf("compile schema %s: %w", key, err)
		s.failures[key] = failure{err: call.err, at: time.Now()}
	} else {
		call.sch = sch
		s.compiled[key] = sch
	}
	close(call.done)
	return call.sch, call.err
}

func absRef(ref, docPath string) (string, error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") ||
		strings.HasPrefix(ref, "file://") || strings.HasPrefix(ref, "embedded://") {
		return ref, nil
	}
	if filepath.IsAbs(ref) {
		return uri.FromPath(ref), nil
	}
	if docPath == "" {
		return "", fmt.Errorf("relative schema path %q has no document anchor", ref)
	}
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(docPath), ref))
	if err != nil {
		return "", err
	}
	return uri.FromPath(abs), nil
}
