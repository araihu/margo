package deck

import (
	"fmt"
	"strings"
	"sync"

	"github.com/araihu/margo"
)

// IDAllocator is a deterministic, render-wide HTML identity registry.
type IDAllocator struct {
	mu        sync.Mutex
	namespace string
	ids       map[string]string
	ordinal   uint64
}

// NewRenderIDAllocator creates an allocator in a sanitized namespace.
func NewRenderIDAllocator(namespace string) *IDAllocator {
	return &IDAllocator{namespace: sanitizeIDPart(namespace, "deck"), ids: make(map[string]string)}
}

func (allocator *IDAllocator) Allocate(kind, sourceKey string) string {
	if allocator == nil || strings.TrimSpace(kind) == "" || strings.TrimSpace(sourceKey) == "" {
		return ""
	}
	key := kind + "\x00" + sourceKey
	allocator.mu.Lock()
	defer allocator.mu.Unlock()
	if id, ok := allocator.ids[key]; ok {
		return id
	}
	allocator.ordinal++
	id := fmt.Sprintf("%s-%s-%08d", allocator.namespace, sanitizeIDPart(kind, "id"), allocator.ordinal)
	allocator.ids[key] = id
	return id
}

func (allocator *IDAllocator) Resolve(kind, sourceKey string) (string, bool) {
	if allocator == nil || strings.TrimSpace(kind) == "" || strings.TrimSpace(sourceKey) == "" {
		return "", false
	}
	allocator.mu.Lock()
	defer allocator.mu.Unlock()
	id, ok := allocator.ids[kind+"\x00"+sourceKey]
	return id, ok
}

type scopedIDAllocator struct {
	root  margo.RenderIDAllocator
	scope string
}

func (allocator scopedIDAllocator) Allocate(kind, sourceKey string) string {
	return allocator.root.Allocate(kind, allocator.scope+"\x00"+sourceKey)
}

func (allocator scopedIDAllocator) Resolve(kind, sourceKey string) (string, bool) {
	return allocator.root.Resolve(kind, allocator.scope+"\x00"+sourceKey)
}

func sanitizeIDPart(value, fallback string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(value) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return fallback
	}
	return result
}
