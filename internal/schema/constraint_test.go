package schema

import (
	"strings"
	"testing"
)

func TestPrimaryKey_Valid(t *testing.T) {
	pk := PrimaryKey{
		Name:    "pk_users",
		Columns: []string{"id"},
	}

	if pk.Name != "pk_users" {
		t.Errorf("expected name 'pk_users', got: %s", pk.Name)
	}
	if len(pk.Columns) != 1 || pk.Columns[0] != "id" {
		t.Errorf("expected columns [id], got: %v", pk.Columns)
	}
}

func TestPrimaryKey_Composite(t *testing.T) {
	pk := PrimaryKey{
		Name:    "pk_user_role",
		Columns: []string{"user_id", "role_id"},
	}

	if len(pk.Columns) != 2 {
		t.Errorf("expected 2 columns, got: %d", len(pk.Columns))
	}
}

func TestForeignKey_Validate_Valid(t *testing.T) {
	fk := ForeignKey{
		Name:       "fk_posts_user",
		Columns:    []string{"user_id"},
		RefTable:   "users",
		RefColumns: []string{"id"},
		OnDelete:   ActionCascade,
		OnUpdate:   ActionNoAction,
	}

	if err := fk.Validate(); err != nil {
		t.Errorf("expected valid FK, got error: %v", err)
	}
}

func TestForeignKey_Validate_EmptyName(t *testing.T) {
	fk := ForeignKey{
		Name:       "",
		Columns:    []string{"user_id"},
		RefTable:   "users",
		RefColumns: []string{"id"},
	}

	err := fk.Validate()
	if err == nil {
		t.Fatal("expected error for empty FK name, got nil")
	}

	if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("expected 'name cannot be empty' in error, got: %v", err)
	}
}

func TestForeignKey_Validate_NoColumns(t *testing.T) {
	fk := ForeignKey{
		Name:       "fk_test",
		Columns:    []string{},
		RefTable:   "users",
		RefColumns: []string{"id"},
	}

	err := fk.Validate()
	if err == nil {
		t.Fatal("expected error for FK with no columns, got nil")
	}

	if !strings.Contains(err.Error(), "has no columns") {
		t.Errorf("expected 'has no columns' in error, got: %v", err)
	}
}

func TestForeignKey_Validate_EmptyRefTable(t *testing.T) {
	fk := ForeignKey{
		Name:       "fk_test",
		Columns:    []string{"user_id"},
		RefTable:   "",
		RefColumns: []string{"id"},
	}

	err := fk.Validate()
	if err == nil {
		t.Fatal("expected error for empty RefTable, got nil")
	}

	if !strings.Contains(err.Error(), "empty reference table") {
		t.Errorf("expected 'empty reference table' in error, got: %v", err)
	}
}

func TestForeignKey_Validate_ColumnCountMismatch(t *testing.T) {
	tests := []struct {
		name       string
		fk         ForeignKey
		wantErrMsg string
	}{
		{
			name: "1 column vs 2 ref columns",
			fk: ForeignKey{
				Name:       "fk_test",
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id", "email"},
			},
			wantErrMsg: "1 columns but 2 reference columns",
		},
		{
			name: "2 columns vs 1 ref column",
			fk: ForeignKey{
				Name:       "fk_test",
				Columns:    []string{"user_id", "org_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantErrMsg: "2 columns but 1 reference columns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fk.Validate()
			if err == nil {
				t.Fatal("expected error for column count mismatch, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("expected %q in error, got: %v", tt.wantErrMsg, err)
			}
		})
	}
}

func TestForeignKey_CompositeForeignKey(t *testing.T) {
	fk := ForeignKey{
		Name:       "fk_order_items_product",
		Columns:    []string{"product_id", "variant_id"},
		RefTable:   "product_variants",
		RefColumns: []string{"product_id", "id"},
		OnDelete:   ActionCascade,
	}

	if err := fk.Validate(); err != nil {
		t.Errorf("expected valid composite FK, got error: %v", err)
	}
}

func TestForeignKey_DeferrableConstraints(t *testing.T) {
	fk := ForeignKey{
		Name:              "fk_test",
		Columns:           []string{"user_id"},
		RefTable:          "users",
		RefColumns:        []string{"id"},
		Deferrable:        true,
		InitiallyDeferred: true,
	}

	if err := fk.Validate(); err != nil {
		t.Errorf("expected valid deferrable FK, got error: %v", err)
	}

	if !fk.Deferrable {
		t.Error("expected Deferrable to be true")
	}
	if !fk.InitiallyDeferred {
		t.Error("expected InitiallyDeferred to be true")
	}
}

func TestReferentialAction_String(t *testing.T) {
	tests := []struct {
		action   ReferentialAction
		expected string
	}{
		{ActionNoAction, "NO ACTION"},
		{ActionRestrict, "RESTRICT"},
		{ActionCascade, "CASCADE"},
		{ActionSetNull, "SET NULL"},
		{ActionSetDefault, "SET DEFAULT"},
		{ReferentialAction(999), "UNKNOWN"}, // unknown action
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, got)
			}
		})
	}
}

func TestIndex_Valid(t *testing.T) {
	idx := Index{
		Name: "idx_users_email",
		Columns: []IndexColumn{
			{Name: "email", Order: SortAsc},
		},
		Unique: true,
		Type:   IndexBTree,
	}

	if idx.Name != "idx_users_email" {
		t.Errorf("expected name 'idx_users_email', got: %s", idx.Name)
	}
	if !idx.Unique {
		t.Error("expected Unique to be true")
	}
	if idx.Type != IndexBTree {
		t.Errorf("expected IndexBTree, got: %v", idx.Type)
	}
}

func TestIndex_CompositeIndex(t *testing.T) {
	idx := Index{
		Name: "idx_users_name_email",
		Columns: []IndexColumn{
			{Name: "last_name", Order: SortAsc},
			{Name: "first_name", Order: SortAsc},
		},
		Type: IndexBTree,
	}

	if len(idx.Columns) != 2 {
		t.Errorf("expected 2 columns, got: %d", len(idx.Columns))
	}
}

func TestIndex_PartialIndex(t *testing.T) {
	idx := Index{
		Name: "idx_active_users",
		Columns: []IndexColumn{
			{Name: "email", Order: SortAsc},
		},
		Where: "deleted_at IS NULL",
		Type:  IndexBTree,
	}

	if idx.Where != "deleted_at IS NULL" {
		t.Errorf("expected WHERE clause, got: %s", idx.Where)
	}
}

func TestIndex_WithOpclass(t *testing.T) {
	idx := Index{
		Name: "idx_users_email_pattern",
		Columns: []IndexColumn{
			{
				Name:    "email",
				Opclass: "text_pattern_ops",
				Order:   SortAsc,
			},
		},
		Type: IndexBTree,
	}

	if idx.Columns[0].Opclass != "text_pattern_ops" {
		t.Errorf("expected Opclass 'text_pattern_ops', got: %s", idx.Columns[0].Opclass)
	}
}

func TestIndex_WithNullsPosition(t *testing.T) {
	idx := Index{
		Name: "idx_users_last_login",
		Columns: []IndexColumn{
			{
				Name:     "last_login_at",
				Order:    SortDesc,
				NullsPos: NullsLast,
			},
		},
		Type: IndexBTree,
	}

	if idx.Columns[0].NullsPos != NullsLast {
		t.Errorf("expected NullsLast, got: %v", idx.Columns[0].NullsPos)
	}
}

func TestIndexType_String(t *testing.T) {
	tests := []struct {
		indexType IndexType
		expected  string
	}{
		{IndexBTree, "btree"},
		{IndexHash, "hash"},
		{IndexGIN, "gin"},
		{IndexGiST, "gist"},
		{IndexSPGiST, "spgist"},
		{IndexBRIN, "brin"},
		{IndexType(999), "unknown"}, // unknown type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.indexType.String(); got != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, got)
			}
		})
	}
}

func TestConstraint_CheckConstraint(t *testing.T) {
	constraint := Constraint{
		Name:       "check_age_positive",
		Type:       ConstraintCheck,
		Expression: "age >= 0",
	}

	if constraint.Type != ConstraintCheck {
		t.Errorf("expected ConstraintCheck, got: %v", constraint.Type)
	}
	if constraint.Expression != "age >= 0" {
		t.Errorf("expected expression 'age >= 0', got: %s", constraint.Expression)
	}
}

func TestConstraint_UniqueConstraint(t *testing.T) {
	constraint := Constraint{
		Name:    "unique_email",
		Type:    ConstraintUnique,
		Columns: []string{"email"},
	}

	if constraint.Type != ConstraintUnique {
		t.Errorf("expected ConstraintUnique, got: %v", constraint.Type)
	}
	if len(constraint.Columns) != 1 || constraint.Columns[0] != "email" {
		t.Errorf("expected columns [email], got: %v", constraint.Columns)
	}
}

func TestConstraint_CompositeUniqueConstraint(t *testing.T) {
	constraint := Constraint{
		Name:    "unique_user_org",
		Type:    ConstraintUnique,
		Columns: []string{"user_id", "organization_id"},
	}

	if len(constraint.Columns) != 2 {
		t.Errorf("expected 2 columns, got: %d", len(constraint.Columns))
	}
}

func TestConstraint_ExcludeConstraint(t *testing.T) {
	constraint := Constraint{
		Name:       "exclude_overlapping_periods",
		Type:       ConstraintExclude,
		Expression: "USING gist (room_id WITH =, period WITH &&)",
	}

	if constraint.Type != ConstraintExclude {
		t.Errorf("expected ConstraintExclude, got: %v", constraint.Type)
	}
}

func TestConstraintType_String(t *testing.T) {
	tests := []struct {
		constraintType ConstraintType
		expected       string
	}{
		{ConstraintCheck, "CHECK"},
		{ConstraintUnique, "UNIQUE"},
		{ConstraintExclude, "EXCLUDE"},
		{ConstraintType(999), "UNKNOWN"}, // unknown type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.constraintType.String(); got != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, got)
			}
		})
	}
}
