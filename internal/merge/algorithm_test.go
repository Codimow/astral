package merge

import (
	"testing"
)

func TestThreeWayMerge_NoConflict_BothSidesIdentical(t *testing.T) {
	base := "line1\nline2\nline3\n"
	ours := "line1\nline2\nline3\n"
	theirs := "line1\nline2\nline3\n"

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if result.HasConflict {
		t.Error("expected no conflict for identical files")
	}

	if result.Content != ours {
		t.Errorf("expected content to match\ngot:\n%s\nwant:\n%s", result.Content, ours)
	}
}

func TestThreeWayMerge_NoConflict_OnlyOursChanged(t *testing.T) {
	base := "line1\nline2\nline3\n"
	ours := "line1\nmodified\nline3\n"
	theirs := "line1\nline2\nline3\n"

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if result.HasConflict {
		t.Error("expected no conflict when only one side changed")
	}

	if result.Content != ours {
		t.Errorf("expected ours version\ngot:\n%s\nwant:\n%s", result.Content, ours)
	}
}

func TestThreeWayMerge_NoConflict_OnlyTheirsChanged(t *testing.T) {
	base := "line1\nline2\nline3\n"
	ours := "line1\nline2\nline3\n"
	theirs := "line1\nmodified\nline3\n"

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if result.HasConflict {
		t.Error("expected no conflict when only one side changed")
	}

	if result.Content != theirs {
		t.Errorf("expected theirs version\ngot:\n%s\nwant:\n%s", result.Content, theirs)
	}
}

func TestThreeWayMerge_NoConflict_IdenticalChanges(t *testing.T) {
	base := "line1\nline2\nline3\n"
	ours := "line1\nmodified\nline3\n"
	theirs := "line1\nmodified\nline3\n"

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if result.HasConflict {
		t.Error("expected no conflict for identical changes")
	}

	expected := "line1\nmodified\nline3\n"
	if result.Content != expected {
		t.Errorf("expected merged content\ngot:\n%s\nwant:\n%s", result.Content, expected)
	}
}

func TestThreeWayMerge_Conflict_DifferentChanges(t *testing.T) {
	base := "line1\nline2\nline3\n"
	ours := "line1\nour change\nline3\n"
	theirs := "line1\ntheir change\nline3\n"

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if !result.HasConflict {
		t.Error("expected conflict for different changes to same line")
	}

	if len(result.Conflicts) == 0 {
		t.Error("expected at least one conflict")
	}

	if result.Conflicts[0].Type != ConflictContent {
		t.Errorf("expected ConflictContent, got %s", result.Conflicts[0].Type)
	}
}

func TestThreeWayMerge_BinaryFile(t *testing.T) {
	// Binary content with null bytes
	base := "binary\x00data\x00here"
	ours := "binary\x00data\x00modified"
	theirs := "binary\x00data\x00different"

	result := ThreeWayMerge(base, ours, theirs, "test.bin")

	if !result.HasConflict {
		t.Error("expected conflict for binary file with different content")
	}

	if len(result.Conflicts) == 0 {
		t.Fatal("expected binary conflict")
	}

	if result.Conflicts[0].Type != ConflictBinary {
		t.Errorf("expected ConflictBinary, got %s", result.Conflicts[0].Type)
	}
}

func TestThreeWayMerge_BinaryFile_Identical(t *testing.T) {
	// Identical binary content
	base := "binary\x00data"
	ours := "binary\x00data"
	theirs := "binary\x00data"

	result := ThreeWayMerge(base, ours, theirs, "test.bin")

	if result.HasConflict {
		t.Error("expected no conflict for identical binary files")
	}
}

func TestThreeWayMerge_EmptyFiles(t *testing.T) {
	base := ""
	ours := ""
	theirs := ""

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if result.HasConflict {
		t.Error("expected no conflict for empty files")
	}
}

func TestThreeWayMerge_EmptyBase_BothAdded(t *testing.T) {
	base := ""
	ours := "ours content\n"
	theirs := "theirs content\n"

	result := ThreeWayMerge(base, ours, theirs, "test.txt")

	if !result.HasConflict {
		t.Error("expected conflict when both sides add different content")
	}
}

func TestIsBinary_TextFile(t *testing.T) {
	content := []byte("This is regular text\nwith multiple lines\n")

	if isBinary(content) {
		t.Error("expected text file to not be detected as binary")
	}
}

func TestIsBinary_WithNullBytes(t *testing.T) {
	content := []byte("Some text\x00with null bytes")

	if !isBinary(content) {
		t.Error("expected file with null bytes to be detected as binary")
	}
}

func TestIsBinary_LargeBinaryContent(t *testing.T) {
	// Create content with many non-text bytes
	content := make([]byte, 10000)
	for i := range content {
		if i%2 == 0 {
			content[i] = 0x00 // Null byte
		} else {
			content[i] = 0xFF // High byte
		}
	}

	if !isBinary(content) {
		t.Error("expected large binary content to be detected as binary")
	}
}

func TestSplitLines_EmptyString(t *testing.T) {
	lines := splitLines("")

	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty string, got %d", len(lines))
	}
}

func TestSplitLines_SingleLine(t *testing.T) {
	lines := splitLines("single line")

	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}

	if lines[0] != "single line" {
		t.Errorf("unexpected line content: %s", lines[0])
	}
}

func TestSplitLines_MultipleLines(t *testing.T) {
	lines := splitLines("line1\nline2\nline3")

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	lines := splitLines("line1\nline2\n")

	if len(lines) != 2 {
		t.Errorf("expected 2 lines (trailing newline removed), got %d", len(lines))
	}
}

func BenchmarkThreeWayMerge_NoConflict(b *testing.B) {
	base := generateLargeFile(100)
	ours := base + "added line\n"
	theirs := "prepended line\n" + base

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ThreeWayMerge(base, ours, theirs, "bench.txt")
	}
}

func BenchmarkThreeWayMerge_WithConflicts(b *testing.B) {
	base := generateLargeFile(100)
	ours := base + "our addition\n"
	theirs := base + "their addition\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ThreeWayMerge(base, ours, theirs, "bench.txt")
	}
}

func generateLargeFile(lines int) string {
	var content string
	for i := 0; i < lines; i++ {
		content += "This is line number " + string(rune('0'+i%10)) + "\n"
	}
	return content
}
