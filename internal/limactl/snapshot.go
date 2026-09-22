package limactl

import (
	"bufio"
	"context"
	"fmt"
	"strings"
)

// SnapshotList lists snapshots for an instance.
func (c *Client) SnapshotList(ctx context.Context, instance string) ([]Snapshot, error) {
	if err := ValidateName(instance); err != nil {
		return nil, err
	}

	out, err := c.Run(ctx, "snapshot", "list", instance)
	if err != nil {
		// If no snapshots exist or command reports empty
		if strings.Contains(strings.ToLower(err.Error()), "no snapshot") {
			return []Snapshot{}, nil
		}
		return nil, err
	}

	var snapshots []Snapshot
	scanner := bufio.NewScanner(strings.NewReader(out))
	isHeader := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "time=") {
			continue
		}
		// Skip table header (e.g. TAG  CREATED  SIZE)
		if isHeader && (strings.HasPrefix(line, "TAG") || strings.HasPrefix(line, "Snapshot")) {
			isHeader = false
			continue
		}
		isHeader = false

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		snap := Snapshot{
			Tag: fields[0],
		}
		if len(fields) >= 2 {
			snap.Size = fields[len(fields)-1]
		}
		snapshots = append(snapshots, snap)
	}

	return snapshots, nil
}

// SnapshotCreate creates a new snapshot with the given tag for an instance.
func (c *Client) SnapshotCreate(ctx context.Context, instance string, tag string) error {
	if err := ValidateName(instance); err != nil {
		return err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("snapshot tag cannot be empty")
	}

	_, err := c.Run(ctx, "snapshot", "create", instance, "--tag", tag)
	return err
}

// SnapshotDelete removes an existing snapshot.
func (c *Client) SnapshotDelete(ctx context.Context, instance string, tag string) error {
	if err := ValidateName(instance); err != nil {
		return err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("snapshot tag cannot be empty")
	}

	_, err := c.Run(ctx, "snapshot", "delete", instance, "--tag", tag)
	return err
}

// SnapshotApply restores an instance to the state of a snapshot.
func (c *Client) SnapshotApply(ctx context.Context, instance string, tag string) error {
	if err := ValidateName(instance); err != nil {
		return err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("snapshot tag cannot be empty")
	}

	_, err := c.Run(ctx, "snapshot", "apply", instance, "--tag", tag)
	return err
}
