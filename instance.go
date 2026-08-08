package margo

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type InstanceAllocator struct {
	mu   sync.Mutex
	next uint64
}

func NewInstanceAllocator() *InstanceAllocator {
	return &InstanceAllocator{}
}

func (a *InstanceAllocator) Next() (RenderInstanceID, error) {
	if a == nil {
		return "", runtimeDiagnostic("runtime.instance_allocator_invalid", "instance allocator is nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ordinal := strconv.FormatUint(a.next, 36)
	if len(ordinal) > 32 {
		return "", runtimeDiagnostic("runtime.instance_exhausted", "instance allocator exhausted the v1 ID grammar")
	}
	a.next++
	if len(ordinal) < 8 {
		ordinal = strings.Repeat("0", 8-len(ordinal)) + ordinal
	}
	value := RenderInstanceID(fmt.Sprintf("ri-%s", ordinal))
	if err := ValidateRenderInstanceID(value); err != nil {
		return "", err
	}
	return value, nil
}

type InstanceRegistry struct {
	mu       sync.Mutex
	reserved map[RenderInstanceID]struct{}
}

func NewInstanceRegistry() *InstanceRegistry {
	return &InstanceRegistry{reserved: make(map[RenderInstanceID]struct{})}
}

func (r *InstanceRegistry) Reserve(value RenderInstanceID) error {
	if r == nil {
		return runtimeDiagnostic("runtime.instance_registry_invalid", "instance registry is nil")
	}
	if err := ValidateRenderInstanceID(value); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reserved == nil {
		r.reserved = make(map[RenderInstanceID]struct{})
	}
	if _, exists := r.reserved[value]; exists {
		return runtimeDiagnostic("runtime.instance_duplicate", "render instance ID is already registered")
	}
	r.reserved[value] = struct{}{}
	return nil
}
