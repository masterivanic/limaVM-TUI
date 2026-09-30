package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeHarness builds a minimal but realistic harness directory: a couple of
// executable scripts and a Lima template, mirroring how a real repository ships
// a profile next to its scripts.
func writeHarness(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lima"), 0o755); err != nil {
		t.Fatalf("mkdir lima: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}

	for _, name := range []string{"vm-up.sh", "vm-check.sh", "step-one.sh", "step-two.sh"} {
		p := filepath.Join(root, "scripts", name)
		if err := os.WriteFile(p, []byte("#!/usr/bin/env bash\ntrue\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// A non-executable file, to assert the executable check.
	if err := os.WriteFile(filepath.Join(root, "scripts", "not-exec.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write not-exec.sh: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "lima", "vm.yaml.in"), []byte("arch: x86_64\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	return root
}

func writeProfile(t *testing.T, root, content string) string {
	t.Helper()
	p := filepath.Join(root, DefaultProfileFileName)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	return p
}

const validProfile = `
apiVersion: v1
name: test-harness
description: Test harness
vmName: test-vm
template:
  source: lima/vm.yaml.in
  output: vm_config/vm.yaml
defaults:
  vmType: qemu
  arch: x86_64
  cpus: 4
  memory: 6G
  disk: 40G
  configDir: config
  workspaceDir: workspace
mounts:
  - host: config
    guest: /opt/provisioning
lifecycle:
  create:
    script: scripts/vm-up.sh
    args: ["${VM_NAME}", "${TEMPLATE_OUT}", "${VM_TYPE}", "${CPUS}"]
  healthCheck:
    script: scripts/vm-check.sh
provisioningModes:
  - name: mode-a
    description: First mode
    steps:
      - script: scripts/step-one.sh
      - script: scripts/step-two.sh
`

func TestLoadValidProfile(t *testing.T) {
	root := writeHarness(t)
	path := writeProfile(t, root, validProfile)

	p, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "test-harness" {
		t.Errorf("name = %q, want test-harness", p.Name)
	}
	if p.VMName != "test-vm" {
		t.Errorf("vmName = %q, want test-vm", p.VMName)
	}
	if p.Root() != root {
		t.Errorf("root = %q, want %q", p.Root(), root)
	}
	if p.Path() != path {
		t.Errorf("path = %q, want %q", p.Path(), path)
	}
	if got := p.ModeNames(); len(got) != 1 || got[0] != "mode-a" {
		t.Errorf("ModeNames() = %v, want [mode-a]", got)
	}
	if _, ok := p.Mode("mode-a"); !ok {
		t.Errorf("Mode(mode-a) not found")
	}
	if _, ok := p.Mode("nope"); ok {
		t.Errorf("Mode(nope) unexpectedly found")
	}
	if p.Lifecycle.Create == nil {
		t.Fatalf("lifecycle.create was not parsed")
	}
}

func TestLoadRejectsInvalidProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr string
	}{
		{
			name:    "missing apiVersion",
			profile: "name: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\n",
			wantErr: "apiVersion is required",
		},
		{
			name:    "unsupported apiVersion",
			profile: "apiVersion: v99\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\n",
			wantErr: "unsupported apiVersion",
		},
		{
			name:    "missing name",
			profile: "apiVersion: v1\nvmName: y\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\n",
			wantErr: "name is required",
		},
		{
			name:    "missing vmName",
			profile: "apiVersion: v1\nname: x\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\n",
			wantErr: "vmName is required",
		},
		{
			name:    "no create and no template source",
			profile: "apiVersion: v1\nname: x\nvmName: y\n",
			wantErr: "either lifecycle.create or template.source",
		},
		{
			name:    "unknown field",
			profile: "apiVersion: v1\nname: x\nvmName: y\nbogusField: 1\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\n",
			wantErr: "field bogusField not found",
		},
		{
			name:    "missing script",
			profile: "apiVersion: v1\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/does-not-exist.sh\n",
			wantErr: "not found",
		},
		{
			name:    "non executable script",
			profile: "apiVersion: v1\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/not-exec.sh\n",
			wantErr: "not executable",
		},
		{
			name:    "duplicate mode names",
			profile: "apiVersion: v1\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\nprovisioningModes:\n  - name: dup\n    steps:\n      - script: scripts/step-one.sh\n  - name: dup\n    steps:\n      - script: scripts/step-two.sh\n",
			wantErr: "duplicate mode name",
		},
		{
			name:    "mode without steps",
			profile: "apiVersion: v1\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\nprovisioningModes:\n  - name: empty\n    steps: []\n",
			wantErr: "at least one step is required",
		},
		{
			name:    "mode without name",
			profile: "apiVersion: v1\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: scripts/vm-up.sh\nprovisioningModes:\n  - steps:\n      - script: scripts/step-one.sh\n",
			wantErr: "name is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeHarness(t)
			path := writeProfile(t, root, tc.profile)

			_, err := Load(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// TestPathContainment is the security-critical case: a profile must never be
// able to point at an executable outside its own directory.
func TestPathContainment(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{name: "parent traversal", script: "../evil.sh"},
		{name: "deep traversal", script: "../../../../bin/sh"},
		{name: "absolute path", script: "/bin/sh"},
		{name: "sneaky traversal", script: "scripts/../../evil.sh"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeHarness(t)

			// Place a real executable just outside the root to prove that
			// rejection is due to containment and not to a missing file.
			outside := filepath.Join(filepath.Dir(root), "evil.sh")
			_ = os.WriteFile(outside, []byte("#!/bin/sh\n"), 0o755)
			t.Cleanup(func() { _ = os.Remove(outside) })

			profile := "apiVersion: v1\nname: x\nvmName: y\nlifecycle:\n  create:\n    script: " + tc.script + "\n"
			path := writeProfile(t, root, profile)

			if _, err := Load(path); err == nil {
				t.Fatalf("expected containment rejection for %q, got nil", tc.script)
			}
		})
	}
}

func TestExpandArgs(t *testing.T) {
	root := writeHarness(t)
	path := writeProfile(t, root, validProfile)

	p, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := p.ExpandArgs(*p.Lifecycle.Create)
	want := []string{"test-vm", "vm_config/vm.yaml", "qemu", "4"}

	if len(got) != len(want) {
		t.Fatalf("ExpandArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestExpandArgsDefaultsToVMName documents the calling convention shared by
// most harness scripts: a single positional VM name.
func TestExpandArgsDefaultsToVMName(t *testing.T) {
	root := writeHarness(t)
	path := writeProfile(t, root, validProfile)

	p, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := p.ExpandArgs(*p.Lifecycle.HealthCheck)
	if len(got) != 1 || got[0] != "test-vm" {
		t.Errorf("ExpandArgs() = %v, want [test-vm]", got)
	}
}

func TestExpandArgsLeavesUnknownPlaceholders(t *testing.T) {
	root := writeHarness(t)
	path := writeProfile(t, root, validProfile)

	p, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := p.ExpandArgs(Command{Script: "scripts/step-one.sh", Args: []string{"${NOPE}"}})
	if len(got) != 1 || got[0] != "${NOPE}" {
		t.Errorf("ExpandArgs() = %v, want [${NOPE}] (unknown placeholders are preserved)", got)
	}
}

func TestResolveScript(t *testing.T) {
	root := writeHarness(t)
	path := writeProfile(t, root, validProfile)

	p, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	abs, err := p.ResolveScript("scripts/vm-up.sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if abs != filepath.Join(root, "scripts", "vm-up.sh") {
		t.Errorf("ResolveScript() = %q", abs)
	}

	if _, err := p.ResolveScript("scripts"); err == nil {
		t.Errorf("expected error when script points at a directory")
	}
	if _, err := p.ResolveScript(""); err == nil {
		t.Errorf("expected error for empty script path")
	}
}

// TestResolvePathAllowsMissingTarget covers rendered outputs, which do not
// exist until the harness produces them.
func TestResolvePathAllowsMissingTarget(t *testing.T) {
	root := writeHarness(t)
	path := writeProfile(t, root, validProfile)

	p, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := p.ResolvePath("vm_config/vm.yaml"); err != nil {
		t.Errorf("unexpected error for not-yet-rendered output: %v", err)
	}
	if _, err := p.ResolvePath("../outside.yaml"); err == nil {
		t.Errorf("expected containment rejection for escaping output path")
	}
}

func TestLoadMissingAndDirectory(t *testing.T) {
	root := writeHarness(t)

	if _, err := Load(filepath.Join(root, "nope.yaml")); err == nil {
		t.Errorf("expected error for missing profile")
	}
	if _, err := Load(root); err == nil {
		t.Errorf("expected error when profile path is a directory")
	}
}

func TestDiscover(t *testing.T) {
	root := writeHarness(t)
	writeProfile(t, root, validProfile)

	// A directory source resolves to the conventional file name, and passing
	// both the directory and the file must not yield duplicates.
	results := Discover([]string{root, filepath.Join(root, DefaultProfileFileName)}, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 de-duplicated result, got %d", len(results))
	}

	valid := Valid(results)
	if len(valid) != 1 || valid[0].Name != "test-harness" {
		t.Fatalf("Valid() = %v", valid)
	}
	if len(Failed(results)) != 0 {
		t.Errorf("expected no failures, got %v", Failed(results))
	}
}

func TestDiscoverReportsFailures(t *testing.T) {
	emptyDir := t.TempDir()

	results := Discover([]string{
		filepath.Join(emptyDir, "does-not-exist.yaml"),
		emptyDir, // directory without the conventional profile file
	}, nil)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(Failed(results)) != 2 {
		t.Errorf("expected both sources to fail, got %v", Failed(results))
	}
	if len(Valid(results)) != 0 {
		t.Errorf("expected no valid profiles")
	}
}

func TestLoadConfigMissingFileIsNotAnError(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.ProfilePaths) != 0 {
		t.Errorf("expected empty config, got %v", cfg.ProfilePaths)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "profilePaths:\n  - /one\n  - /two\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.ProfilePaths) != 2 || cfg.ProfilePaths[0] != "/one" {
		t.Errorf("ProfilePaths = %v", cfg.ProfilePaths)
	}
}
