package rng

import (
	"math"
	"testing"
)

const epsilon = 1e-15

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// TestMulberry32CrossLanguage verifies seed 12345 produces the exact sequence
// from the TypeScript implementation.
func TestMulberry32CrossLanguage(t *testing.T) {
	expected := []float64{
		0.9797282677609473,
		0.3067522644996643,
		0.484205421525985,
		0.817934412509203,
		0.5094283693470061,
		0.34747186047025025,
		0.07375754183158278,
		0.7663964673411101,
		0.9968264393974096,
		0.8250224851071835,
	}

	r := New(12345)
	for i, want := range expected {
		got := r.Next()
		if !almostEqual(got, want) {
			t.Errorf("Next()[%d]: got %.16f, want %.16f (diff=%e)", i, got, want, math.Abs(got-want))
		}
	}
}

// TestCrossLanguageSeed42 verifies seed 42 first 3 values and rollDie(6) returns 4.
func TestCrossLanguageSeed42(t *testing.T) {
	// rollDie(6) with seed 42 should return 4
	r := New(42)
	die := r.RollDie(6)
	if die != 4 {
		t.Errorf("seed 42 RollDie(6): got %d, want 4", die)
	}

	// First 3 next() calls after resetting
	expected := []float64{
		0.6011037519201636,
		0.44829055899754167,
		0.8524657934904099,
	}
	r2 := New(42)
	for i, want := range expected {
		got := r2.Next()
		if !almostEqual(got, want) {
			t.Errorf("seed 42 Next()[%d]: got %.16f, want %.16f (diff=%e)", i, got, want, math.Abs(got-want))
		}
	}
}

// TestCrossLanguageRollDice verifies seed 42 rollDice(3,6) returns 13.
func TestCrossLanguageRollDice(t *testing.T) {
	r := New(42)
	result := r.RollDice(3, 6)
	if result != 13 {
		t.Errorf("seed 42 RollDice(3,6): got %d, want 13", result)
	}
}

// TestCrossLanguageDieSequence verifies seed 99 first 20 d6 rolls match exactly.
func TestCrossLanguageDieSequence(t *testing.T) {
	expected := []int{2, 5, 4, 5, 1, 5, 1, 1, 3, 5, 3, 1, 3, 5, 2, 2, 3, 5, 3, 2}
	r := New(99)
	for i, want := range expected {
		got := r.RollDie(6)
		if got != want {
			t.Errorf("seed 99 RollDie(6)[%d]: got %d, want %d", i, got, want)
		}
	}
}

// TestDeterminism verifies same seed produces same sequence over 100 values.
func TestDeterminism(t *testing.T) {
	r1 := New(777)
	r2 := New(777)

	for i := 0; i < 100; i++ {
		v1 := r1.Next()
		v2 := r2.Next()
		if v1 != v2 {
			t.Errorf("determinism failure at index %d: %f != %f", i, v1, v2)
		}
	}
}

// TestRollDie verifies d6 values are always in [1,6].
func TestRollDie(t *testing.T) {
	r := New(1)
	for i := 0; i < 1000; i++ {
		v := r.RollDie(6)
		if v < 1 || v > 6 {
			t.Errorf("RollDie(6) out of range at index %d: %d", i, v)
		}
	}
}

// TestRollDice verifies 3d6 values are always in [3,18].
func TestRollDice(t *testing.T) {
	r := New(2)
	for i := 0; i < 1000; i++ {
		v := r.RollDice(3, 6)
		if v < 3 || v > 18 {
			t.Errorf("RollDice(3,6) out of range at index %d: %d", i, v)
		}
	}
}

// TestRollNotation parses and rolls various dice notations.
func TestRollNotation(t *testing.T) {
	tests := []struct {
		notation string
		min, max int
	}{
		{"d6", 1, 6},
		{"2d8", 2, 16},
		{"3d6", 3, 18},
		{"d20", 1, 20},
		{"10d4", 10, 40},
	}

	r := New(3)
	for _, tc := range tests {
		for i := 0; i < 100; i++ {
			v := r.RollNotation(tc.notation)
			if v < tc.min || v > tc.max {
				t.Errorf("RollNotation(%q) out of range [%d,%d]: got %d", tc.notation, tc.min, tc.max, v)
			}
		}
	}
}

// TestParseDice verifies parsing of various dice notations.
func TestParseDice(t *testing.T) {
	tests := []struct {
		notation      string
		count, sides  int
		expectError   bool
	}{
		{"d6", 1, 6, false},
		{"2d8", 2, 8, false},
		{"3d6", 3, 6, false},
		{"d20", 1, 20, false},
		{"10d4", 10, 4, false},
		{"invalid", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tc := range tests {
		count, sides, err := ParseDice(tc.notation)
		if tc.expectError {
			if err == nil {
				t.Errorf("ParseDice(%q): expected error but got none", tc.notation)
			}
		} else {
			if err != nil {
				t.Errorf("ParseDice(%q): unexpected error: %v", tc.notation, err)
			}
			if count != tc.count {
				t.Errorf("ParseDice(%q) count: got %d, want %d", tc.notation, count, tc.count)
			}
			if sides != tc.sides {
				t.Errorf("ParseDice(%q) sides: got %d, want %d", tc.notation, sides, tc.sides)
			}
		}
	}
}

// TestPick verifies Pick selects from a slice and covers most items over 100 trials.
func TestPick(t *testing.T) {
	items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	seen := make(map[int]bool)
	r := New(4)

	for i := 0; i < 100; i++ {
		v := Pick(r, items)
		seen[v] = true
	}

	// Expect to see most values over 100 trials
	if len(seen) < 8 {
		t.Errorf("Pick covered only %d of %d items over 100 trials", len(seen), len(items))
	}

	// Verify all picked values are from the slice
	for v := range seen {
		found := false
		for _, item := range items {
			if item == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Pick returned value %d not in items", v)
		}
	}
}

// TestShuffle verifies Shuffle preserves elements (same sum).
func TestShuffle(t *testing.T) {
	original := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	items := make([]int, len(original))
	copy(items, original)

	originalSum := 0
	for _, v := range original {
		originalSum += v
	}

	r := New(5)
	Shuffle(r, items)

	shuffledSum := 0
	for _, v := range items {
		shuffledSum += v
	}

	if shuffledSum != originalSum {
		t.Errorf("Shuffle changed sum: got %d, want %d", shuffledSum, originalSum)
	}

	// Verify all original elements are present
	counts := make(map[int]int)
	for _, v := range original {
		counts[v]++
	}
	for _, v := range items {
		counts[v]--
	}
	for v, c := range counts {
		if c != 0 {
			t.Errorf("Shuffle lost element %d (count diff: %d)", v, c)
		}
	}
}

// TestChance verifies Chance(0.5) over 1000 trials gives approximately 500 trues.
func TestChance(t *testing.T) {
	r := New(6)
	trues := 0
	trials := 1000

	for i := 0; i < trials; i++ {
		if r.Chance(0.5) {
			trues++
		}
	}

	// Allow +/- 10% tolerance (400-600 range)
	if trues < 400 || trues > 600 {
		t.Errorf("Chance(0.5) over %d trials: got %d trues, expected ~500 (±100)", trials, trues)
	}
}
