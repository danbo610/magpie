// A windowless native app for checking the real update package through
// LaunchServices. Run check.py on a Mac; no installed magpie is touched.
package main

/*
#cgo LDFLAGS: -framework Cocoa
void startCocoa(void);
void quitCocoa(void);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/update"
)

var root string

func save(name string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	tmp := filepath.Join(root, name+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		panic(err)
	}
	if err := os.Rename(tmp, filepath.Join(root, name)); err != nil {
		panic(err)
	}
}

func check(bundle, mode string, cycles int) error {
	n := 0
	if b, err := os.ReadFile(filepath.Join(root, "cycle")); err == nil {
		n, err = strconv.Atoi(string(b))
		if err != nil {
			return err
		}
	}
	update.RemoveOldBundles(bundle)
	olds, err := filepath.Glob(filepath.Join(root, ".Probe.app.previous-*.app"))
	if err != nil || len(olds) != 0 {
		return fmt.Errorf("old copies not cleaned after launch: %v, %v", olds, err)
	}
	if n == cycles {
		return nil
	}
	stage := filepath.Join(root, ".magpie-update")
	staged := filepath.Join(stage, "app", "magpie.app")
	if err := os.MkdirAll(filepath.Dir(staged), 0o755); err != nil {
		return err
	}
	if mode == "rollback" {
		if err := update.Install(staged, bundle); err == nil {
			return fmt.Errorf("installed a missing download")
		}
		return proc.Command("/usr/bin/true").Run()
	}
	if out, err := proc.Command("/usr/bin/ditto", bundle, staged).CombinedOutput(); err != nil {
		return fmt.Errorf("stage: %w: %s", err, out)
	}
	if err := update.Install(staged, bundle); err != nil {
		return err
	}
	// A later download is allowed to clear the staging tree while this
	// image is still running; its retained copy must survive that.
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "cycle"), []byte(strconv.Itoa(n+1)), 0o600); err != nil {
		return err
	}
	if mode == "delete-control" {
		olds, _ := filepath.Glob(filepath.Join(root, ".Probe.app.previous-*.app"))
		for _, old := range olds {
			if err := os.RemoveAll(old); err != nil {
				return err
			}
		}
	}
	if mode == "ordinary-quit" {
		save("waiting.json", os.Getpid())
	} else if err := update.Relaunch(bundle, true); err != nil {
		return err
	}
	C.quitCocoa()
	return fmt.Errorf("Cocoa returned instead of terminating")
}

func main() {
	runtime.LockOSThread()
	bundle := update.Bundle()
	root = filepath.Dir(bundle)
	var cfg struct {
		Mode   string
		Cycles int
	}
	b, err := os.ReadFile(filepath.Join(root, "config.json"))
	if err != nil || json.Unmarshal(b, &cfg) != nil {
		panic("missing fixture config")
	}
	C.startCocoa()
	err = check(bundle, cfg.Mode, cfg.Cycles)
	save("result.json", map[string]any{"mode": cfg.Mode, "success": err == nil, "error": fmt.Sprint(err)})
	C.quitCocoa()
}
