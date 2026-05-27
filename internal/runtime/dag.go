package runtime

import (
	"fmt"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// executionGraph represents a DAG of workflow steps
type executionGraph struct {
	nodes map[string][]string // node -> dependencies
	order []string            // topological order
}

// buildExecutionGraph builds a DAG from workflow steps
func buildExecutionGraph(steps []workflow.Step) (*executionGraph, error) {
	graph := &executionGraph{
		nodes: make(map[string][]string),
	}

	// Build adjacency list
	for _, step := range steps {
		graph.nodes[step.ID] = step.DependsOn
	}

	// Perform topological sort
	order, err := topologicalSort(graph.nodes)
	if err != nil {
		return nil, err
	}

	graph.order = order
	return graph, nil
}

// topologicalSort performs topological sort on the graph
// graph maps node -> list of dependencies (nodes it depends on)
func topologicalSort(graph map[string][]string) ([]string, error) {
	// Calculate in-degree for each node (number of dependencies)
	inDegree := make(map[string]int)
	for node := range graph {
		inDegree[node] = len(graph[node])
	}

	// Build reverse adjacency: dep -> list of nodes that depend on it
	dependents := make(map[string][]string)
	for node, deps := range graph {
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], node)
		}
	}

	// Find all nodes with no dependencies
	queue := make([]string, 0)
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	// Process nodes
	result := make([]string, 0, len(graph))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Decrease in-degree for nodes that depend on this one
		for _, dependent := range dependents[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for cycles
	if len(result) != len(graph) {
		return nil, fmt.Errorf("cycle detected in workflow dependencies")
	}

	return result, nil
}

// topologicalOrder returns the execution order
func (g *executionGraph) topologicalOrder() []string {
	return g.order
}

// dependenciesSatisfied checks if all dependencies for a step are satisfied
func (g *executionGraph) dependenciesSatisfied(stepID string, completedSteps map[string]workflow.StepResult) bool {
	return len(g.missingDependencies(stepID, completedSteps)) == 0
}

func (g *executionGraph) missingDependencies(stepID string, completedSteps map[string]workflow.StepResult) []string {
	deps, exists := g.nodes[stepID]
	if !exists {
		return nil
	}

	missing := make([]string, 0)
	for _, dep := range deps {
		_, completed := completedSteps[dep]
		if !completed {
			missing = append(missing, dep)
		}
	}

	return missing
}

// GetExecutionOrder returns the execution order for a workflow
func GetExecutionOrder(wf *workflow.Workflow) ([]string, error) {
	graph, err := buildExecutionGraph(wf.Steps)
	if err != nil {
		return nil, err
	}
	return graph.topologicalOrder(), nil
}
