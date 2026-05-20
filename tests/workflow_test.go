package tests

import (
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

func TestLoadWorkflow(t *testing.T) {
	validJSON := []byte(`{
		"name": "test-workflow",
		"description": "A test workflow",
		"steps": [
			{
				"id": "step1",
				"type": "test",
				"config": {"key": "value"}
			}
		]
	}`)

	wf, err := workflow.LoadWorkflow(validJSON)
	if err != nil {
		t.Fatalf("Failed to load valid workflow: %v", err)
	}

	if wf.Name != "test-workflow" {
		t.Errorf("Expected name 'test-workflow', got '%s'", wf.Name)
	}

	if len(wf.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(wf.Steps))
	}
}

func TestLoadWorkflowInvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{invalid json}`)

	_, err := workflow.LoadWorkflow(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestWorkflowValidate(t *testing.T) {
	tests := []struct {
		name    string
		wf      *workflow.Workflow
		wantErr bool
	}{
		{
			name: "valid workflow",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			wf: &workflow.Workflow{
				Steps: []workflow.Step{
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "no steps",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{},
			},
			wantErr: true,
		},
		{
			name: "missing step ID",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{
					{Type: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing step type",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{
					{ID: "step1"},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate step ID",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{
					{ID: "step1", Type: "test"},
					{ID: "step1", Type: "test2"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid dependency",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{
					{ID: "step1", Type: "test"},
					{ID: "step2", Type: "test2", DependsOn: []string{"step1"}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid dependency",
			wf: &workflow.Workflow{
				Name: "test",
				Steps: []workflow.Step{
					{ID: "step1", Type: "test"},
					{ID: "step2", Type: "test2", DependsOn: []string{"nonexistent"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wf.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetStepByID(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Steps: []workflow.Step{
			{ID: "step1", Type: "test"},
			{ID: "step2", Type: "test2"},
		},
	}

	step := wf.GetStepByID("step1")
	if step == nil {
		t.Fatal("Expected to find step1, got nil")
	}

	if step.ID != "step1" {
		t.Errorf("Expected step ID 'step1', got '%s'", step.ID)
	}

	nonexistent := wf.GetStepByID("nonexistent")
	if nonexistent != nil {
		t.Error("Expected nil for nonexistent step, got step")
	}
}

func TestNewExecutionContext(t *testing.T) {
	ctx := workflow.NewExecutionContext()

	if ctx.Inputs == nil {
		t.Error("Expected Inputs to be initialized")
	}

	if ctx.Steps == nil {
		t.Error("Expected Steps to be initialized")
	}

	if ctx.Memory == nil {
		t.Error("Expected Memory to be initialized")
	}

	if ctx.Env == nil {
		t.Error("Expected Env to be initialized")
	}
}

func TestRetryConfig(t *testing.T) {
	config := &workflow.RetryConfig{
		MaxAttempts: 3,
		Delay:       5,
		Backoff:     2.0,
		MaxDelay:    60,
	}

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", config.MaxAttempts)
	}

	if config.Delay != 5 {
		t.Errorf("Expected Delay 5, got %d", config.Delay)
	}
}

func TestStepResult(t *testing.T) {
	result := workflow.StepResult{
		Success: true,
		Output:  map[string]interface{}{"key": "value"},
		Error:   "",
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Output["key"] != "value" {
		t.Error("Expected output key to be 'value'")
	}
}
