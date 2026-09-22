package limactl

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	valid := []string{"ubuntu", "vm-1", "test_vm.2", "Rocky9", "default"}
	for _, n := range valid {
		if err := ValidateName(n); err != nil {
			t.Errorf("expected valid name %q, got err: %v", n, err)
		}
	}

	invalid := []string{"", "  ", "-startwithdash", "_leadingund", "has space", "bad@symbol", strings.Repeat("a", 64)}
	for _, n := range invalid {
		if err := ValidateName(n); err == nil {
			t.Errorf("expected invalid name for %q, got nil", n)
		}
	}
}

func TestDecodeNDJSON(t *testing.T) {
	ndjsonInput := `time="2026-09-22T14:53:47+02:00" level=warning msg="Non-strict YAML detected"
{"name":"vm1","status":"Running","cpus":2,"memory":2147483648,"disk":42949672960}
{"name":"vm2","status":"Stopped","cpus":4,"memory":4294967296,"disk":64424509440}
random non json line
`
	items, err := DecodeNDJSON[Instance](strings.NewReader(ndjsonInput))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Name != "vm1" || items[0].Status != "Running" || items[0].CPUs != 2 {
		t.Errorf("item 0 mismatch: %+v", items[0])
	}
	if items[1].Name != "vm2" || items[1].Status != "Stopped" || items[1].CPUs != 4 {
		t.Errorf("item 1 mismatch: %+v", items[1])
	}
}
