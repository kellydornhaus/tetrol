package xmlui

import (
	"github.com/kellydornhaus/layouter/layout"
	"strings"
	"sync"
)

// BuildFunc builds a component from a parsed XML node.
// It receives the Loader for helpers and to build child nodes.
type BuildFunc func(l *Loader, n *Node) (layout.Component, error)

// Registry maps element names to builder functions, supporting custom tags.
type Registry struct {
	mu       sync.RWMutex
	builders map[string]BuildFunc
}

func NewRegistry() *Registry { return &Registry{builders: make(map[string]BuildFunc)} }

// Register associates an element name with a builder.
// Names are case-insensitive; stored in lower case.
func (r *Registry) Register(name string, fn BuildFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.builders == nil {
		r.builders = make(map[string]BuildFunc)
	}
	r.builders[strings.ToLower(name)] = fn
}

// Lookup returns a builder for a tag, or nil if none.
func (r *Registry) Lookup(name string) BuildFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.builders == nil {
		return nil
	}
	return r.builders[strings.ToLower(name)]
}

// Snapshot returns a shallow copy of the registered builders.
func (r *Registry) Snapshot() map[string]BuildFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make(map[string]BuildFunc, len(r.builders))
	for k, v := range r.builders {
		cp[k] = v
	}
	return cp
}
