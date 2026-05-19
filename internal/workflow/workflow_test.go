package workflow

import (
	"testing"
)

func TestWorkflowValidation(t *testing.T) {
	tests := []struct {
		name    string
		wf      *Workflow
		wantErr bool
	}{
		{
			name: "valid workflow",
			wf: &Workflow{
				Name: "test-workflow",
				Steps: []Step{
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			wf: &Workflow{
				Steps: []Step{
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "no steps",
			wf: &Workflow{
				Name: "test-workflow",
			},
			wantErr: true,
		},
		{
			name: "duplicate step IDs",
			wf: &Workflow{
				Name: "test-workflow",
				Steps: []Step{
					{ID: "step1", Type: "test"},
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "circular dependency",
			wf: &Workflow{
				Name: "test-workflow",
				Steps: []Step{
					{ID: "step1", Type: "test", DependsOn: []string{"step2"}},
					{ID: "step2", Type: "test", DependsOn: []string{"step1"}},
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

func TestLoadWorkflow(t *testing.T) {
	jsonData := `{
		"name": "test-workflow",
		"steps": [
			{
				"id": "step1",
				"type": "test"
			}
		]
	}`

	wf, err := LoadWorkflow([]byte(jsonData))
	if err != nil {
		t.Fatalf("LoadWorkflow() error = %v", err)
	}

	if wf.Name != "test-workflow" {
		t.Errorf("Expected name 'test-workflow', got '%s'", wf.Name)
	}

	if len(wf.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(wf.Steps))
	}
}

func TestGetStepByID(t *testing.T) {
	wf := &Workflow{
		Name: "test-workflow",
		Steps: []Step{
			{ID: "step1", Type: "test"},
			{ID: "step2", Type: "test"},
		},
	}

	step := wf.GetStepByID("step1")
	if step == nil {
		t.Fatal("GetStepByID() returned nil for existing step")
	}

	if step.ID != "step1" {
		t.Errorf("Expected step ID 'step1', got '%s'", step.ID)
	}

	step = wf.GetStepByID("nonexistent")
	if step != nil {
		t.Error("GetStepByID() returned non-nil for non-existent step")
	}
}
