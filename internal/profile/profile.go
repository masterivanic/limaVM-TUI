// Package profile implements a generic, declarative mechanism to describe and
// drive an external Lima VM harness from the TUI.
//
// A profile is a YAML file shipped by the harness repository itself. It
// references the harness' own scripts and Lima template by path relative to the
// profile file, so that limavm-tui stays generic: it knows how to load and run
// a profile, but embeds no knowledge of any particular harness.
//
// Because a profile ultimately causes external executables to run, every path
// it references is resolved and constrained to the profile root directory (see
// Profile.resolveScript). Arguments are always passed as argv, never through a
// shell, so a profile cannot inject shell syntax.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SupportedAPIVersion is the only profile schema version understood today.
const SupportedAPIVersion = "v1"

// Command describes a single external executable invocation declared by a
// profile. Script is a path relative to the profile root; Args may contain
// placeholders expanded by ExpandArgs.
type Command struct {
	Script string   `yaml:"script"`
	Args   []string `yaml:"args,omitempty"`
}

// Template describes how the Lima YAML consumed by limactl is obtained.
//
// Source is the input template (informational, and used by the declarative
// path). Render is an optional custom rendering step: harnesses that template
// their Lima YAML themselves declare it here. Output is the rendered file the
// harness produces and that limactl consumes.
type Template struct {
	Source string   `yaml:"source,omitempty"`
	Render *Command `yaml:"render,omitempty"`
	Output string   `yaml:"output,omitempty"`
}

// Defaults carries the VM sizing and layout values a profile wants to use.
// They are exposed to commands through placeholders.
type Defaults struct {
	VMType       string `yaml:"vmType,omitempty"`
	MountType    string `yaml:"mountType,omitempty"`
	Arch         string `yaml:"arch,omitempty"`
	Rosetta      string `yaml:"rosetta,omitempty"`
	CPUs         int    `yaml:"cpus,omitempty"`
	Memory       string `yaml:"memory,omitempty"`
	Disk         string `yaml:"disk,omitempty"`
	ConfigDir    string `yaml:"configDir,omitempty"`
	WorkspaceDir string `yaml:"workspaceDir,omitempty"`
	PortForward  string `yaml:"portForward,omitempty"`
}

// Mount documents a host directory mounted into the guest. It is descriptive:
// the authoritative mount definition lives in the harness' Lima template.
type Mount struct {
	Host  string `yaml:"host"`
	Guest string `yaml:"guest"`
}

// Lifecycle groups the optional hooks a profile can expose.
//
// Create is an escape hatch for harnesses that already own their "bring the VM
// up" logic (rendering, port forwarding, post-creation fixups). When set, it is
// delegated to as-is instead of limavm-tui driving limactl itself, so no
// harness behaviour is lost or duplicated.
type Lifecycle struct {
	Create      *Command `yaml:"create,omitempty"`
	HealthCheck *Command `yaml:"healthCheck,omitempty"`
}

// ProvisioningMode is a named, ordered sequence of steps applied to a VM that
// is already up (for example installing packages then loading images).
type ProvisioningMode struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description,omitempty"`
	Steps       []Command `yaml:"steps"`
}

// Profile is a loaded, validated harness description.
type Profile struct {
	APIVersion        string             `yaml:"apiVersion"`
	Name              string             `yaml:"name"`
	Description       string             `yaml:"description,omitempty"`
	VMName            string             `yaml:"vmName"`
	Template          Template           `yaml:"template,omitempty"`
	Defaults          Defaults           `yaml:"defaults,omitempty"`
	Mounts            []Mount            `yaml:"mounts,omitempty"`
	Lifecycle         Lifecycle          `yaml:"lifecycle,omitempty"`
	ProvisioningModes []ProvisioningMode `yaml:"provisioningModes,omitempty"`

	// path is the absolute path of the profile file itself.
	path string
	// root is the absolute directory containing the profile file. All relative
	// paths in the profile resolve against it, and every resolved path must
	// remain inside it.
	root string
}

// Path returns the absolute path of the profile file.
func (p *Profile) Path() string { return p.path }

// Root returns the absolute directory the profile was loaded from. External
// commands must be executed with this as their working directory: harness
// scripts routinely call sibling scripts through relative paths.
func (p *Profile) Root() string { return p.root }

// Mode returns the provisioning mode with the given name.
func (p *Profile) Mode(name string) (ProvisioningMode, bool) {
	for _, m := range p.ProvisioningModes {
		if m.Name == name {
			return m, true
		}
	}
	return ProvisioningMode{}, false
}

// ModeNames lists the declared provisioning mode names, in file order.
func (p *Profile) ModeNames() []string {
	names := make([]string, 0, len(p.ProvisioningModes))
	for _, m := range p.ProvisioningModes {
		names = append(names, m.Name)
	}
	return names
}

// resolveScript turns a profile-relative script path into an absolute path,
// rejecting anything that escapes the profile root.
//
// This is the security boundary of the profile mechanism: a profile causes
// executables to run, so it must not be able to point at arbitrary locations on
// the host (for example "../../../../usr/bin/something" or "/etc/shadow").
func (p *Profile) resolveScript(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("script path is empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("script path %q must be relative to the profile directory", rel)
	}

	abs := filepath.Join(p.root, rel)
	abs = filepath.Clean(abs)

	// Containment check: abs must be the root itself or live underneath it.
	relToRoot, err := filepath.Rel(p.root, abs)
	if err != nil {
		return "", fmt.Errorf("cannot resolve script path %q: %w", rel, err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("script path %q escapes the profile directory", rel)
	}

	return abs, nil
}

// ResolveScript resolves and checks a script referenced by the profile: it must
// stay inside the profile directory, exist, be a regular file and be
// executable.
func (p *Profile) ResolveScript(rel string) (string, error) {
	abs, err := p.resolveScript(rel)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("script %q not found at %s", rel, abs)
		}
		return "", fmt.Errorf("cannot stat script %q: %w", rel, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("script %q is a directory", rel)
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("script %q is not executable (chmod +x %s)", rel, abs)
	}

	return abs, nil
}

// ResolvePath resolves a non-executable path referenced by the profile (a
// template source, a render output) with the same containment rule. The target
// is not required to exist: rendered outputs are produced at runtime.
func (p *Profile) ResolvePath(rel string) (string, error) {
	return p.resolveScript(rel)
}

// placeholders returns the substitution table exposed to command arguments.
func (p *Profile) placeholders() map[string]string {
	cpus := ""
	if p.Defaults.CPUs > 0 {
		cpus = strconv.Itoa(p.Defaults.CPUs)
	}

	templateOut := p.Template.Output
	if templateOut == "" {
		templateOut = p.Template.Source
	}

	return map[string]string{
		"VM_NAME":       p.VMName,
		"PROFILE_NAME":  p.Name,
		"PROFILE_ROOT":  p.root,
		"VM_TYPE":       p.Defaults.VMType,
		"MOUNT_TYPE":    p.Defaults.MountType,
		"ARCH":          p.Defaults.Arch,
		"ROSETTA":       p.Defaults.Rosetta,
		"CPUS":          cpus,
		"MEMORY":        p.Defaults.Memory,
		"DISK":          p.Defaults.Disk,
		"CONFIG_DIR":    p.Defaults.ConfigDir,
		"WORKSPACE_DIR": p.Defaults.WorkspaceDir,
		"PORT_FORWARD":  p.Defaults.PortForward,
		"TEMPLATE_OUT":  templateOut,
		"TEMPLATE_SRC":  p.Template.Source,
	}
}

// ExpandArgs substitutes ${NAME} placeholders in a command's arguments.
//
// When a command declares no arguments, it defaults to the VM name, which is
// the calling convention of most harness scripts. Unknown placeholders are left
// untouched so that a typo surfaces visibly instead of silently becoming an
// empty argument.
func (p *Profile) ExpandArgs(c Command) []string {
	args := c.Args
	if len(args) == 0 {
		return []string{p.VMName}
	}

	table := p.placeholders()
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, expand(a, table))
	}
	return out
}

// expand replaces every known ${KEY} occurrence in s.
func expand(s string, table map[string]string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	for k, v := range table {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}
