package diff

import (
	"fmt"
	"sort"
)

// Planner orders operations by dependency and flags destructive changes.
type Planner struct{}

// NewPlanner creates a new planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// Plan represents an ordered migration plan.
type Plan struct {
	Operations  []Operation
	Destructive []Operation
	Warnings    []string
}

// Plan takes unordered operations and returns an ordered plan.
func (p *Planner) Plan(ops []Operation) (*Plan, error) {
	// Build dependency graph
	graph := p.buildDependencyGraph(ops)

	// Topologically sort operations
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to sort operations: %w", err)
	}

	// Classify destructive operations
	var destructive []Operation
	var warnings []string

	for _, op := range sorted {
		level := op.IsDestructive()
		if level >= PotentiallyDestructive {
			destructive = append(destructive, op)
			warnings = append(warnings, fmt.Sprintf("%s: %s (%s)",
				op.Kind(), op.Description(), level))
		}
	}

	return &Plan{
		Operations:  sorted,
		Destructive: destructive,
		Warnings:    warnings,
	}, nil
}

// buildDependencyGraph builds a dependency graph from operations.
func (p *Planner) buildDependencyGraph(ops []Operation) *DependencyGraph {
	graph := NewDependencyGraph()

	// Add all operations as nodes
	for i, op := range ops {
		nodeID := fmt.Sprintf("op_%d", i)
		graph.AddNode(nodeID, op)
	}

	// Add edges based on dependency rules
	// For MVP, we use a simple priority-based ordering:
	// 1. CREATE ENUM (must happen before tables that use them)
	// 2. DROP FOREIGN KEY (before dropping tables)
	// 3. CREATE TABLE
	// 4. DROP TABLE
	// 5. ADD/DROP COLUMN
	// 6. ALTER COLUMN
	// 7. CREATE INDEX
	// 8. DROP INDEX
	// 9. CREATE FOREIGN KEY (after tables exist)

	priorities := make(map[string]int)
	for i, op := range ops {
		nodeID := fmt.Sprintf("op_%d", i)
		priorities[nodeID] = p.getOperationPriority(op)
	}

	// Simple ordering: lower priority executes first
	graph.priorities = priorities

	return graph
}

// getOperationPriority returns the execution priority for an operation.
// Lower numbers execute first.
func (p *Planner) getOperationPriority(op Operation) int {
	switch op.Kind() {
	case OpCreateEnum:
		return 1
	case OpDropForeignKey:
		return 2
	case OpDropIndex:
		return 3
	case OpCreateTable:
		return 4
	case OpAddColumn:
		return 5
	case OpAlterColumn:
		return 6
	case OpRenameColumn:
		return 7
	case OpDropColumn:
		return 8
	case OpDropTable:
		return 9
	case OpCreateIndex:
		return 10
	case OpCreateForeignKey:
		return 11
	case OpAlterEnum:
		return 12
	case OpRenameTable:
		return 13
	default:
		return 100
	}
}

// DependencyGraph represents operation dependencies.
type DependencyGraph struct {
	nodes      map[string]*GraphNode
	priorities map[string]int
}

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID        string
	Operation Operation
	DependsOn []string
}

// NewDependencyGraph creates a new dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:      make(map[string]*GraphNode),
		priorities: make(map[string]int),
	}
}

// AddNode adds a node to the graph.
func (g *DependencyGraph) AddNode(id string, op Operation) {
	g.nodes[id] = &GraphNode{
		ID:        id,
		Operation: op,
		DependsOn: []string{},
	}
}

// AddEdge adds a dependency edge (from depends on to).
func (g *DependencyGraph) AddEdge(from, to string) {
	if node, exists := g.nodes[from]; exists {
		node.DependsOn = append(node.DependsOn, to)
	}
}

// TopologicalSort returns operations in dependency order.
func (g *DependencyGraph) TopologicalSort() ([]Operation, error) {
	// For MVP, we use a simple priority-based sort
	// In a full implementation, we would use Kahn's algorithm or DFS

	type nodeWithPriority struct {
		id       string
		priority int
		op       Operation
	}

	var nodes []nodeWithPriority
	for id, node := range g.nodes {
		priority := g.priorities[id]
		nodes = append(nodes, nodeWithPriority{
			id:       id,
			priority: priority,
			op:       node.Operation,
		})
	}

	// Sort by priority
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].priority < nodes[j].priority
	})

	// Extract operations
	result := make([]Operation, len(nodes))
	for i, n := range nodes {
		result[i] = n.op
	}

	return result, nil
}
