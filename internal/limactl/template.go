package limactl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ListTemplates discovers available Lima templates and enriches them with descriptions.
func (c *Client) ListTemplates(ctx context.Context) ([]Template, error) {
	// First attempt: retrieve from info
	info, err := c.Info(ctx)
	if err == nil && len(info.Templates) > 0 {
		defaultName := "default"
		if s, ok := info.DefaultTemplate.(string); ok && s != "" {
			defaultName = s
		}
		templates := make([]Template, 0, len(info.Templates))
		for _, loc := range info.Templates {
			if strings.HasPrefix(loc.Name, "_") {
				continue
			}
			desc := extractTemplateDescription(loc.Location)
			t := Template{
				Name:        loc.Name,
				Location:    loc.Location,
				Description: desc,
				IsDefault:   loc.Name == defaultName || loc.Name == "default",
			}
			templates = append(templates, t)
		}
		if len(templates) > 0 {
			sort.Slice(templates, func(i, j int) bool {
				if templates[i].IsDefault != templates[j].IsDefault {
					return templates[i].IsDefault
				}
				return templates[i].Name < templates[j].Name
			})
			return templates, nil
		}
	}

	// Fallback attempt: limactl create --list-templates
	out, err := c.Run(ctx, "create", "--list-templates")
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	var templates []Template
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasPrefix(name, "time=") || strings.HasPrefix(name, "_") {
			continue
		}
		templates = append(templates, Template{
			Name:        name,
			Description: fmt.Sprintf("Standard %s template", name),
			IsDefault:   name == "default",
		})
	}

	sort.Slice(templates, func(i, j int) bool {
		if templates[i].IsDefault != templates[j].IsDefault {
			return templates[i].IsDefault
		}
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// extractTemplateDescription reads leading comment lines from a template YAML file.
func extractTemplateDescription(filePath string) string {
	if filePath == "" {
		return ""
	}

	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var descLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			clean := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if clean != "" {
				descLines = append(descLines, clean)
			}
		} else if line != "" {
			// Hit first non-comment non-empty YAML line
			break
		}
	}

	if len(descLines) == 0 {
		return "Standard Lima VM template"
	}

	return strings.Join(descLines, "\n")
}

// ValidateTemplate runs `limactl validate` on a template or YAML file.
func (c *Client) ValidateTemplate(ctx context.Context, pathOrURL string) error {
	pathOrURL = strings.TrimSpace(pathOrURL)
	if pathOrURL == "" {
		return fmt.Errorf("template path or URL cannot be empty")
	}

	_, err := c.Run(ctx, "validate", pathOrURL)
	return err
}
