//go:build race

package rmhttp

// raceEnabled reports that the test binary was built with the race detector. The detector allocates
// shadow-memory bookkeeping on nearly every memory access, which makes per-request allocation counts
// meaningless, so allocation-budget tests are skipped when it is on.
const raceEnabled = true
