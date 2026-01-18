package merge

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/codimo/astral/internal/core"
	"github.com/codimo/astral/internal/diff"
)

// ConflictType represents the type of merge conflict
type ConflictType string

const (
	ConflictContent      ConflictType = "content"
	ConflictDeleteModify ConflictType = "delete-modify"
	ConflictAddAdd       ConflictType = "add-add"
	ConflictBinary       ConflictType = "binary"
)

// Conflict represents a merge conflict in a file
type Conflict struct {
	Path      string
	Type      ConflictType
	Base      string
	Ours      string
	Theirs    string
	LineStart int
	LineEnd   int
}

// MergeResult represents the result of a three-way merge
type MergeResult struct {
	Content     string
	Conflicts   []Conflict
	HasConflict bool
}

// ThreeWayMerge performs a three-way merge on file content
func ThreeWayMerge(base, ours, theirs, path string) *MergeResult {
	result := &MergeResult{
		Conflicts: make([]Conflict, 0),
	}

	// Check for binary files
	if isBinary([]byte(base)) || isBinary([]byte(ours)) || isBinary([]byte(theirs)) {
		if ours != theirs {
			result.HasConflict = true
			result.Conflicts = append(result.Conflicts, Conflict{
				Path:   path,
				Type:   ConflictBinary,
				Base:   base,
				Ours:   ours,
				Theirs: theirs,
			})
			result.Content = generateBinaryConflictMarkers(path)
		} else {
			result.Content = ours
		}
		return result
	}

	// Check if files are identical
	if ours == theirs {
		result.Content = ours
		return result
	}

	// If one side is unchanged, use the other
	if ours == base {
		result.Content = theirs
		return result
	}
	if theirs == base {
		result.Content = ours
		return result
	}

	// Both sides changed - perform three-way merge
	return mergeContent(base, ours, theirs, path)
}

// mergeContent performs the actual content merging
func mergeContent(base, ours, theirs, path string) *MergeResult {
	result := &MergeResult{
		Conflicts: make([]Conflict, 0),
	}

	// Compute diffs from base
	diffOurs := diff.MyersDiff(base, ours)
	diffTheirs := diff.MyersDiff(base, theirs)

	// Build change maps
	ourChanges := buildChangeMap(diffOurs)
	theirChanges := buildChangeMap(diffTheirs)

	// Merge line by line
	baseLines := splitLines(base)
	ourLines := splitLines(ours)
	theirLines := splitLines(theirs)

	merged, conflicts := mergeLinesWithConflicts(
		baseLines, ourLines, theirLines,
		ourChanges, theirChanges, path,
	)

	if len(conflicts) > 0 {
		result.HasConflict = true
		result.Conflicts = conflicts
		result.Content = generateConflictMarkers(merged, conflicts, path)
	} else {
		result.Content = strings.Join(merged, "\n")
		if len(merged) > 0 && !strings.HasSuffix(result.Content, "\n") {
			result.Content += "\n"
		}
	}

	return result
}

// ChangeInfo represents a change at a specific line
type ChangeInfo struct {
	Type      diff.EditType
	Content   string
	BaseStart int
	BaseEnd   int
}

// buildChangeMap builds a map of line changes from a diff
func buildChangeMap(d *diff.Diff) map[int][]ChangeInfo {
	changes := make(map[int][]ChangeInfo)

	for _, hunk := range d.Hunks {
		baseIdx := hunk.OldStart
		for _, edit := range hunk.Edits {
			switch edit.Type {
			case diff.EditDelete, diff.EditEqual:
				info := ChangeInfo{
					Type:      edit.Type,
					Content:   edit.Text,
					BaseStart: baseIdx,
					BaseEnd:   baseIdx + 1,
				}
				changes[baseIdx] = append(changes[baseIdx], info)
				baseIdx++
			case diff.EditInsert:
				// Insert doesn't advance base index
				info := ChangeInfo{
					Type:      edit.Type,
					Content:   edit.Text,
					BaseStart: baseIdx,
					BaseEnd:   baseIdx,
				}
				changes[baseIdx] = append(changes[baseIdx], info)
			}
		}
	}

	return changes
}

// mergeLinesWithConflicts merges lines and detects conflicts
func mergeLinesWithConflicts(
	base, ours, theirs []string,
	ourChanges, theirChanges map[int][]ChangeInfo,
	path string,
) ([]string, []Conflict) {
	merged := make([]string, 0)
	conflicts := make([]Conflict, 0)

	// Simple strategy: go through base line by line
	for i := 0; i < len(base); i++ {
		ourEdits := ourChanges[i]
		theirEdits := theirChanges[i]

		// No changes on either side
		if len(ourEdits) == 0 && len(theirEdits) == 0 {
			merged = append(merged, base[i])
			continue
		}

		// Only one side changed
		if len(ourEdits) == 0 {
			// Only theirs changed
			for _, edit := range theirEdits {
				if edit.Type == diff.EditInsert {
					merged = append(merged, edit.Content)
				} else if edit.Type == diff.EditEqual {
					merged = append(merged, base[i])
				}
				// DeleteEdit skips the line
			}
			continue
		}

		if len(theirEdits) == 0 {
			// Only ours changed
			for _, edit := range ourEdits {
				if edit.Type == diff.EditInsert {
					merged = append(merged, edit.Content)
				} else if edit.Type == diff.EditEqual {
					merged = append(merged, base[i])
				}
			}
			continue
		}

		// Both sides changed - check if identical
		if editsIdentical(ourEdits, theirEdits) {
			// Same changes on both sides - use either
			for _, edit := range ourEdits {
				if edit.Type != diff.EditDelete {
					merged = append(merged, edit.Content)
				}
			}
			continue
		}

		// Different changes - conflict!
		conflict := Conflict{
			Path:      path,
			Type:      ConflictContent,
			Base:      base[i],
			Ours:      formatEdits(ourEdits),
			Theirs:    formatEdits(theirEdits),
			LineStart: i,
			LineEnd:   i + 1,
		}
		conflicts = append(conflicts, conflict)

		// Add conflict marker placeholder
		merged = append(merged, fmt.Sprintf("<<<CONFLICT_%d>>>", len(conflicts)-1))
	}

	return merged, conflicts
}

// editsIdentical checks if two edit lists are identical
func editsIdentical(a, b []ChangeInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type || a[i].Content != b[i].Content {
			return false
		}
	}
	return true
}

// formatEdits formats edit list to string
func formatEdits(edits []ChangeInfo) string {
	var result strings.Builder
	for _, edit := range edits {
		if edit.Type != diff.EditDelete {
			result.WriteString(edit.Content)
			result.WriteString("\n")
		}
	}
	s := result.String()
	return strings.TrimSuffix(s, "\n")
}

// generateConflictMarkers creates text with conflict markers
func generateConflictMarkers(merged []string, conflicts []Conflict, path string) string {
	var result strings.Builder

	for _, line := range merged {
		if strings.HasPrefix(line, "<<<CONFLICT_") {
			// Extract conflict index
			var idx int
			fmt.Sscanf(line, "<<<CONFLICT_%d>>>", &idx)

			if idx < len(conflicts) {
				c := conflicts[idx]
				result.WriteString("<<<<<<< HEAD (ours)\n")
				result.WriteString(c.Ours)
				result.WriteString("\n||||||| BASE\n")
				result.WriteString(c.Base)
				result.WriteString("\n=======\n")
				result.WriteString(c.Theirs)
				result.WriteString("\n>>>>>>> theirs\n")
			}
		} else {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// generateBinaryConflictMarkers creates markers for binary conflicts
func generateBinaryConflictMarkers(path string) string {
	return fmt.Sprintf(`<<<<<<< HEAD (ours)
Binary file %s (ours)
=======
Binary file %s (theirs)
>>>>>>> theirs

Cannot auto-merge binary files.
Use 'asl resolve --ours %s' or 'asl resolve --theirs %s'
`, path, path, path, path)
}

// splitLines splits content into lines
func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// isBinary detects if content is binary
func isBinary(content []byte) bool {
	// Check for null bytes (common in binary files)
	if bytes.Contains(content, []byte{0}) {
		return true
	}

	// Sample first 8KB to check
	sampleSize := 8192
	if len(content) < sampleSize {
		sampleSize = len(content)
	}

	// Count non-text bytes
	nonText := 0
	for i := 0; i < sampleSize; i++ {
		b := content[i]
		if b < 7 || b == 11 || (b >= 14 && b < 32 && b != 27) {
			nonText++
		}
	}

	// If more than 30% non-text, consider binary
	return nonText > sampleSize*30/100
}

// FormatConflictMarkers creates enhanced conflict markers with context
func FormatConflictMarkers(conflict Conflict, ourBranch, theirBranch, ourCommit, theirCommit, baseCommit core.Hash) string {
	var result strings.Builder

	// Enhanced header with context
	result.WriteString(fmt.Sprintf("<<<<<<< HEAD (%s @ %s)\n", ourBranch, ourCommit.Short()))
	result.WriteString(conflict.Ours)
	result.WriteString("\n")

	// Show base for 3-way comparison
	if conflict.Base != "" {
		result.WriteString(fmt.Sprintf("||||||| BASE (%s)\n", baseCommit.Short()))
		result.WriteString(conflict.Base)
		result.WriteString("\n")
	}

	result.WriteString("=======\n")
	result.WriteString(conflict.Theirs)
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf(">>>>>>> %s (%s)\n", theirBranch, theirCommit.Short()))

	return result.String()
}
