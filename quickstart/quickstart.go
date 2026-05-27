package quickstart

import (
	"fmt"
	"os"
)

// Template defines the interface for quickstart templates
type Template interface {
	Name() string
	Description() string
	Initialize(interactive, directConfig bool, configArgs []string) error
}

// Registry holds all available templates
var templates = make(map[string]Template)

// Register adds a template to the registry
func Register(name string, template Template) {
	templates[name] = template
}

// Get retrieves a template by name
func Get(name string) (Template, bool) {
	t, ok := templates[name]
	return t, ok
}

// List returns all registered template names
func List() []string {
	var names []string
	for name := range templates {
		names = append(names, name)
	}
	return names
}

// Initialize runs a template initialization
func Initialize(name string, interactive, directConfig bool, configArgs []string) error {
	template, ok := Get(name)
	if !ok {
		return fmt.Errorf("template '%s' not found. Available templates: %v", name, List())
	}

	fmt.Printf("Initializing %s template...\n", template.Name())
	return template.Initialize(interactive, directConfig, configArgs)
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// WriteFile writes content to a file
func WriteFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	return nil
}

func init() {
	Register("code-review", NewCodeReviewTemplate())
}
