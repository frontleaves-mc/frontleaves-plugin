package matrix

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseManageIntervalValid(t *testing.T) {
	result := parseManageInterval("2s")
	assert.Equal(t, 2*time.Second, result)
}

func TestParseManageIntervalDefault(t *testing.T) {
	result := parseManageInterval("5s")
	assert.Equal(t, 5*time.Second, result)
}

func TestParseManageIntervalInvalidString(t *testing.T) {
	result := parseManageInterval("not-a-duration")
	assert.Equal(t, 5*time.Second, result)
}

func TestParseManageIntervalZero(t *testing.T) {
	result := parseManageInterval("0s")
	assert.Equal(t, 5*time.Second, result)
}

func TestParseManageIntervalNegative(t *testing.T) {
	result := parseManageInterval("-3s")
	assert.Equal(t, 5*time.Second, result)
}

func TestParseManageIntervalMilliseconds(t *testing.T) {
	result := parseManageInterval("500ms")
	assert.Equal(t, 500*time.Millisecond, result)
}
