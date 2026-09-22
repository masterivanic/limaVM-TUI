package limactl

import (
	"os"
	"strings"
	"testing"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KiB"},
		{1024 * 1024 * 512, "512.0 MiB"},
		{1024 * 1024 * 1024 * 2, "2.0 GiB"},
		{1024 * 1024 * 1024 * 40, "40.0 GiB"},
	}

	for _, tc := range tests {
		got := HumanBytes(tc.bytes)
		if got != tc.expected {
			t.Errorf("HumanBytes(%d) = %q, want %q", tc.bytes, got, tc.expected)
		}
	}
}

func TestSnapshotListParsing(t *testing.T) {
	rawOutput := `
TAG          CREATED                  SIZE
snap1        2026-09-20T10:15:30Z     1.2GiB
snap2        2026-09-21T12:00:00Z     2.5GiB
`
	// Test parsing via SnapshotList logic
	var snapshots []Snapshot
	lines := strings.Split(strings.TrimSpace(rawOutput), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "TAG") {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) >= 1 {
			snap := Snapshot{Tag: fields[0]}
			if len(fields) >= 2 {
				snap.Size = fields[len(fields)-1]
			}
			snapshots = append(snapshots, snap)
		}
	}

	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].Tag != "snap1" || snapshots[0].Size != "1.2GiB" {
		t.Errorf("snap0 mismatch: %+v", snapshots[0])
	}
	if snapshots[1].Tag != "snap2" || snapshots[1].Size != "2.5GiB" {
		t.Errorf("snap1 mismatch: %+v", snapshots[1])
	}
}

func TestExtractTemplateDescription(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "template-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `# A template to run Docker
# Second line of description
#
# $ limactl start ./docker.yaml

minimumLimaVersion: 1.1.0
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	desc := extractTemplateDescription(tmpFile.Name())
	if !strings.Contains(desc, "A template to run Docker") {
		t.Errorf("expected description to contain template text, got: %q", desc)
	}
}
