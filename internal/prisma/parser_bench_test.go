package prisma

import (
	"strings"
	"testing"
)

// BenchmarkParser_SimpleModel benchmarks parsing a simple model.
func BenchmarkParser_SimpleModel(b *testing.B) {
	input := `model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  name  String?

  @@map("users")
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_ComplexModel benchmarks parsing a complex model.
func BenchmarkParser_ComplexModel(b *testing.B) {
	input := `enum Role {
  USER
  ADMIN
  MODERATOR
}

model User {
  id            String   @id @default(uuid()) @db.Uuid
  email         String   @unique
  passwordHash  String   @map("password_hash")
  firstName     String   @map("first_name")
  lastName      String   @map("last_name")
  role          Role     @default(USER)
  emailVerified Boolean  @default(false) @map("email_verified")
  createdAt     DateTime @default(now()) @map("created_at")
  updatedAt     DateTime @updatedAt @map("updated_at")

  @@index([email])
  @@index([role])
  @@map("users")
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_LargeSchema benchmarks parsing a large schema (100 models).
func BenchmarkParser_LargeSchema(b *testing.B) {
	// Generate a schema with 100 models, each with 10 fields
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("model Model")
		sb.WriteString(strings.Repeat("X", i/26))
		sb.WriteString(string(rune('A' + (i % 26))))
		sb.WriteString(" {\n")
		sb.WriteString("  id    Int     @id @default(autoincrement())\n")
		for j := 0; j < 9; j++ {
			sb.WriteString("  field")
			sb.WriteString(string(rune('0' + j)))
			sb.WriteString(" String?\n")
		}
		sb.WriteString("\n  @@map(\"table_")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString("\")\n")
		sb.WriteString("}\n\n")
	}

	input := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_10kLines benchmarks parsing a ~10k line schema.
func BenchmarkParser_10kLines(b *testing.B) {
	// Generate a schema with ~10k lines (~1000 models with 10 fields each)
	var sb strings.Builder

	// Add some enums (50 lines)
	for i := 0; i < 5; i++ {
		sb.WriteString("enum Enum")
		sb.WriteString(string(rune('0' + i)))
		sb.WriteString(" {\n")
		sb.WriteString("  VALUE1\n  VALUE2\n  VALUE3\n}\n\n")
	}

	// Generate models to reach ~10k lines
	// Each model is ~10 lines (with fields and attributes)
	for i := 0; i < 950; i++ {
		sb.WriteString("model Model")
		sb.WriteString(string(rune('A' + (i % 26))))
		if i >= 26 {
			sb.WriteString(string(rune('A' + ((i / 26) % 26))))
		}
		if i >= 676 {
			sb.WriteString(string(rune('A' + ((i / 676) % 26))))
		}
		sb.WriteString(" {\n")
		sb.WriteString("  id        Int      @id @default(autoincrement())\n")
		sb.WriteString("  name      String\n")
		sb.WriteString("  email     String   @unique\n")
		sb.WriteString("  active    Boolean  @default(true)\n")
		sb.WriteString("  createdAt DateTime @default(now())\n")
		sb.WriteString("  updatedAt DateTime @updatedAt\n")
		sb.WriteString("\n  @@index([email])\n")
		sb.WriteString("  @@map(\"table_")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString("\")\n}\n\n")
	}

	input := sb.String()
	lines := strings.Count(input, "\n")
	b.Logf("Generated schema with %d lines", lines)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLexer_Tokenization benchmarks just the lexer tokenization.
func BenchmarkLexer_Tokenization(b *testing.B) {
	input := `model User {
  id            String   @id @default(uuid()) @db.Uuid
  email         String   @unique
  passwordHash  String   @map("password_hash")
  createdAt     DateTime @default(now())

  @@map("users")
  @@index([email])
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		tokens := lexer.AllTokens()
		if len(tokens) == 0 {
			b.Fatal("no tokens")
		}
	}
}

// BenchmarkConverter_Conversion benchmarks AST to Internal Schema conversion.
func BenchmarkConverter_Conversion(b *testing.B) {
	input := `enum Role {
  USER
  ADMIN
}

model User {
  id        String   @id @default(uuid()) @db.Uuid
  email     String   @unique
  name      String?
  role      Role     @default(USER)
  createdAt DateTime @default(now())

  @@map("users")
  @@index([email])
}`

	// Parse once to get AST
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter := NewConverter()
		_, err := converter.Convert(ast)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoundTrip benchmarks full round-trip: Parse → Convert → Write.
func BenchmarkRoundTrip(b *testing.B) {
	input := `enum Role {
  USER
  ADMIN
}

model User {
  id        String   @id @default(uuid()) @db.Uuid
  email     String   @unique
  name      String?
  role      Role     @default(USER)
  createdAt DateTime @default(now())

  @@map("users")
  @@index([email])
}`

	naming := NamingConvention{
		Fields: "snake_case",
		Models: "snake_case",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Parse
		result, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}

		// Write back
		writer := NewWriter(naming)
		_, err = writer.Write(result.Schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}
