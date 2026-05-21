package runtime

import (
	"context"
	"log"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

type captureModule struct {
	configs []map[string]interface{}
}

func (m *captureModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	wfStep := step.(workflow.Step)
	m.configs = append(m.configs, wfStep.Config)
	return map[string]interface{}{"text": wfStep.Config["message"]}, nil
}

func (m *captureModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{Name: "capture"}
}

func TestExecutorUsesResolvedConfigAndPauses(t *testing.T) {
	registry := modules.NewRegistry()
	module := &captureModule{}
	if err := registry.Register("capture", module); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	store := state.NewMemoryStore()
	executor := NewExecutor(registry, log.Default())
	executor.SetStateStore(store)

	wf := &workflow.Workflow{
		Name: "approval-test",
		Steps: []workflow.Step{
			{ID: "decision", Type: "capture", Config: map[string]interface{}{"message": "allow"}},
			{ID: "approval", Type: "capture", DependsOn: []string{"decision"}, Pause: true, Config: map[string]interface{}{"message": "Review {{steps.decision.text}}"}},
			{ID: "after", Type: "capture", DependsOn: []string{"approval"}, Config: map[string]interface{}{"message": "after"}},
		},
	}

	execCtx, err := executor.Execute(context.Background(), wf, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, ok := execCtx.Steps["after"]; ok {
		t.Fatal("step after pause executed during initial run")
	}
	if got := module.configs[1]["message"]; got != "Review allow" {
		t.Fatalf("resolved approval message = %v, want Review allow", got)
	}

	execCtx, err = executor.Execute(context.Background(), wf, nil, nil)
	if err != nil {
		t.Fatalf("resume Execute() error = %v", err)
	}
	if _, ok := execCtx.Steps["after"]; !ok {
		t.Fatal("step after pause did not execute during resume")
	}
}
