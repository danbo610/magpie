package update

import (
	"crypto/rand"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Keep the running image on its original volume, outside the staging tree:
// Stage clears that tree on its next download. Deleting the old image can
// make macOS 13 kill newly forked children before exec, including the shell
// that opens the new version (#2362). Every install gets a name of its own.
func retainedBundle(bundle string) string {
	return filepath.Join(filepath.Dir(bundle), retainedPrefix(bundle)+strconv.Itoa(os.Getpid())+"-"+rand.Text()+".app")
}

func retainedPrefix(bundle string) string { return "." + filepath.Base(bundle) + ".previous-" }

var retainedSuffix = regexp.MustCompile(`^([1-9][0-9]*)-[A-Z2-7]{26,}\.app$`)

// RemoveOldBundles reclaims earlier installations only after their updater
// and any other process using their executable have exited. A failed check
// or removal leaves the copy for the next launch or update.
func RemoveOldBundles(bundle string) {
	for _, old := range unusedBundles(bundle) {
		if err := os.RemoveAll(old); err != nil {
			log.Println("update: keeping the previous app:", err)
		}
	}
}

func unusedBundles(bundle string) []string {
	dir := filepath.Dir(bundle)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var unused []string
	for _, entry := range entries {
		suffix, ok := strings.CutPrefix(entry.Name(), retainedPrefix(bundle))
		if !ok || !entry.IsDir() {
			continue // includes symlinks: never follow one to a different app
		}
		m := retainedSuffix.FindStringSubmatch(suffix)
		if m == nil {
			continue
		}
		pid, err := strconv.Atoi(m[1])
		if err != nil || !bundleOwnerExited(pid) {
			continue
		}
		old := filepath.Join(dir, entry.Name())
		if bundleUnused(old) {
			unused = append(unused, old)
		}
	}
	return unused
}
