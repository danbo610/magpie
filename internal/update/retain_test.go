package update

import (
	"os"
	"path/filepath"
	"testing"
)

// #2362: a native Mac app can no longer fork its relaunch shell after its
// executable is unlinked. Staging another update must not unlink it either.
func TestInstallKeepsRunningBundleOutsideStage(t *testing.T) {
	testInstallKeepsRunningBundle(t, Install)
}

func testInstallKeepsRunningBundle(t *testing.T, install func(string, string) error) {
	t.Helper()
	dir := t.TempDir()
	bundle := filepath.Join(dir, "magpie.app")
	stage := filepath.Join(dir, ".magpie-update")
	staged := filepath.Join(stage, "app", "magpie.app")
	app(t, bundle, "running")
	for _, next := range []string{"new", "newer"} {
		app(t, staged, next)
		if err := install(staged, bundle); err != nil {
			t.Fatal(err)
		}
		if version(t, bundle) != next {
			t.Fatal("the new app isn't installed")
		}
		// Stage starts by removing this directory, including on retries.
		if err := os.RemoveAll(stage); err != nil {
			t.Fatal(err)
		}
		var running string
		if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == "v" {
				if b, err := os.ReadFile(path); err == nil && string(b) == "running" {
					running = path
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if running == "" {
			t.Fatal("install deleted the running bundle before the relaunch helper could start")
		}
	}
}

func TestInstallRestoresBundleOnFailedMove(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "magpie.app")
	staged := filepath.Join(dir, ".magpie-update", "app", "magpie.app")
	app(t, bundle, "running")
	if err := Install(staged, bundle); err == nil {
		t.Fatal("installed a missing download")
	}
	if version(t, bundle) != "running" {
		t.Fatal("the running app wasn't restored")
	}
}
