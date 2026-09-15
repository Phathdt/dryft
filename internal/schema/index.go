package schema

// Index represents a database index that speeds up data retrieval operations.
type Index struct {
	// Name is the index name. Must be unique within the database.
	Name string

	// Columns is the list of indexed columns with their configuration.
	Columns []IndexColumn

	// Unique indicates whether the index enforces uniqueness.
	Unique bool

	// Type specifies the index access method (B-tree, Hash, GIN, etc.).
	Type IndexType

	// Where is an optional partial index predicate (e.g., "status = 'active'").
	Where string

	// Comment is an optional description of the index's purpose.
	Comment string
}

// IndexColumn represents a single column within an index.
type IndexColumn struct {
	// Name is the column name.
	Name string

	// Opclass is the operator class (e.g., "text_pattern_ops").
	Opclass string

	// Order specifies the sort direction (SortAsc or SortDesc).
	Order SortOrder

	// NullsPos specifies where NULL values appear in the sort order.
	NullsPos NullsPosition
}

// IndexType represents the index access method.
type IndexType int

const (
	// IndexBTree is the default balanced tree index.
	IndexBTree IndexType = iota

	// IndexHash is optimized for equality comparisons.
	IndexHash

	// IndexGIN (Generalized Inverted Index) for composite values like arrays and JSONB.
	IndexGIN

	// IndexGiST (Generalized Search Tree) for geometric data and full-text search.
	IndexGiST

	// IndexSPGiST (Space-Partitioned GiST) for non-balanced data structures.
	IndexSPGiST

	// IndexBRIN (Block Range Index) for large naturally ordered tables.
	IndexBRIN
)

// String returns the PostgreSQL name for the index type.
func (i IndexType) String() string {
	switch i {
	case IndexBTree:
		return "btree"
	case IndexHash:
		return "hash"
	case IndexGIN:
		return "gin"
	case IndexGiST:
		return "gist"
	case IndexSPGiST:
		return "spgist"
	case IndexBRIN:
		return "brin"
	default:
		return "unknown"
	}
}
