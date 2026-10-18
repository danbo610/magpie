"""Manual macOS regression: python3 internal/update/testdata/relaunch/check.py.

Builds ad-hoc signed Cocoa fixtures using the actual update package and opens
them through LaunchServices. Keeps logs in the printed temporary directory.
The deletion control reproduces #2362 on macOS 13; an OS where it succeeds
is reported separately, not counted as a failure of the retained-image fix.
Does not start magpie's gateway, touch its configuration, or install an app.
"""
import json
import os
from pathlib import Path
import plistlib
import shutil
import subprocess
import tempfile
import time

repo = Path(__file__).resolve().parents[4]
root = Path(tempfile.mkdtemp(prefix="magpie-native-relaunch-"))
print(f"Evidence: {root}", flush=True)
subprocess.run(["go", "build", "-o", str(root / "probe"),
                "./internal/update/testdata/relaunch"], cwd=repo, check=True)
results = []
for mode, cycles in [("delete-control", 1), ("consecutive", 3),
                     ("ordinary-quit", 1), ("rollback", 1)]:
    case = root / mode
    app = case / "Probe.app"
    mac = app / "Contents/MacOS"
    mac.mkdir(parents=True)
    shutil.copyfile(root / "probe", mac / "magpie")
    (mac / "magpie").chmod(0o755)
    (app / "Contents/Info.plist").write_bytes(plistlib.dumps({
        "CFBundleExecutable": "magpie", "CFBundleName": "Update Probe",
        "CFBundleIdentifier": f"local.magpie.update-probe.{root.name}.{mode}",
        "CFBundleVersion": "1", "CFBundlePackageType": "APPL", "LSUIElement": True,
    }))
    (case / "config.json").write_text(json.dumps({"Mode": mode, "Cycles": cycles}))
    subprocess.run(["/usr/bin/codesign", "--force", "--sign", "-", "--options",
                    "runtime", str(app)], check=True, capture_output=True)
    subprocess.run(["/usr/bin/open", "-g", str(app)], check=True)
    deadline = time.monotonic() + 60
    manual = False
    while time.monotonic() < deadline:
        result = case / "result.json"
        if result.exists():
            row = json.loads(result.read_text())
            results.append(row)
            print(json.dumps(row), flush=True)
            break
        waiting = case / "waiting.json"
        if mode == "ordinary-quit" and waiting.exists() and not manual:
            try:
                os.kill(json.loads(waiting.read_text()), 0)
            except ProcessLookupError:
                subprocess.run(["/usr/bin/open", "-g", str(app)], check=True)
                manual = True
        time.sleep(0.1)
    else:
        raise SystemExit(f"Timed out: inspect {case}")
    time.sleep(0.5)  # the short Cocoa termination timer
(root / "results.json").write_text(json.dumps(results, indent=2) + "\n")
if any(not row["success"] for row in results if row["mode"] != "delete-control"):
    raise SystemExit(1)
