package component

// VLTracker tracks violation levels with decay and flagging thresholds.
// NOT thread-safe — caller must protect with mutex.
type VLTracker struct {
	violations      float64
	decayRate       float64
	setbackThreshold float64
	maxViolations   float64
}

// NewVLTracker creates a new VLTracker with the specified decay rate and setback threshold.
// maxViolations is capped at 100.0 as a reasonable upper bound.
func NewVLTracker(decayRate, setbackThreshold float64) *VLTracker {
	return &VLTracker{
		violations:      0,
		decayRate:       decayRate,
		setbackThreshold: setbackThreshold,
		maxViolations:   100.0,
	}
}

// Flag adds the specified amount to the violations count, capped at maxViolations.
func (v *VLTracker) Flag(amount float64) {
	v.violations += amount
	if v.violations > v.maxViolations {
		v.violations = v.maxViolations
	}
}

// Reward reduces the violations count by the decayRate, floored at 0.
func (v *VLTracker) Reward() {
	v.violations -= v.decayRate
	if v.violations < 0 {
		v.violations = 0
	}
}

// ShouldFlag returns true if violations exceed the setback threshold (strictly greater).
func (v *VLTracker) ShouldFlag() bool {
	return v.violations > v.setbackThreshold
}

// Violations returns the current violation count.
func (v *VLTracker) Violations() float64 {
	return v.violations
}

// Reset sets the violation count to 0.
func (v *VLTracker) Reset() {
	v.violations = 0
}