package impactscore

import "testing"

func TestCompute_FloorHit_Returns100(t *testing.T) {
	got := Compute(Inputs{
		FloorCategory: "egress",
		FilesChanged:  99,
		LinesChanged:  99999,
	})
	if got != 100 {
		t.Fatalf("Compute() = %d, want 100", got)
	}
}

func TestCompute_NoFloor_FilesAndLines(t *testing.T) {
	got := Compute(Inputs{
		FilesChanged: 5,
		LinesChanged: 200,
	})
	want := 5*5 + 200/10
	if got != want {
		t.Fatalf("Compute() = %d, want %d", got, want)
	}
}

func TestCompute_NoFloor_CapAt99(t *testing.T) {
	got := Compute(Inputs{
		FilesChanged: 1000,
		LinesChanged: 1000000,
	})
	if got != 99 {
		t.Fatalf("Compute() = %d, want 99", got)
	}
}

func TestCompute_NegativeInputs_Clamped(t *testing.T) {
	got := Compute(Inputs{
		FilesChanged: -10,
		LinesChanged: -50,
	})
	if got != 0 {
		t.Fatalf("Compute() = %d, want 0", got)
	}
}

func TestIsHardStop_100_True(t *testing.T) {
	if !IsHardStop(100) {
		t.Fatal("IsHardStop(100) = false, want true")
	}
}

func TestIsHardStop_99_False(t *testing.T) {
	if IsHardStop(99) {
		t.Fatal("IsHardStop(99) = true, want false")
	}
}

func TestCompute_NoFloor_NeverReaches100(t *testing.T) {
	for files := -100; files <= 10000; files += 17 {
		for lines := -100; lines <= 100000; lines += 137 {
			score := Compute(Inputs{
				FilesChanged: files,
				LinesChanged: lines,
			})
			if score >= 100 {
				t.Fatalf("score=%d for files=%d lines=%d without floor", score, files, lines)
			}
		}
	}
}
