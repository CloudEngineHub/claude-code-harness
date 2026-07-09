package failurecodifier

// Confidence thresholds (Phase 100.1.3 literals — do not externalize).
const (
	ConfidenceThresholdMedium = 3
	ConfidenceThresholdHigh   = 5
)

const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

// ConfidenceFromCount maps occurrence count to failure-rule.v1 confidence.
// count >= 5 → high, count >= 3 → medium, else low.
func ConfidenceFromCount(count int) string {
	if count >= ConfidenceThresholdHigh {
		return ConfidenceHigh
	}
	if count >= ConfidenceThresholdMedium {
		return ConfidenceMedium
	}
	return ConfidenceLow
}
