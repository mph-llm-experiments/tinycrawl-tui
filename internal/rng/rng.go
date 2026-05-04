package rng

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

var diceRegex = regexp.MustCompile(`^(\d*)d(\d+)$`)

// ParseDice parses dice notation like "d6", "2d8", "3d6".
func ParseDice(notation string) (count, sides int, err error) {
	m := diceRegex.FindStringSubmatch(notation)
	if m == nil {
		return 0, 0, fmt.Errorf("invalid dice notation: %s", notation)
	}
	if m[1] == "" {
		count = 1
	} else {
		count, _ = strconv.Atoi(m[1])
	}
	sides, _ = strconv.Atoi(m[2])
	return count, sides, nil
}

// Rng is a Mulberry32 seeded PRNG. It produces the same sequence as
// the TypeScript implementation in src/lib/rng.ts for the same seed.
type Rng struct {
	state int32
}

// New creates a new Rng with the given seed.
func New(seed int) *Rng {
	return &Rng{state: int32(seed)}
}

// Next returns a float64 in [0, 1). Advances internal state.
// This is a direct port of the TypeScript mulberry32 implementation.
// Uses unsigned right shifts (>>>) matching JavaScript semantics.
func (r *Rng) Next() float64 {
	r.state += 0x6d2b79f5
	t := r.state
	t = imul(t^int32(uint32(t)>>15), t|1)
	t ^= t + imul(t^int32(uint32(t)>>7), t|61)
	return float64(uint32(t^int32(uint32(t)>>14))) / 4294967296.0
}

// imul performs 32-bit integer multiplication matching JavaScript's Math.imul.
func imul(a, b int32) int32 {
	return int32(int64(a) * int64(b))
}

// RollDie rolls a single die with the given number of sides.
func (r *Rng) RollDie(sides int) int {
	return int(math.Floor(r.Next()*float64(sides))) + 1
}

// RollDice rolls count dice with the given sides and sums them.
func (r *Rng) RollDice(count, sides int) int {
	total := 0
	for i := 0; i < count; i++ {
		total += r.RollDie(sides)
	}
	return total
}

// RollNotation parses dice notation and rolls. Panics on invalid notation.
func (r *Rng) RollNotation(notation string) int {
	count, sides, err := ParseDice(notation)
	if err != nil {
		panic(err)
	}
	return r.RollDice(count, sides)
}

// Chance returns true with the given probability (0-1).
func (r *Rng) Chance(probability float64) bool {
	return r.Next() < probability
}

// GetState returns the current state for serialization.
func (r *Rng) GetState() int {
	return int(r.state)
}

// Pick returns a random element from a slice.
func Pick[T any](r *Rng, items []T) T {
	return items[int(math.Floor(r.Next()*float64(len(items))))]
}

// Shuffle shuffles a slice in place using Fisher-Yates.
func Shuffle[T any](r *Rng, items []T) {
	for i := len(items) - 1; i > 0; i-- {
		j := int(math.Floor(r.Next() * float64(i+1)))
		items[i], items[j] = items[j], items[i]
	}
}
