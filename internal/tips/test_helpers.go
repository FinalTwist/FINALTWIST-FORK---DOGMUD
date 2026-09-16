package tips

// SeedForTest replaces the store, resets the rotation, and returns a restore
// func to defer. Intended for cross-package tests (hooks, narration).
func SeedForTest(t []string) func() {
	oldAll, oldNext := all, next
	all, next = t, 0
	return func() { all, next = oldAll, oldNext }
}
