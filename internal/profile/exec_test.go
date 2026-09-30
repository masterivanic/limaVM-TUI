package profile_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epel/limavm-tui/internal/profile"
)

// TestCommandExecutionContract locks the two guarantees the UI relies on when
// running a profile command:
//
//  1. arguments reach the script as separate argv entries, with placeholders
//     already expanded (and never interpreted by a shell);
//  2. the script runs with the profile directory as working directory, which
//     harness scripts depend on to reach their sibling scripts and assets.
func TestCommandExecutionContract(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "lima.yaml.in"), []byte("arch: x86_64\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	// The script reports its working directory and each argument on its own
	// line, so splitting is unambiguous.
	script := "#!/usr/bin/env bash\n" +
		"echo \"CWD=$PWD\"\n" +
		"for a in \"$@\"; do echo \"ARG=$a\"; done\n"
	scriptPath := filepath.Join(root, "scripts", "report.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	// A sibling script proves relative invocation works from the profile root.
	sibling := "#!/usr/bin/env bash\n./scripts/report.sh sibling-ok\n"
	if err := os.WriteFile(filepath.Join(root, "scripts", "outer.sh"), []byte(sibling), 0o755); err != nil {
		t.Fatalf("write sibling: %v", err)
	}

	profileYAML := `
apiVersion: v1
name: exec-harness
vmName: exec-vm
template:
  source: lima.yaml.in
  output: vm_config/out.yaml
defaults:
  vmType: qemu
  cpus: 4
  memory: 6G
lifecycle:
  create:
    script: scripts/report.sh
    args: ["${VM_NAME}", "${TEMPLATE_OUT}", "${VM_TYPE}", "${CPUS}", "${MEMORY}", "a b c"]
  healthCheck:
    script: scripts/outer.sh
`
	profilePath := filepath.Join(root, profile.DefaultProfileFileName)
	if err := os.WriteFile(profilePath, []byte(profileYAML), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	p, err := profile.Load(profilePath)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	runCommand := func(c profile.Command) string {
		t.Helper()
		abs, err := p.ResolveScript(c.Script)
		if err != nil {
			t.Fatalf("resolve %q: %v", c.Script, err)
		}
		cmd := exec.Command(abs, p.ExpandArgs(c)...)
		cmd.Dir = p.Root()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run %q: %v (output: %s)", c.Script, err, out)
		}
		return string(out)
	}

	out := runCommand(*p.Lifecycle.Create)

	var gotArgs []string
	var gotCWD string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		switch {
		case strings.HasPrefix(line, "CWD="):
			gotCWD = strings.TrimPrefix(line, "CWD=")
		case strings.HasPrefix(line, "ARG="):
			gotArgs = append(gotArgs, strings.TrimPrefix(line, "ARG="))
		}
	}

	// On macOS the temp dir is a symlink (/var -> /private/var), so compare
	// resolved paths.
	wantCWD, _ := filepath.EvalSymlinks(p.Root())
	haveCWD, _ := filepath.EvalSymlinks(gotCWD)
	if wantCWD != haveCWD {
		t.Errorf("working directory = %q, want %q", gotCWD, p.Root())
	}

	want := []string{"exec-vm", "vm_config/out.yaml", "qemu", "4", "6G", "a b c"}
	if len(gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", gotArgs, want)
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], want[i])
		}
	}

	// "a b c" must arrive as a single argument, proving no shell word splitting.
	if gotArgs[len(gotArgs)-1] != "a b c" {
		t.Errorf("argument with spaces was split: %q", gotArgs[len(gotArgs)-1])
	}

	// The sibling script resolves ./scripts/... relative to the profile root.
	if out := runCommand(*p.Lifecycle.HealthCheck); !strings.Contains(out, "ARG=sibling-ok") {
		t.Errorf("sibling script invocation failed, output: %s", out)
	}
}
