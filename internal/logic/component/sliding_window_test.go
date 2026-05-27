package component

import (
	"testing"
	"time"
)

func TestRingBuffer_PushAndItems(t *testing.T) {
	rb := NewRingBuffer[int](5)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	items := rb.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	if items[0] != 1 || items[1] != 2 || items[2] != 3 {
		t.Fatalf("expected [1, 2, 3], got %v", items)
	}

	if rb.Size() != 3 {
		t.Fatalf("expected size 3, got %d", rb.Size())
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer[int](3)

	// Push 5 items, should only keep last 3
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	items := rb.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Last 3 should be [3, 4, 5]
	if items[0] != 3 || items[1] != 4 || items[2] != 5 {
		t.Fatalf("expected [3, 4, 5], got %v", items)
	}

	if rb.Size() != 3 {
		t.Fatalf("expected size 3, got %d", rb.Size())
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer[int](5)

	items := rb.Items()
	if len(items) != 0 {
		t.Fatalf("expected empty items, got %d items", len(items))
	}

	if rb.Size() != 0 {
		t.Fatalf("expected size 0, got %d", rb.Size())
	}

	if rb.Cap() != 5 {
		t.Fatalf("expected capacity 5, got %d", rb.Cap())
	}
}

func TestRingBuffer_Cap(t *testing.T) {
	rb := NewRingBuffer[int](10)

	if rb.Cap() != 10 {
		t.Fatalf("expected capacity 10, got %d", rb.Cap())
	}

	rb.Push(1)
	rb.Push(2)

	if rb.Cap() != 10 {
		t.Fatalf("expected capacity 10 after push, got %d", rb.Cap())
	}
}

func TestSlidingWindow_AddAndItems(t *testing.T) {
	sw := NewSlidingWindow[int](10, 1000) // 1 second window

	now := time.Now().UnixMilli()
	sw.Add(now-500, 1)
	sw.Add(now-300, 2)
	sw.Add(now-100, 3)

	items := sw.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items within window, got %d", len(items))
	}

	if items[0] != 1 || items[1] != 2 || items[2] != 3 {
		t.Fatalf("expected [1, 2, 3], got %v", items)
	}
}

func TestSlidingWindow_ExpireOld(t *testing.T) {
	sw := NewSlidingWindow[int](10, 1000) // 1 second window

	// Use fixed timestamp as the current time
	now := time.Now().UnixMilli()

	// Add items, some outside the window
	sw.Add(now-1100, 1) // outside window (>1000ms ago)
	sw.Add(now-900, 2)  // inside window
	sw.Add(now-800, 3)  // inside window
	sw.Add(now-600, 4)  // inside window
	sw.Add(now-400, 5)  // inside window
	sw.Add(now-200, 6)  // inside window
	sw.Add(now-50, 7)   // inside window

	// Items() uses current time to filter by windowMs
	items := sw.Items()
	if len(items) != 6 {
		t.Fatalf("expected 6 items within window (2,3,4,5,6,7), got %d", len(items))
	}

	// Should contain items 2, 3, 4, 5, 6, 7
	if items[0] != 2 || items[1] != 3 || items[2] != 4 || items[3] != 5 || items[4] != 6 || items[5] != 7 {
		t.Fatalf("expected [2, 3, 4, 5, 6, 7], got %v", items)
	}
}

func TestSlidingWindow_EmptyWindow(t *testing.T) {
	sw := NewSlidingWindow[int](10, 1000)

	items := sw.Items()
	if len(items) != 0 {
		t.Fatalf("expected empty items, got %d items", len(items))
	}

	itemsSince := sw.ItemsSince(time.Now().UnixMilli() - 500)
	if len(itemsSince) != 0 {
		t.Fatalf("expected empty itemsSince, got %d items", len(itemsSince))
	}
}

func TestSlidingWindow_ItemsSince(t *testing.T) {
	sw := NewSlidingWindow[int](10, 1000)

	now := time.Now().UnixMilli()
	sw.Add(now-900, 1)
	sw.Add(now-700, 2)
	sw.Add(now-500, 3)
	sw.Add(now-300, 4)
	sw.Add(now-100, 5)

	// Get items since now - 600 (should include 3, 4, 5)
	items := sw.ItemsSince(now - 600)
	if len(items) != 3 {
		t.Fatalf("expected 3 items since now-600, got %d", len(items))
	}

	if items[0] != 3 || items[1] != 4 || items[2] != 5 {
		t.Fatalf("expected [3, 4, 5], got %v", items)
	}

	// Get items since now - 200 (should include 5 only)
	items = sw.ItemsSince(now - 200)
	if len(items) != 1 {
		t.Fatalf("expected 1 item since now-200, got %d", len(items))
	}

	if items[0] != 5 {
		t.Fatalf("expected [5], got %v", items)
	}
}

func TestSlidingWindow_ItemTypeString(t *testing.T) {
	sw := NewSlidingWindow[string](10, 1000)

	now := time.Now().UnixMilli()
	sw.Add(now-500, "first")
	sw.Add(now-300, "second")
	sw.Add(now-100, "third")

	items := sw.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	if items[0] != "first" || items[1] != "second" || items[2] != "third" {
		t.Fatalf("expected [first, second, third], got %v", items)
	}
}