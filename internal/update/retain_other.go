//go:build !darwin

package update

// Only macOS installs app bundles. Other platforms replace a bare binary.
func bundleOwnerExited(int) bool { return false }
func bundleUnused(string) bool   { return false }
