package cli

import (
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/pkg/git"
	"github.com/rkuthiala/shiro-automation/pkg/gitlab"
	"github.com/rkuthiala/shiro-automation/pkg/print"
	"github.com/rkuthiala/shiro-automation/pkg/shell"
	"github.com/rkuthiala/shiro-automation/pkg/slack"
)

// registerAllModules registers all core built-in modules.
// External modules are loaded at runtime as subprocess plugins from .shiro/plugins/ or PATH.
func registerAllModules(registry *modules.Registry) error {
	if err := registry.Register("slack.notify", slack.NewSlackModule(false)); err != nil {
		return err
	}
	if err := registry.Register("git.diff", git.NewGitModule()); err != nil {
		return err
	}
	if err := registry.Register("print", print.NewPrintModule()); err != nil {
		return err
	}
	if err := registry.Register("shell.exec", shell.NewShellModule()); err != nil {
		return err
	}
	if err := registry.Register("gitlab", gitlab.NewGitLabModule()); err != nil {
		return err
	}
	return nil
}
