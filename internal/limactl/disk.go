package limactl

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// DiskList lists all Lima virtual disks.
func (c *Client) DiskList(ctx context.Context) ([]Disk, error) {
	cmd := c.CommandContext(ctx, "disk", "list", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &ExecError{
			Args:   []string{"disk", "list", "--json"},
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	return DecodeNDJSON[Disk](&stdout)
}

// DiskCreate creates a new virtual disk.
func (c *Client) DiskCreate(ctx context.Context, name string, size string, format string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("disk name: %w", err)
	}
	size = strings.TrimSpace(size)
	if size == "" {
		return fmt.Errorf("disk size is required (e.g. 10GiB)")
	}

	args := []string{"disk", "create", name, "--size", size}
	if format != "" {
		args = append(args, "--format", format)
	}

	_, err := c.Run(ctx, args...)
	return err
}

// DiskDelete deletes a virtual disk.
func (c *Client) DiskDelete(ctx context.Context, name string, force bool) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("disk name: %w", err)
	}
	args := []string{"disk", "delete", "-y"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	_, err := c.Run(ctx, args...)
	return err
}

// DiskResize resizes an existing virtual disk.
func (c *Client) DiskResize(ctx context.Context, name string, size string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("disk name: %w", err)
	}
	size = strings.TrimSpace(size)
	if size == "" {
		return fmt.Errorf("target size is required (e.g. 20GiB)")
	}

	_, err := c.Run(ctx, "disk", "resize", name, "--size", size)
	return err
}

// DiskUnlock unlocks an orphaned locked disk.
func (c *Client) DiskUnlock(ctx context.Context, name string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("disk name: %w", err)
	}
	_, err := c.Run(ctx, "disk", "unlock", name)
	return err
}
