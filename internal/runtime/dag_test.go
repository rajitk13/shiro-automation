package runtime

import (
	"reflect"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

func TestBuildExecutionGraphOrdersDependenciesBeforeDependents(t *testing.T) {
	graph, err := buildExecutionGraph([]workflow.Step{
		{ID: "ai_decision", Type: "test"},
		{ID: "approval", Type: "test", DependsOn: []string{"ai_decision"}},
		{ID: "final_action", Type: "test", DependsOn: []string{"approval"}},
		{ID: "notify_complete", Type: "test", DependsOn: []string{"final_action"}},
	})
	if err != nil {
		t.Fatalf("buildExecutionGraph() error = %v", err)
	}

	want := []string{"ai_decision", "approval", "final_action", "notify_complete"}
	if !reflect.DeepEqual(graph.topologicalOrder(), want) {
		t.Fatalf("topologicalOrder() = %v, want %v", graph.topologicalOrder(), want)
	}
}

func TestMissingDependencies(t *testing.T) {
	graph, err := buildExecutionGraph([]workflow.Step{
		{ID: "first", Type: "test"},
		{ID: "second", Type: "test", DependsOn: []string{"first"}},
	})
	if err != nil {
		t.Fatalf("buildExecutionGraph() error = %v", err)
	}

	if graph.dependenciesSatisfied("second", map[string]workflow.StepResult{}) {
		t.Fatal("dependenciesSatisfied() = true, want false")
	}

	missing := graph.missingDependencies("second", map[string]workflow.StepResult{})
	if !reflect.DeepEqual(missing, []string{"first"}) {
		t.Fatalf("missingDependencies() = %v, want [first]", missing)
	}

	if !graph.dependenciesSatisfied("second", map[string]workflow.StepResult{"first": {Success: false}}) {
		t.Fatal("dependenciesSatisfied() = false, want true for previously executed dependency")
	}
}
