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

	node, exists := graph.nodes["op_1"]
	if !exists {
		t.Errorf("node op_1 not found")
	}

	if node.Operation.Kind() != OpCreateTable {
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
