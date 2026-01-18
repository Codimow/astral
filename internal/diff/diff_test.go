package diff

import (
	"testing"
)

func TestMyersDiff_EmptyFiles(t *testing.T) {
	diff := MyersDiff("", "")
	if len(diff.Hunks) != 0 {
		t.Errorf("expected no hunks for empty files, got %d", len(diff.Hunks))
	}
}

func TestMyersDiff_IdenticalFiles(t *testing.T) {
	text := "line1\nline2\nline3\n"
	diff := MyersDiff(text, text)
	if len(diff.Hunks) != 0 {
		t.Errorf("expected no hunks for identical files, got %d", len(diff.Hunks))
	}
}

func TestMyersDiff_SimpleAddition(t *testing.T) {
	old := "line1\nline2\n"
	new := "line1\nline2\nline3\n"

	diff := MyersDiff(old, new)

	if len(diff.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(diff.Hunks))
	}

	hunk := diff.Hunks[0]
	if hunk.NewCount != hunk.OldCount+1 {
		t.Error("expected one line added")
	}
}

func TestMyersDiff_SimpleDeletion(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline3\n"

	diff := MyersDiff(old, new)

	if len(diff.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(diff.Hunks))
	}

	hunk := diff.Hunks[0]
	if hunk.OldCount != hunk.NewCount+1 {
		t.Error("expected one line deleted")
	}
}

func TestMyersDiff_Modification(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nmodified\nline3\n"

	diff := MyersDiff(old, new)

	if len(diff.Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}
}

func TestMyersDiff_MultipleChanges(t *testing.T) {
	old := `line1
line2
line3
line4
line5`

	new := `line1
modified2
line3
line4
added5
line5`

	diff := MyersDiff(old, new)

	if len(diff.Hunks) == 0 {
		t.Fatal("expected hunks for multiple changes")
	}
}

func TestPatch_SimpleApplication(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nmodified\nline3\n"

	diff := MyersDiff(old, new)
	result, err := Patch(old, diff)

	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}

	if result != new {
		t.Errorf("patch result doesn't match expected.\nGot:\n%s\nWant:\n%s", result, new)
	}
}

func TestPatch_Addition(t *testing.T) {
	old := "line1\nline2\n"
	new := "line1\nline2\nline3\n"

	diff := MyersDiff(old, new)
	result, err := Patch(old, diff)

	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}

	if result != new {
		t.Errorf("patch result doesn't match expected")
	}
}

func TestPatch_Deletion(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline3\n"

	diff := MyersDiff(old, new)
	result, err := Patch(old, diff)

	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}

	if result != new {
		t.Errorf("patch result doesn't match expected")
	}
}

func BenchmarkMyersDiff_SmallFile(b *testing.B) {
	old := `line1
line2
line3
line4
line5`

	new := `line1
modified
line3
line4
added
line5`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MyersDiff(old, new)
	}
}

func BenchmarkMyersDiff_MediumFile(b *testing.B) {
	// Generate larger test files
	old := ""
	new := ""
	for i := 0; i < 100; i++ {
		old += "line content " + string(rune(i)) + "\n"
		if i%10 == 5 {
			new += "modified content\n"
		} else {
			new += "line content " + string(rune(i)) + "\n"
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MyersDiff(old, new)
	}
}
