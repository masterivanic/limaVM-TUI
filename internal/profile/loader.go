package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// maxProfileSize caps how much of a profile file is read, guarding against
// accidentally pointing --profile at a huge unrelated file.
const maxProfileSize = 1 << 20 // 1 MiB

// Load reads, parses and validates a profile file.
//
// The returned Profile has its root set to the directory containing the file;
// all relative paths it declares resolve against that directory.
func Load(path string) (*Profile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve profile path %q: %w", path, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q not found", path)
		}
		return nil, fmt.Errorf("cannot stat profile %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("profile %q is a directory, expected a YAML file", path)
	}
	if info.Size() > maxProfileSize {
		return nil, fmt.Errorf("profile %q is too large (%d bytes, max %d)", path, info.Size(), maxProfileSize)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read profile %q: %w", path, err)
	}

	var p Profile
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", path, err)
	}

	p.path = abs
	p.root = filepath.Dir(abs)

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("invalid profile %q: %w", path, err)
	}

	return &p, nil
}

// Validate checks the profile is internally consistent and that every path it
// references is safe and usable.
func (p *Profile) Validate() error {
	if p.APIVersion == "" {
		return fmt.Errorf("apiVersion is required (expected %q)", SupportedAPIVersion)
	}
	if p.APIVersion != SupportedAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q (expected %q)", p.APIVersion, SupportedAPIVersion)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(p.VMName) == "" {
		return fmt.Errorf("vmName is required")
	}

	// A profile must offer at least one way to bring the VM up, otherwise it
	// cannot do anything useful.
	if p.Lifecycle.Create == nil && p.Template.Source == "" {
		return fmt.Errorf("profile must declare either lifecycle.create or template.source")
	}

	if p.Template.Source != "" {
		if _, err := p.ResolvePath(p.Template.Source); err != nil {
			return fmt.Errorf("template.source: %w", err)
		}
	}
	if p.Template.Output != "" {
		if _, err := p.ResolvePath(p.Template.Output); err != nil {
			return fmt.Errorf("template.output: %w", err)
		}
	}
	if p.Template.Render != nil {
		if err := p.validateCommand(*p.Template.Render); err != nil {
			return fmt.Errorf("template.render: %w", err)
		}
	}
	if p.Lifecycle.Create != nil {
		if err := p.validateCommand(*p.Lifecycle.Create); err != nil {
			return fmt.Errorf("lifecycle.create: %w", err)
		}
	}
	if p.Lifecycle.HealthCheck != nil {
		if err := p.validateCommand(*p.Lifecycle.HealthCheck); err != nil {
			return fmt.Errorf("lifecycle.healthCheck: %w", err)
		}
	}

	seen := make(map[string]struct{}, len(p.ProvisioningModes))
	for i, m := range p.ProvisioningModes {
		if strings.TrimSpace(m.Name) == "" {
			return fmt.Errorf("provisioningModes[%d]: name is required", i)
		}
		if _, dup := seen[m.Name]; dup {
			return fmt.Errorf("provisioningModes: duplicate mode name %q", m.Name)
		}
		seen[m.Name] = struct{}{}

		if len(m.Steps) == 0 {
			return fmt.Errorf("provisioningModes[%q]: at least one step is required", m.Name)
		}
		for j, s := range m.Steps {
			if err := p.validateCommand(s); err != nil {
				return fmt.Errorf("provisioningModes[%q].steps[%d]: %w", m.Name, j, err)
			}
		}
	}

	for i, mnt := range p.Mounts {
		if strings.TrimSpace(mnt.Host) == "" {
			return fmt.Errorf("mounts[%d]: host is required", i)
		}
		if strings.TrimSpace(mnt.Guest) == "" {
			return fmt.Errorf("mounts[%d]: guest is required", i)
		}
		if _, err := p.ResolvePath(mnt.Host); err != nil {
			return fmt.Errorf("mounts[%d].host: %w", i, err)
		}
	}

	return nil
}

// validateCommand checks a declared command points at a usable script.
func (p *Profile) validateCommand(c Command) error {
	if strings.TrimSpace(c.Script) == "" {
		return fmt.Errorf("script is required")
	}
	if _, err := p.ResolveScript(c.Script); err != nil {
		return err
	}
	return nil
}
