package limactl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	// DefaultCommandTimeout for short read-only or inspection commands.
	DefaultCommandTimeout = 20 * time.Second

	// LifecycleCommandTimeout for VM creation, startup, stop, etc.
	LifecycleCommandTimeout = 10 * time.Minute

	// vmNameRegex defines valid Lima instance names.
	vmNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$`)
)

// ValidateName ensures an instance or disk name conforms to Lima requirements.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if !vmNameRegex.MatchString(name) {
		return fmt.Errorf("invalid name %q: must start with alphanumeric character and contain only [a-zA-Z0-9_.-] (max 63 chars)", name)
	}
	return nil
}

// ExecError captures command execution failures with structured context.
type ExecError struct {
	Args     []string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *ExecError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("limactl %s failed (code %d): %s", strings.Join(e.Args, " "), e.ExitCode, strings.TrimSpace(e.Stderr))
	}
	return fmt.Sprintf("limactl %s failed: %v", strings.Join(e.Args, " "), e.Err)
}

func (e *ExecError) Unwrap() error {
	return e.Err
}

// Client wraps limactl command executions.
type Client struct {
	BinaryPath string
}

// NewClient returns a limactl client, resolving the binary from PATH or LIMA_BINARY.
func NewClient() (*Client, error) {
	bin := os.Getenv("LIMA_BINARY")
	if bin == "" {
		bin = "limactl"
	}

	path, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("limactl binary not found in PATH: %w", err)
	}

	return &Client{BinaryPath: path}, nil
}

// CommandContext creates an exec.Cmd with limactl binary and context.
func (c *Client) CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, c.BinaryPath, args...)
}

// Run executes a command and returns trimmed stdout as string.
func (c *Client) Run(ctx context.Context, args ...string) (string, error) {
	cmd := c.CommandContext(ctx, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return "", &ExecError{
			Args:     args,
			ExitCode: exitCode,
			Stderr:   stderr.String(),
			Err:      err,
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

// DecodeNDJSON parses newline-delimited JSON stream into a slice of T.
// Ignores non-JSON log lines (such as logrus warnings).
func DecodeNDJSON[T any](r io.Reader) ([]T, error) {
	var items []T
	scanner := bufio.NewScanner(r)
	// Buffer large JSON lines (e.g. instances with deep configs up to 10MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Skip logrus log lines: time="2026-..." level=...
		if strings.HasPrefix(line, "time=") || !strings.HasPrefix(line, "{") {
			continue
		}

		var item T
		dec := json.NewDecoder(strings.NewReader(line))
		if err := dec.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode JSON line: %w", err)
		}
		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return items, nil
}
