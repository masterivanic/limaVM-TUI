package limactl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CreateOpts holds configuration parameters when provisioning a new instance.
type CreateOpts struct {
	Name            string
	TemplateOrPath  string
	CPUs            int
	MemoryGiB       float64
	DiskGiB         float64
	Arch            string
	VMType          string
	ContainerdMode  string
	AdditionalFlags []string
}

// List retrieves all Lima VM instances.
func (c *Client) List(ctx context.Context) ([]Instance, error) {
	cmd := c.CommandContext(ctx, "list", "--json", "--all-fields")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &ExecError{
			Args:   []string{"list", "--json", "--all-fields"},
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	return DecodeNDJSON[Instance](&stdout)
}

// Get retrieves details for a single Lima instance by name.
func (c *Client) Get(ctx context.Context, name string) (*Instance, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	cmd := c.CommandContext(ctx, "list", "--json", "--all-fields", name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &ExecError{
			Args:   []string{"list", "--json", name},
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	instances, err := DecodeNDJSON[Instance](&stdout)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("instance %q not found", name)
	}

	return &instances[0], nil
}

// Create provisions a new Lima instance.
func (c *Client) Create(ctx context.Context, opts CreateOpts) error {
	if opts.Name != "" {
		if err := ValidateName(opts.Name); err != nil {
			return err
		}
	}
	if strings.TrimSpace(opts.TemplateOrPath) == "" {
		return fmt.Errorf("template or YAML path is required")
	}

	args := []string{"create", "-y"}
	if opts.Name != "" {
		args = append(args, fmt.Sprintf("--name=%s", opts.Name))
	}
	if opts.CPUs > 0 {
		args = append(args, fmt.Sprintf("--cpus=%d", opts.CPUs))
	}
	if opts.MemoryGiB > 0 {
		args = append(args, fmt.Sprintf("--memory=%s", strconv.FormatFloat(opts.MemoryGiB, 'f', -1, 64)))
	}
	if opts.DiskGiB > 0 {
		args = append(args, fmt.Sprintf("--disk=%s", strconv.FormatFloat(opts.DiskGiB, 'f', -1, 64)))
	}
	if opts.Arch != "" {
		args = append(args, fmt.Sprintf("--arch=%s", opts.Arch))
	}
	if opts.VMType != "" {
		args = append(args, fmt.Sprintf("--vm-type=%s", opts.VMType))
	}
	if opts.ContainerdMode != "" {
		args = append(args, fmt.Sprintf("--containerd=%s", opts.ContainerdMode))
	}
	args = append(args, opts.AdditionalFlags...)
	args = append(args, opts.TemplateOrPath)

	_, err := c.Run(ctx, args...)
	return err
}

// Start launches a stopped Lima VM instance.
func (c *Client) Start(ctx context.Context, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	_, err := c.Run(ctx, "start", "-y", name)
	return err
}

// Stop halts a running Lima VM instance.
func (c *Client) Stop(ctx context.Context, name string, force bool) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	args := []string{"stop", "-y"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	_, err := c.Run(ctx, args...)
	return err
}

// Restart reboots a running Lima VM instance.
func (c *Client) Restart(ctx context.Context, name string, force bool) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	args := []string{"restart", "-y"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	_, err := c.Run(ctx, args...)
	return err
}

// Delete removes an instance.
func (c *Client) Delete(ctx context.Context, name string, force bool) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	args := []string{"delete", "-y"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	_, err := c.Run(ctx, args...)
	return err
}

// Clone creates a clone of an existing instance with a new name.
func (c *Client) Clone(ctx context.Context, oldName, newName string) error {
	if err := ValidateName(oldName); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if err := ValidateName(newName); err != nil {
		return fmt.Errorf("destination: %w", err)
	}

	_, err := c.Run(ctx, "clone", oldName, newName)
	return err
}

// Rename emulates renaming an instance via atomic Clone then Delete.
func (c *Client) Rename(ctx context.Context, oldName, newName string) error {
	if err := c.Clone(ctx, oldName, newName); err != nil {
		return fmt.Errorf("clone stage failed: %w", err)
	}
	if err := c.Delete(ctx, oldName, true); err != nil {
		return fmt.Errorf("cleanup original instance %q failed (clone %q succeeded): %w", oldName, newName, err)
	}
	return nil
}

// Protect flags an instance against accidental removal.
func (c *Client) Protect(ctx context.Context, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	_, err := c.Run(ctx, "protect", name)
	return err
}

// Unprotect removes deletion protection from an instance.
func (c *Client) Unprotect(ctx context.Context, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	_, err := c.Run(ctx, "unprotect", name)
	return err
}

// ShowSSH retrieves SSH connection command / config format for an instance.
func (c *Client) ShowSSH(ctx context.Context, name string, format string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	if format == "" {
		format = "config"
	}
	return c.Run(ctx, "show-ssh", fmt.Sprintf("--format=%s", format), name)
}

// GetYAML reads the lima.yaml content for an instance directly from its directory.
func (c *Client) GetYAML(ctx context.Context, name string) (string, error) {
	inst, err := c.Get(ctx, name)
	if err != nil {
		return "", err
	}

	yamlPath := filepath.Join(inst.Dir, "lima.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", yamlPath, err)
	}

	return string(data), nil
}

// SaveYAML writes updated YAML content to an instance's lima.yaml after validation.
func (c *Client) SaveYAML(ctx context.Context, name string, yamlContent string) error {
	inst, err := c.Get(ctx, name)
	if err != nil {
		return err
	}

	// Validate content before writing
	tmp, err := os.CreateTemp("", "lima-validate-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp validation file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(yamlContent); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp validation file: %w", err)
	}
	_ = tmp.Close()

	if err := c.ValidateYAML(ctx, tmpName); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	yamlPath := filepath.Join(inst.Dir, "lima.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o640); err != nil {
		return fmt.Errorf("write %q: %w", yamlPath, err)
	}

	return nil
}
