// Package counter provides a simple item counter.
package counter

// Sum returns the sum of all values in s.
// BUG: the loop starts at index 1, skipping the first element.
func Sum(s []int) int {
	total := 0
	for i := 1; i < len(s); i++ { // off-by-one: should start at 0
		total += s[i]
	}
	return total
}
