package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

// QueryTemplate represents a query template definition.
type QueryTemplate struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Category    string      `yaml:"category"`
	Filter      string      `yaml:"filter"`
	Examples    []Example   `yaml:"examples"`
	Parameters  []Parameter `yaml:"parameters"`
}

// Example represents a usage example for a query template.
type Example struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
}

// Parameter represents a parameter for a query template.
type Parameter struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Optional    bool   `yaml:"optional"`
}

// LoadTemplates loads all query templates from the templates directory.
func LoadTemplates() (map[string]QueryTemplate, error) {
	templates := make(map[string]QueryTemplate)

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	possiblePaths := []string{
		filepath.Join(cwd, "internal", "query", "templates"),
		filepath.Join(cwd, "query", "templates"),
		"internal/query/templates",
		"templates",
	}

	var foundTemplatesDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			foundTemplatesDir = path
			break
		}
	}

	if foundTemplatesDir == "" {
		return templates, nil
	}

	files, err := os.ReadDir(foundTemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		templatePath := filepath.Join(foundTemplatesDir, file.Name())
		data, err := os.ReadFile(templatePath)
		if err != nil {
			continue
		}

		var tmpl QueryTemplate
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			continue
		}

		templates[tmpl.Name] = tmpl
	}

	return templates, nil
}
