package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rkuthiala/shiro-automation/internal/modules"
)

// BuildCommand handles the 'shiro build' command
func BuildCommand(args []string) {
	// Parse flags
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	output := fs.String("o", "./shiro", "Output binary path")
	ldflags := fs.String("ldflags", "", "Flags to pass to go linker")

	if err := fs.Parse(args); err != nil {
		printBuildHelp()
		os.Exit(1)
	}

	// Get shiro directory
	shiroDir := ".shiro"
	if envDir := os.Getenv("SHIRO_DIR"); envDir != "" {
		shiroDir = envDir
	}

	registryPath := filepath.Join(shiroDir, "modules", "registry.yaml")
	outputPath := "internal/cli/registry.go"

	// Step 1: Generate registry code
	fmt.Println("Generating module registry...")
	if err := modules.GenerateRegistry(registryPath, outputPath); err != nil {
		log.Fatalf("✗ Failed to generate registry: %v", err)
	}
	fmt.Println("✓ Generated internal/cli/registry.go")

	// Step 2: Run go mod tidy
	fmt.Println("Tidying Go modules...")
	if err := tidyModules(); err != nil {
		log.Fatalf("✗ Failed to tidy modules: %v", err)
	}
	fmt.Println("✓ Tidied modules")

	// Step 3: Build binary
	fmt.Println("Building shiro binary...")
	if err := buildBinary(*output, *ldflags); err != nil {
		log.Fatalf("✗ Build failed: %v", err)
	}

	// Step 4: Report success
	fmt.Printf("\n✓ Built %s\n", *output)
	fmt.Println("\nRegistered modules:")
	listModulesForBuild()
}

// tidyModules runs go mod tidy
func tidyModules() error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildBinary compiles the shiro binary
func buildBinary(output, ldflags string) error {
	args := []string{"build", "-o", output}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "./cmd/runtime")

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// listModulesForBuild lists registered modules for build output
func listModulesForBuild() {
	// Core modules
	coreModules := []string{
		"ai.generate",
		"slack.notify",
		"git.diff",
		"print",
		"shell.exec",
		"data.store",
		"data.load",
		"data.delete",
		"data.exists",
	}

	// Load registry to get external modules
	shiroDir := ".shiro"
	if envDir := os.Getenv("SHIRO_DIR"); envDir != "" {
		shiroDir = envDir
	}
	registryPath := filepath.Join(shiroDir, "modules", "registry.yaml")

	discoverer := modules.NewDiscoverer(registryPath, nil)
	var externalModules []string

	if err := discoverer.LoadRegistry(); err == nil {
		mods := discoverer.ListModules()
		for _, name := range mods {
			if !isCoreModule(name, coreModules) {
				config, _ := discoverer.GetModuleConfig(name)
				if config.Type == "builtin" {
					externalModules = append(externalModules, name)
				}
			}
		}
	}

	// Print core modules
	fmt.Println("  Core:")
	for _, m := range coreModules {
		fmt.Printf("    - %s\n", m)
	}

	// Print external modules
	if len(externalModules) > 0 {
		fmt.Println("  External (added via 'shiro add module'):")
		for _, m := range externalModules {
			fmt.Printf("    - %s\n", m)
		}
	}
}

// isCoreModule checks if a module is in the core list
func isCoreModule(name string, core []string) bool {
	for _, c := range core {
		if c == name {
			return true
		}
	}
	return false
}

// printBuildHelp prints help for the build command
func printBuildHelp() {
	fmt.Print(`Usage: shiro build [options]

Build shiro with all registered modules (core and external).

This command:
  1. Generates internal/cli/registry.go
  2. Runs 'go mod tidy' to fetch external modules
  3. Compiles the shiro binary

Options:
  -o <path>       Output binary path (default: ./shiro)
  -ldflags <flags> Pass flags to go linker
  -help           Show this help message

Examples:
  shiro build
  shiro build -o /usr/local/bin/shiro
  shiro build -ldflags "-s -w" -o ./dist/shiro

Environment Variables:
  SHIRO_DIR       Path to .shiro directory (default: ./.shiro)

After building, external modules added via 'shiro add module' will be
available in workflows using "type": "<module-name>".
`)
}
