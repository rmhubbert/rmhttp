//go:build !race

package benchmarks

// raceEnabled reports that the test binary was built without the race detector, so allocation counts
// reflect the code under test.
const raceEnabled = false
