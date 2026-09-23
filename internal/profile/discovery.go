package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultProfileFileName is the conventional profile file name a harness
// repository ships at its root.
const DefaultProfileFileName = "lima-ui.profile.yaml"

// Config is the optional user configuration file that lists where to look for
// profiles, so the Profiles screen can show several without passing --profile
// every time.
//
// Expected at ~/.config/limavm-tui/config.yaml:
//
//	profilePaths:
//	  - ~/sources/pro/rdb-dev
//	  - ~/sources/pro/other-harness/lima-ui.profile.yaml
type Config struct {
	ProfilePaths []string `yaml:"profilePaths"`
}

// LoadResult pairs a profile source with the outcome of loading it, so the UI
// can surface a broken profile instead of silently hiding it.
type LoadResult struct {
	Path    string
	Profile *Profile
	Err     error
}

// DefaultConfigPath returns the location of the optional user config file.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".config", "limavm-tui", "config.yaml"), nil
}

// LoadConfig reads the optional user config file. A missing file is not an
// error: the config is purely additive.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return &cfg, nil
}

// Discover resolves every requested source into load results.
//
// A source may be either a profile file or a directory containing one named
// DefaultProfileFileName. Explicit sources (typically --profile flags) come
// first and take precedence; configured search paths are appended. Duplicate
// resolved paths are collapsed, keeping the first occurrence.
func Discover(explicit []string, configured []string) []LoadResult {
	var results []LoadResult
	seen := make(map[string]struct{})

	add := func(src string) {
		src = strings.TrimSpace(src)
		if src == "" {
			return
		}

		resolved, err := resolveSource(src)
		if err != nil {
			// Keep the original spelling in the error so the user recognises it.
			results = append(results, LoadResult{Path: src, Err: err})
			return
		}
		if _, dup := seen[resolved]; dup {
			return
		}
		seen[resolved] = struct{}{}

		p, err := Load(resolved)
		results = append(results, LoadResult{Path: resolved, Profile: p, Err: err})
	}

	for _, s := range explicit {
		add(s)
	}
	for _, s := range configured {
		add(s)
	}

	return results
}

// resolveSource expands a source into the concrete profile file path it refers
// to, accepting both a file and a directory holding the conventional file.
func resolveSource(src string) (string, error) {
	expanded, err := expandHome(src)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", src, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("profile source %q not found", src)
		}
		return "", fmt.Errorf("cannot stat %q: %w", src, err)
	}

	if info.IsDir() {
		candidate := filepath.Join(abs, DefaultProfileFileName)
		if _, err := os.Stat(candidate); err != nil {
			return "", fmt.Errorf("no %s found in directory %q", DefaultProfileFileName, src)
		}
		return candidate, nil
	}

	return abs, nil
}

// expandHome expands a leading ~ so config files can stay portable.
func expandHome(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	if p == "~" {
		return home, nil
	}
	return filepath.Join(home, p[2:]), nil
}

// Valid extracts the successfully loaded profiles, sorted by name for a stable
// display order.
func Valid(results []LoadResult) []*Profile {
	var out []*Profile
	for _, r := range results {
		if r.Err == nil && r.Profile != nil {
			out = append(out, r.Profile)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Failed extracts the sources that could not be loaded.
func Failed(results []LoadResult) []LoadResult {
	var out []LoadResult
	for _, r := range results {
		if r.Err != nil {
			out = append(out, r)
		}
	}
	return out
}
