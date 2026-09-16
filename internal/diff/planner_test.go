package diff

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestPlanner_Plan(t *testing.T) {
	tests := []struct {
		name        string
		ops         []Operation
		wantOrdered []OperationKind
		wantDestructive int
	}{
		{
			name: "simple ordering",
			ops: []Operation{
				CreateTable{Table: schema.Table{Name: "users"}},
				CreateEnum{Enum: schema.Enum{Name: "status"}},
				CreateIndex{Table: "users", Index: schema.Index{}},
			},
			wantOrdered: []OperationKind{
				OpCreateEnum,
				OpCreateTable,
				OpCreateIndex,
			},
			wantDestructive: 0,
		},
		{
			name: "with foreign keys",
			ops: []Operation{
				CreateForeignKey{Table: "posts", Constraint: schema.ForeignKey{RefTable: "users"}},
				CreateTable{Table: schema.Table{Name: "users"}},
				CreateTable{Table: schema.Table{Name: "posts"}},
			},
			wantOrdered: []OperationKind{
				OpCreateTable,
				OpCreateTable,
				OpCreateForeignKey,
			},
			wantDestructive: 0, // CreateForeignKey is Risky, not PotentiallyDestructive
		},
		{
			name: "with drops",
			ops: []Operation{
				DropTable{Name: "posts"},
				DropForeignKey{Table: "posts", Name: "fk_user"},
			},
			wantOrdered: []OperationKind{
				OpDropForeignKey,
				OpDropTable,
			},
			wantDestructive: 1, // DropTable is destructive
		},
		{
			name: "destructive operations",
			ops: []Operation{
				DropColumn{Table: "users", Column: "email"},
				AddColumn{
					Table: "users",
					Column: schema.Column{
						Name:     "required_field",
						Nullable: false,
						Default:  nil,
					},
				},
			},
			wantOrdered: []OperationKind{
				OpAddColumn,
				OpDropColumn,
			},
			wantDestructive: 2, // Both are destructive/potentially destructive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := NewPlanner()
			plan, err := planner.Plan(tt.ops)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}

			// Check ordering
			if len(plan.Operations) != len(tt.wantOrdered) {
				t.Errorf("got %d operations, want %d", len(plan.Operations), len(tt.wantOrdered))
			}

			for i, wantKind := range tt.wantOrdered {
				if i >= len(plan.Operations) {
					break
				}
				if plan.Operations[i].Kind() != wantKind {
					t.Errorf("operation[%d]: got kind %v, want %v",
						i, plan.Operations[i].Kind(), wantKind)
				}
			}

			// Check destructive count
			if len(plan.Destructive) != tt.wantDestructive {
				t.Errorf("got %d destructive operations, want %d",
					len(plan.Destructive), tt.wantDestructive)
			}

			// Check warnings
			if len(plan.Warnings) != tt.wantDestructive {
				t.Errorf("got %d warnings, want %d",
					len(plan.Warnings), tt.wantDestructive)
			}
		})
	}
}

func TestPlanner_GetOperationPriority(t *testing.T) {
	planner := NewPlanner()

	tests := []struct {
		op       Operation
		wantLess Operation // This should have higher priority (lower number)
	}{
		{
			op:       CreateTable{Table: schema.Table{Name: "users"}},
			wantLess: CreateEnum{Enum: schema.Enum{Name: "status"}},
		},
		{
			op:       CreateForeignKey{Table: "posts"},
			wantLess: CreateTable{Table: schema.Table{Name: "posts"}},
		},
		{
			op:       DropTable{Name: "posts"},
			wantLess: DropForeignKey{Table: "posts", Name: "fk_user"},
		},
		{
			op:       CreateIndex{Table: "users"},
			wantLess: CreateTable{Table: schema.Table{Name: "users"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.op.Kind().String(), func(t *testing.T) {
			opPriority := planner.getOperationPriority(tt.op)
			lessPriority := planner.getOperationPriority(tt.wantLess)

			if opPriority <= lessPriority {
				t.Errorf("%v (priority %d) should execute after %v (priority %d)",
					tt.op.Kind(), opPriority, tt.wantLess.Kind(), lessPriority)
			}
		})
	}
}

func TestDependencyGraph_AddNode(t *testing.T) {
	graph := NewDependencyGraph()
	op := CreateTable{Table: schema.Table{Name: "users"}}

	graph.AddNode("op_1", op)

	if len(graph.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.nodes))
	}

	if node, exists := graph.nodes["op_1"]; !exists {
		t.Errorf("node op_1 not found")
	} else if node.Operation.Kind() != OpCreateTable {
		t.Errorf("expected OpCreateTable, got %v", node.Operation.Kind())
	}
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()

	op1 := CreateEnum{Enum: schema.Enum{Name: "status"}}
	op2 := CreateTable{Table: schema.Table{Name: "users"}}
	op3 := CreateIndex{Table: "users"}

	graph.AddNode("op_1", op1)
	graph.AddNode("op_2", op2)
	graph.AddNode("op_3", op3)

	planner := NewPlanner()
	graph.priorities = map[string]int{
		"op_1": planner.getOperationPriority(op1),
		"op_2": planner.getOperationPriority(op2),
		"op_3": planner.getOperationPriority(op3),
	}

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort() error = %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("expected 3 operations, got %d", len(sorted))
	}

	// Verify order: enum -> table -> index
	expectedOrder := []OperationKind{OpCreateEnum, OpCreateTable, OpCreateIndex}
	for i, expected := range expectedOrder {
		if sorted[i].Kind() != expected {
			t.Errorf("operation[%d]: got %v, want %v", i, sorted[i].Kind(), expected)
		}
	}
}

func TestDependencyGraph_AddEdge(t *testing.T) {
	graph := NewDependencyGraph()

	op1 := CreateEnum{Enum: schema.Enum{Name: "status"}}
	op2 := CreateTable{Table: schema.Table{Name: "users"}}

	graph.AddNode("op_1", op1)
	graph.AddNode("op_2", op2)

	// Add edge from op_2 (depends on) to op_1
	graph.AddEdge("op_2", "op_1")

	// Verify edge was added
	if node, exists := graph.nodes["op_2"]; exists {
		if len(node.DependsOn) != 1 {
			t.Errorf("expected 1 dependency, got %d", len(node.DependsOn))
		}
		if node.DependsOn[0] != "op_1" {
			t.Errorf("expected dependency on op_1, got %v", node.DependsOn[0])
		}
	} else {
		t.Errorf("node op_2 not found")
	}

	// Verify other node has no dependencies
	if node, exists := graph.nodes["op_1"]; exists {
		if len(node.DependsOn) != 0 {
			t.Errorf("expected 0 dependencies for op_1, got %d", len(node.DependsOn))
		}
	}
}

func TestDependencyGraph_AddEdge_NonexistentNode(t *testing.T) {
	graph := NewDependencyGraph()

	op1 := CreateEnum{Enum: schema.Enum{Name: "status"}}
	graph.AddNode("op_1", op1)

	// Add edge from non-existent node (should not panic, just skip)
	graph.AddEdge("op_999", "op_1")

	// Should have no effect
	if _, exists := graph.nodes["op_999"]; exists {
		t.Errorf("expected node op_999 to not exist")
	}
}

func TestDependencyGraph_AddEdge_Multiple(t *testing.T) {
	graph := NewDependencyGraph()

	op1 := CreateEnum{Enum: schema.Enum{Name: "status"}}
	op2 := CreateTable{Table: schema.Table{Name: "users"}}
	op3 := CreateIndex{Table: "users"}

	graph.AddNode("op_1", op1)
	graph.AddNode("op_2", op2)
	graph.AddNode("op_3", op3)

	// Add multiple edges to op_3
	graph.AddEdge("op_3", "op_1")
	graph.AddEdge("op_3", "op_2")

	// Verify multiple dependencies
	if node, exists := graph.nodes["op_3"]; exists {
		if len(node.DependsOn) != 2 {
			t.Errorf("expected 2 dependencies, got %d", len(node.DependsOn))
		}
	} else {
		t.Errorf("node op_3 not found")
	}
}
