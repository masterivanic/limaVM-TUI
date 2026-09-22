package limactl

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// NetworkList lists all configured Lima networks.
func (c *Client) NetworkList(ctx context.Context) ([]Network, error) {
	cmd := c.CommandContext(ctx, "network", "list", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &ExecError{
			Args:   []string{"network", "list", "--json"},
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	return DecodeNDJSON[Network](&stdout)
}

// NetworkCreate configures a new Lima virtual network.
func (c *Client) NetworkCreate(ctx context.Context, name string, gateway string, mode string, iface string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("network name: %w", err)
	}
	gateway = strings.TrimSpace(gateway)
	if gateway == "" {
		return fmt.Errorf("gateway CIDR is required (e.g. 192.168.42.1/24)")
	}

	args := []string{"network", "create", name, "--gateway", gateway}
	if mode != "" {
		args = append(args, "--mode", mode)
	}
	if iface != "" {
		args = append(args, "--interface", iface)
	}

	_, err := c.Run(ctx, args...)
	return err
}

// NetworkDelete removes a Lima virtual network.
func (c *Client) NetworkDelete(ctx context.Context, name string, force bool) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("network name: %w", err)
	}

	args := []string{"network", "delete", "-y"}
	// In limactl 1.2.3, --force is currently always required for network delete
	if force || true {
		args = append(args, "-f")
	}
	args = append(args, name)

	_, err := c.Run(ctx, args...)
	return err
}
