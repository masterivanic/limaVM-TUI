package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRealHarnessProfile validates the profile shipped by the rdb-dev harness
// against this loader, when that harness is checked out next to this repo.
//
// It is skipped when the harness is absent, so it never breaks CI, but it
// catches a drift between the schema here and the file shipped there.
func TestRealHarnessProfile(t *testing.T) {
	candidates := []string{
		filepath.Join("..", "..", "..", "rdb-dev", DefaultProfileFileName),
	}
	if env := os.Getenv("LIMAVM_TEST_PROFILE"); env != "" {
		candidates = append([]string{env}, candidates...)
	}

	var path string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			path = c
			break
		}
	}
	if path == "" {
		t.Skip("harness profile not available next to this repository")
	}

	p, err := Load(path)
	if err != nil {
		t.Fatalf("real harness profile failed to load: %v", err)
	}

	if p.Lifecycle.Create == nil {
		t.Error("expected a lifecycle.create hook")
	}
	if p.Lifecycle.HealthCheck == nil {
		t.Error("expected a lifecycle.healthCheck hook")
	}
	if len(p.ProvisioningModes) == 0 {
		t.Error("expected at least one provisioning mode")
	}

	// Every referenced script must resolve, exist and be executable.
	check := func(label string, c Command) {
		t.Helper()
		if _, err := p.ResolveScript(c.Script); err != nil {
			t.Errorf("%s: %v", label, err)
		}
	}
	check("lifecycle.create", *p.Lifecycle.Create)
	check("lifecycle.healthCheck", *p.Lifecycle.HealthCheck)
	for _, m := range p.ProvisioningModes {
		for i, s := range m.Steps {
			check(m.Name+" step", s)
			_ = i
		}
	}

	t.Logf("profile %q OK: vm=%s modes=%v", p.Name, p.VMName, p.ModeNames())
	t.Logf("create argv: %q", p.ExpandArgs(*p.Lifecycle.Create))
	t.Logf("healthCheck argv: %q", p.ExpandArgs(*p.Lifecycle.HealthCheck))
}
