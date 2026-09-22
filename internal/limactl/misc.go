package limactl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Info retrieves diagnostic information about the host and Lima installation.
func (c *Client) Info(ctx context.Context) (*InfoDiagnostic, error) {
	cmd := c.CommandContext(ctx, "info")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &ExecError{
			Args:   []string{"info"},
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	// Filter out any potential leading non-json lines
	out := stdout.String()
	firstBrace := strings.Index(out, "{")
	if firstBrace == -1 {
		return nil, fmt.Errorf("no valid JSON found in info output: %s", out)
	}

	var diag InfoDiagnostic
	if err := json.Unmarshal([]byte(out[firstBrace:]), &diag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal info JSON: %w", err)
	}

	return &diag, nil
}

// Prune cleans up unused Lima resources and cache objects.
func (c *Client) Prune(ctx context.Context, keepReferred bool) (string, error) {
	args := []string{"prune", "-y"}
	if keepReferred {
		args = append(args, "--keep-referred")
	}

	return c.Run(ctx, args...)
}

// ValidateYAML executes `limactl validate` on a YAML file path.
func (c *Client) ValidateYAML(ctx context.Context, filePath string) error {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	_, err := c.Run(ctx, "validate", filePath)
	return err
}
