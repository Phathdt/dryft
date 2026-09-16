package prisma

import (
	"fmt"
	"strings"
)

// Formatter handles Prisma schema formatting with consistent spacing and indentation.
type Formatter struct {
	indent string
}

// NewFormatter creates a new Formatter with 2-space indentation (Prisma standard).
func NewFormatter() *Formatter {
	return &Formatter{
		indent: "  ",
	}
}

// FormatBlock formats a block declaration (generator, datasource, model, enum).
func (f *Formatter) FormatBlock(blockType, name string, content []string) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("%s %s {", blockType, name))

	for _, line := range content {
		if line == "" {
			// Preserve blank separator lines without trailing indent whitespace.
			lines = append(lines, "")
			continue
		}
		lines = append(lines, f.indent+line)
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

// FormatField formats a Prisma model field with proper alignment.
// Example: "id        String   @id @db.Uuid"
func (f *Formatter) FormatField(name, fieldType string, attributes []string) string {
	parts := []string{name, fieldType}
	if len(attributes) > 0 {
		parts = append(parts, strings.Join(attributes, " "))
	}
	return strings.Join(parts, " ")
}

// AlignFields aligns field names and types with consistent spacing.
// It calculates the maximum width for names and types to create a clean table-like layout.
func (f *Formatter) AlignFields(fields []FieldLine) []string {
	if len(fields) == 0 {
		return nil
	}

	// Calculate max widths
	maxNameWidth := 0
	maxTypeWidth := 0
	for _, field := range fields {
		if len(field.Name) > maxNameWidth {
			maxNameWidth = len(field.Name)
		}
		if len(field.Type) > maxTypeWidth {
			maxTypeWidth = len(field.Type)
		}
	}

	// Format with alignment
	var result []string
	for _, field := range fields {
		namePadded := field.Name + strings.Repeat(" ", maxNameWidth-len(field.Name))

		if len(field.Attributes) > 0 {
			typePadded := field.Type + strings.Repeat(" ", maxTypeWidth-len(field.Type))
			line := namePadded + " " + typePadded + " " + strings.Join(field.Attributes, " ")
			result = append(result, line)
		} else {
			line := namePadded + " " + field.Type
			result = append(result, line)
		}
	}

	return result
}

// FieldLine represents a single field in a Prisma model for alignment purposes.
type FieldLine struct {
	Name       string
	Type       string
	Attributes []string
}

// FormatAttributes formats model-level attributes (@@map, @@index, @@unique).
func (f *Formatter) FormatAttributes(attributes []string) []string {
	if len(attributes) == 0 {
		return nil
	}

	result := append([]string(nil), attributes...)
	return result
}

// FormatEnumValue formats an enum value with proper spacing.
func (f *Formatter) FormatEnumValue(value string) string {
	return value
}

// JoinBlocks joins multiple blocks with double newlines for readability.
func (f *Formatter) JoinBlocks(blocks []string) string {
	return strings.Join(blocks, "\n\n")
}
