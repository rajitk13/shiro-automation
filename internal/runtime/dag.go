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
func topologicalSort(graph map[string][]string) ([]string, error) {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for node := range graph {
		inDegree[node] = 0
	}

	for _, deps := range graph {
		for _, dep := range deps {
			inDegree[dep]++
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

		// Decrease in-degree for dependent nodes
		for _, dep := range graph[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
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
	deps, exists := g.nodes[stepID]
	if !exists {
		return true
	}

	for _, dep := range deps {
		_, completed := completedSteps[dep]
		if !completed {
			return false
		}
	}

	return true
}
