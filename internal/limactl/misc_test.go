package limactl

import (
	"encoding/json"
	"testing"
)

func TestDecodeInfoJSON(t *testing.T) {
	fixture := `{
  "version": "v1.2.3",
  "defaultTemplate": {
    "vmType": "qemu",
    "cpus": 4,
    "memory": "4GiB"
  },
  "limaHome": "/home/user/.lima",
  "vmTypes": ["qemu", "vz"],
  "guestAgents": {
    "x86_64": {
      "location": "/usr/local/share/lima/lima-guestagent.Linux-x86_64.gz"
    }
  },
  "templates": [
    {
      "name": "default",
      "location": "/usr/local/share/lima/templates/default.yaml"
    },
    {
      "name": "docker",
      "location": "/usr/local/share/lima/templates/docker.yaml"
    }
  ]
}`

	var diag InfoDiagnostic
	if err := json.Unmarshal([]byte(fixture), &diag); err != nil {
		t.Fatalf("failed to unmarshal info JSON: %v", err)
	}

	if diag.Version != "v1.2.3" {
		t.Errorf("version mismatch: %s", diag.Version)
	}
	if len(diag.Templates) != 2 {
		t.Errorf("templates count mismatch: %d", len(diag.Templates))
	}
	if len(diag.VMTypes) != 2 {
		t.Errorf("vmTypes count mismatch: %d", len(diag.VMTypes))
	}
	if _, ok := diag.GuestAgents["x86_64"]; !ok {
		t.Errorf("expected x86_64 in guestAgents")
	}
}
