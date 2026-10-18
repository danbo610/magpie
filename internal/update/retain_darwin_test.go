package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRetainedBundleCleanup(t *testing.T) {
	if ready := os.Getenv("MAGPIE_TEST_OLD_BUNDLE_READY"); ready != "" {
		if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Minute) // the parent kills and waits for this CLI
		return
	}
	dir := t.TempDir()
	bundle := filepath.Join(dir, "magpie.app")
	old := retainedBundle(bundle)
	app(t, old, "running")
	exe := filepath.Join(old, "Contents", "MacOS", "magpie")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, b, 0o755); err != nil {
		t.Fatal(err)
	}
	RemoveOldBundles(bundle)
	if version(t, old) != "running" {
		t.Fatal("cleanup removed a live updater's image")
	}

	// Give the retained copy an updater which has exited, but keep a
	// second process (the CLI) running from that very image.
	owner := exec.Command("/usr/bin/true")
	if err := owner.Run(); err != nil {
		t.Fatal(err)
	}
	dead := strings.Replace(old, "previous-"+strconv.Itoa(os.Getpid())+"-", "previous-"+strconv.Itoa(owner.Process.Pid)+"-", 1)
	if err := os.Rename(old, dead); err != nil {
		t.Fatal(err)
	}
	exe = filepath.Join(dead, "Contents", "MacOS", "magpie")
	ready := filepath.Join(dir, "cli-ready")
	cli := exec.Command(exe, "-test.run=^TestRetainedBundleCleanup$")
	cli.Env = append(os.Environ(), "MAGPIE_TEST_OLD_BUNDLE_READY="+ready)
	if err := cli.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cli.ProcessState == nil {
			cli.Process.Kill()
			cli.Wait()
		}
	})
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(end) {
			t.Fatal("the CLI never became ready")
		}
	}
	RemoveOldBundles(bundle)
	if version(t, dead) != "running" {
		t.Fatal("cleanup removed the CLI's live image")
	}
	cli.Process.Kill()
	cli.Wait()

	// A permissions failure is deferred, and retried after it is fixed.
	if os.Getuid() != 0 {
		os.Chmod(filepath.Dir(exe), 0o555)
		t.Cleanup(func() { os.Chmod(filepath.Dir(exe), 0o755) })
		RemoveOldBundles(bundle)
		if _, err := os.Stat(exe); err != nil {
			t.Fatalf("cleanup lost the protected image: %v", err)
		}
		os.Chmod(filepath.Dir(exe), 0o755)
	}
	RemoveOldBundles(bundle)
	if _, err := os.Stat(dead); !os.IsNotExist(err) {
		t.Fatalf("the unused copy was not cleaned: %v", err)
	}
}

func TestBundleCleanupUnknownImage(t *testing.T) {
	// A missing/unreadable image is unknown, not proof no process uses it.
	if bundleUnused(filepath.Join(t.TempDir(), "missing.app")) {
		t.Fatal("a failed stat was taken as an unused image")
	}
}

func TestRelaunchRequiresShellAcknowledgment(t *testing.T) {
	for _, shell := range []string{"/usr/bin/true", filepath.Join(t.TempDir(), "missing-shell")} {
		if err := afterExitShell(os.Getpid(), "exit 0", shell); err == nil {
			t.Fatalf("accepted %s without a running relaunch shell", shell)
		}
	}
}
