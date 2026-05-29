package logic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateAverage(t *testing.T) {
	t.Run("normal average", func(t *testing.T) {
		assert.Equal(t, 8.75, calculateAverage(35, 4))
	})
	t.Run("integer average", func(t *testing.T) {
		assert.Equal(t, 25.0, calculateAverage(100, 4))
	})
	t.Run("zero heartbeat", func(t *testing.T) {
		assert.Equal(t, 0.0, calculateAverage(0, 0))
	})
	t.Run("single heartbeat", func(t *testing.T) {
		assert.Equal(t, 20.0, calculateAverage(20, 1))
	})
	t.Run("float precision", func(t *testing.T) {
		assert.InDelta(t, 3.5, calculateAverage(10.5, 3), 0.0001)
	})
	t.Run("negative count", func(t *testing.T) {
		assert.Equal(t, 0.0, calculateAverage(10, -1))
	})
}
