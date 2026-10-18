package gui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/update"
)

func TestBundleRelaunchFailureKeepsAppAndRetries(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "magpie.app")
	staged := filepath.Join(dir, ".magpie-update", "app", "magpie.app")
	for path, content := range map[string]string{bundle: "old", staged: "new"} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "version"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	wasUpdates, wasRelaunch := updates, relaunchBundle
	t.Cleanup(func() { updates, relaunchBundle = wasUpdates, wasRelaunch })
	u := &updater{bundle: bundle, staged: staged, state: "ready", latest: &update.Release{Version: "0.1.11"}}
	updates = u
	calls := 0
	relaunchBundle = func(path string, window bool) error {
		calls++
		if path != bundle || !window {
			t.Fatalf("relaunch(%q, %v)", path, window)
		}
		if calls == 1 {
			return errors.New("the relaunch helper did not become ready")
		}
		return nil
	}
	if restartToUpdate(false, true, "settings") {
		t.Fatal("told the GUI to quit without a ready relaunch helper")
	}
	if j := u.json(); j.State != "ready" || j.Error == "" {
		t.Fatalf("the failed handoff isn't shown with a retry: %+v", j)
	}
	if _, err := os.Stat(filepath.Join(settings.Dir(), updatedInAppFile)); !os.IsNotExist(err) {
		t.Fatal("a failed handoff was marked as an in-app update")
	}
	if !restartToUpdate(false, true, "settings") || calls != 2 {
		t.Fatalf("retry didn't hand off to the installed app; calls = %d", calls)
	}
	if b, err := os.ReadFile(filepath.Join(bundle, "version")); err != nil || string(b) != "new" {
		t.Fatalf("retry lost the installed app: %q, %v", b, err)
	}
	if j := u.json(); j.Error != "" {
		t.Fatalf("retry left the error behind: %+v", j)
	}
	os.Remove(filepath.Join(settings.Dir(), updatedInAppFile))
}
