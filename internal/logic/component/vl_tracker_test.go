package component

import (
	"testing"
)

// TestVLTracker_FlagGrowth tests that Flag adds violations correctly.
func TestVLTracker_FlagGrowth(t *testing.T) {
	tracker := NewVLTracker(1.0, 10.0)
	tracker.Flag(2)
	tracker.Flag(2)
	tracker.Flag(2)

	if tracker.Violations() != 6.0 {
		t.Errorf("expected violations = 6.0, got %.1f", tracker.Violations())
	}
}

// TestVLTracker_RewardDecay tests that Reward reduces violations by decayRate.
func TestVLTracker_RewardDecay(t *testing.T) {
	tracker := NewVLTracker(1.0, 10.0)
	tracker.Flag(5)
	tracker.Reward()

	if tracker.Violations() != 4.0 {
		t.Errorf("expected violations = 4.0 after reward, got %.1f", tracker.Violations())
	}
}

// TestVLTracker_RewardDecayToZero tests that Reward floors violations at 0.
func TestVLTracker_RewardDecayToZero(t *testing.T) {
	tracker := NewVLTracker(5.0, 10.0)
	tracker.Flag(1)
	tracker.Reward()

	if tracker.Violations() != 0 {
		t.Errorf("expected violations = 0 after floor, got %.1f", tracker.Violations())
	}
}

// TestVLTracker_ShouldFlag tests that ShouldFlag returns true when violations exceed threshold.
func TestVLTracker_ShouldFlag(t *testing.T) {
	tracker := NewVLTracker(1.0, 3.0)
	tracker.Flag(2)
	tracker.Flag(2)

	if !tracker.ShouldFlag() {
		t.Errorf("expected ShouldFlag = true with violations %.1f > threshold 3.0", tracker.Violations())
	}
}

// TestVLTracker_ShouldFlagFalse tests that ShouldFlag returns false when violations do not exceed threshold.
func TestVLTracker_ShouldFlagFalse(t *testing.T) {
	tracker := NewVLTracker(1.0, 5.0)
	tracker.Flag(2)
	tracker.Flag(2)

	if tracker.ShouldFlag() {
		t.Errorf("expected ShouldFlag = false with violations %.1f not > threshold 5.0", tracker.Violations())
	}
}

// TestVLTracker_ShouldFlagStrict tests that ShouldFlag is strictly greater (not >=).
func TestVLTracker_ShouldFlagStrict(t *testing.T) {
	tracker := NewVLTracker(1.0, 4.0)
	tracker.Flag(4)

	if tracker.ShouldFlag() {
		t.Errorf("expected ShouldFlag = false with violations %.1f == threshold 4.0 (strictly greater)", tracker.Violations())
	}
}

// TestVLTracker_MaxViolations tests that violations are capped at maxViolations (100.0).
func TestVLTracker_MaxViolations(t *testing.T) {
	tracker := NewVLTracker(1.0, 10.0)
	tracker.Flag(200)

	if tracker.Violations() != 100.0 {
		t.Errorf("expected violations capped at 100.0, got %.1f", tracker.Violations())
	}
}

// TestVLTracker_Reset tests that Reset sets violations to 0.
func TestVLTracker_Reset(t *testing.T) {
	tracker := NewVLTracker(1.0, 10.0)
	tracker.Flag(10)
	tracker.Reset()

	if tracker.Violations() != 0 {
		t.Errorf("expected violations = 0 after reset, got %.1f", tracker.Violations())
	}
}

// TestVLTracker_FlagRewardAlternating tests alternating Flag and Reward operations.
func TestVLTracker_FlagRewardAlternating(t *testing.T) {
	tracker := NewVLTracker(2.0, 5.0)
	tracker.Flag(3)
	if tracker.Violations() != 3.0 {
		t.Errorf("step 1: expected violations = 3.0, got %.1f", tracker.Violations())
	}

	tracker.Reward()
	if tracker.Violations() != 1.0 {
		t.Errorf("step 2: expected violations = 1.0, got %.1f", tracker.Violations())
	}

	tracker.Flag(1)
	if tracker.Violations() != 2.0 {
		t.Errorf("step 3: expected violations = 2.0, got %.1f", tracker.Violations())
	}

	if tracker.ShouldFlag() {
		t.Errorf("expected ShouldFlag = false with violations %.1f not > threshold 5.0", tracker.Violations())
	}
}

// TestVLTracker_ViolationsZeroAfterCreation tests that a new tracker starts with 0 violations.
func TestVLTracker_ViolationsZeroAfterCreation(t *testing.T) {
	tracker := NewVLTracker(1.0, 10.0)

	if tracker.Violations() != 0 {
		t.Errorf("expected initial violations = 0, got %.1f", tracker.Violations())
	}
}

// TestVLTracker_RewardOnZero tests that Reward does not go negative starting from 0.
func TestVLTracker_RewardOnZero(t *testing.T) {
	tracker := NewVLTracker(5.0, 10.0)
	tracker.Reward()

	if tracker.Violations() != 0 {
		t.Errorf("expected violations = 0 after reward from 0, got %.1f", tracker.Violations())
	}
}