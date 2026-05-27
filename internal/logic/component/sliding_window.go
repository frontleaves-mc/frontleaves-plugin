package component

import (
	"time"
)

// RingBuffer is a fixed-size circular buffer for generic types.
// It overwrites the oldest element when full.
type RingBuffer[T any] struct {
	buf   []T
	cap   int
	head  int // next write position
	count int // current number of elements
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf:  make([]T, capacity),
		cap:  capacity,
		head: 0,
	}
}

// Push adds an item to the ring buffer. If full, it overwrites the oldest item.
func (rb *RingBuffer[T]) Push(item T) {
	// If buffer is full, we're overwriting an element, count stays at cap
	// If not full, increment count
	if rb.count < rb.cap {
		rb.count++
	}
	rb.buf[rb.head] = item
	rb.head = (rb.head + 1) % rb.cap
}

// Items returns all items in chronological order (oldest first).
func (rb *RingBuffer[T]) Items() []T {
	if rb.count == 0 {
		return []T{}
	}

	result := make([]T, rb.count)
	for i := 0; i < rb.count; i++ {
		// The oldest element is at position (head - count) mod cap
		// When count == cap, we return all elements starting from head
		// When count < cap, the first element is at 0
		pos := (rb.head - rb.count + i) % rb.cap
		if pos < 0 {
			pos += rb.cap
		}
		result[i] = rb.buf[pos]
	}
	return result
}

// Size returns the current number of elements in the buffer.
func (rb *RingBuffer[T]) Size() int {
	return rb.count
}

// Cap returns the maximum capacity of the buffer.
func (rb *RingBuffer[T]) Cap() int {
	return rb.cap
}

// windowItem is an internal struct for SlidingWindow to store items with timestamps.
type windowItem[T any] struct {
	timestamp int64
	data      T
}

// SlidingWindow is a time-based sliding window for generic types.
// It stores items with timestamps and returns only items within the time window.
type SlidingWindow[T any] struct {
	rb        *RingBuffer[windowItem[T]]
	windowMs  int64
}

// NewSlidingWindow creates a new sliding window with the specified capacity and time window.
func NewSlidingWindow[T any](capacity int, windowMs int64) *SlidingWindow[T] {
	return &SlidingWindow[T]{
		rb:       NewRingBuffer[windowItem[T]](capacity),
		windowMs: windowMs,
	}
}

// Add adds an item with its timestamp to the sliding window.
func (sw *SlidingWindow[T]) Add(timestamp int64, item T) {
	sw.rb.Push(windowItem[T]{
		timestamp: timestamp,
		data:      item,
	})
}

// Items returns all items within the time window.
func (sw *SlidingWindow[T]) Items() []T {
	now := time.Now().UnixMilli()
	return sw.ItemsSince(now - sw.windowMs)
}

// ItemsSince returns all items with timestamp >= sinceMs.
func (sw *SlidingWindow[T]) ItemsSince(sinceMs int64) []T {
	items := sw.rb.Items()
	result := make([]T, 0, len(items))
	for _, item := range items {
		if item.timestamp >= sinceMs {
			result = append(result, item.data)
		}
	}
	return result
}