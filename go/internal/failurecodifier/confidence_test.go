package failurecodifier

import "testing"

func TestConfidence_ThreeOccurrencesMedium(t *testing.T) {
	if got := ConfidenceFromCount(ConfidenceThresholdMedium); got != ConfidenceMedium {
		t.Fatalf("ConfidenceFromCount(3) = %q, want %q", got, ConfidenceMedium)
	}
	if got := ConfidenceFromCount(4); got != ConfidenceMedium {
		t.Fatalf("ConfidenceFromCount(4) = %q, want %q", got, ConfidenceMedium)
	}
}

func TestConfidence_FiveOccurrencesHigh(t *testing.T) {
	if got := ConfidenceFromCount(ConfidenceThresholdHigh); got != ConfidenceHigh {
		t.Fatalf("ConfidenceFromCount(5) = %q, want %q", got, ConfidenceHigh)
	}
	if got := ConfidenceFromCount(10); got != ConfidenceHigh {
		t.Fatalf("ConfidenceFromCount(10) = %q, want %q", got, ConfidenceHigh)
	}
}

func TestConfidence_BelowThreeIsLow(t *testing.T) {
	if got := ConfidenceFromCount(2); got != ConfidenceLow {
		t.Fatalf("ConfidenceFromCount(2) = %q, want %q", got, ConfidenceLow)
	}
}
