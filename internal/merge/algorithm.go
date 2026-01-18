package merge

import (
	"fmt"
	"strings"

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
	Path    string
	Type    ConflictType
	Base    string
	Ours    string
	Theirs  string
	Markers string // Generated conflict markers
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

	// Both sides changed - need to merge
	return mergeContent(base, ours, theirs, path)
}

// mergeContent performs the actual content merging
func mergeContent(base, ours, theirs, path string) *MergeResult {
	result := &MergeResult{
		Conflicts: make([]Conflict, 0),
	}

	// Compute diffs
	diffOurs := diff.MyersDiff(base, ours)
	diffTheirs := diff.MyersDiff(base, theirs)

	// Get lines for processing
	baseLines := splitLines(base)
	ourLines := splitLines(ours)
	theirLines := splitLines(theirs)

	// Track which lines have been processed
	processed := make(map[int]bool)
	merged := make([]string, 0)

	// Find and handle conflicts
	conflicts := findConflicts(diffOurs, diffTheirs, baseLines, ourLines, theirLines)

	if len(conflicts) > 0 {
		// Has conflicts - generate conflict markers
		result.HasConflict = true
		result.Content = generateConflictMarkers(baseLines, ourLines, theirLines, conflicts, path)
		result.Conflicts = conflicts
	} else {
		// No conflicts - merge both  change sets
		result.Content = mergeChanges(baseLines, ourLines, theirLines)
		result.HasConflict = false
	}

	return result
}

// findConflicts identifies conflicting changes
func findConflicts(diffOurs, diffTheirs *diff.Diff, base, ours, theirs []string) []Conflict {
	conflicts := make([]Conflict, 0)

	// Build maps of changed line ranges
	ourChanges := make(map[int]bool)
	theirChanges := make(map[int]bool)

	for _, hunk := range diffOurs.Hunks {
		for i := hunk.OldStart; i < hunk.OldStart+hunk.OldCount; i++ {
			ourChanges[i] = true
		}
	}

	for _, hunk := range diffTheirs.Hunks {
		for i := hunk.OldStart; i < hunk.OldStart+hunk.OldCount; i++ {
			theirChanges[i] = true
		}
	}

	// Find overlapping changes
	for line := range ourChanges {
		if theirChanges[line] {
			// Both sides modified this line - conflict
			conflicts = append(conflicts, Conflict{
				Type: ConflictContent,
			})
			break // For now, treat entire file as one conflict region
		}
	}

	return conflicts
}

// generateConflictMarkers creates the text with conflict markers
func generateConflictMarkers(base, ours, theirs []string, conflicts []Conflict, path string) string {
	var result strings.Builder

	result.WriteString("<<<<<<< HEAD (ours)\n")
	result.WriteString(strings.Join(ours, "\n"))
	result.WriteString("\n||||||| BASE\n")
	result.WriteString(strings.Join(base, "\n"))
	result.WriteString("\n=======\n")
	result.WriteString(strings.Join(theirs, "\n"))
	result.WriteString("\n>>>>>>> theirs\n")

	return result.String()
}

// mergeChanges combines non-conflicting changes
func mergeChanges(base, ours, theirs []string) string {
	// Simple strategy: if both sides made same changes, use them
	// Otherwise use ours (will be improved with better diff merging)

	if strings.Join(ours, "\n") == strings.Join(theirs, "\n") {
		return strings.Join(ours, "\n")
	}

	// For now, if different, use ours
	// TODO: Implement proper non-conflicting merge
	return strings.Join(ours, "\n")
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

// FormatConflictMarkers creates enhanced conflict markers with context
func FormatConflictMarkers(conflict Conflict, ourBranch, theirBranch, ourCommit, theirCommit, baseCommit string) string {
	var result strings.Builder

	// Enhanced header with context
	result.WriteString(fmt.Sprintf("<<<<<<< HEAD (%s @ %s)\n", ourBranch, shortHash(ourCommit)))
	result.WriteString(conflict.Ours)
	result.WriteString("\n")

	// Show base for 3-way comparison
	if conflict.Base != "" {
		result.WriteString(fmt.Sprintf("||||||| BASE (%s)\n", shortHash(baseCommit)))
		result.WriteString(conflict.Base)
		result.WriteString("\n")
	}

	result.WriteString("=======\n")
	result.WriteString(conflict.Theirs)
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf(">>>>>>> %s (%s)\n", theirBranch, shortHash(theirCommit)))

	return result.String()
}

// shortHash returns first 7 characters of hash
func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
