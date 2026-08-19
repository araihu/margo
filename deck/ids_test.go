package deck

import "testing"

func TestRenderIDAllocatorIsIdempotentAndInjective(t *testing.T) {
	allocator := NewRenderIDAllocator("deck")
	first := allocator.Allocate("table", "slide-0001/table-0")
	if first == "" {
		t.Fatal("allocator returned an empty ID")
	}
	if got := allocator.Allocate("table", "slide-0001/table-0"); got != first {
		t.Fatalf("repeated allocation = %q want %q", got, first)
	}
	second := allocator.Allocate("table", "slide-0002/table-0")
	if second == first || second == "" {
		t.Fatalf("distinct allocation = %q, first = %q", second, first)
	}
	if got, ok := allocator.Resolve("table", "slide-0001/table-0"); !ok || got != first {
		t.Fatalf("resolve = %q, %v", got, ok)
	}
	if _, ok := allocator.Resolve("table", "missing"); ok {
		t.Fatal("missing allocation unexpectedly resolved")
	}
}
