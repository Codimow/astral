package diff

import (
	"strings"
)

// EditType represents the type of edit in a diff
type EditType int

const (
	EditEqual EditType = iota
	EditInsert
	EditDelete
)

// Edit represents a single edit operation
type Edit struct {
	Type EditType
	Text string
}

// Hunk represents a group of changes with context
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Edits    []Edit
}

// Diff represents the complete difference between two texts
type Diff struct {
	Hunks []Hunk
}

// MyersDiff computes the diff between two texts using Myers algorithm
func MyersDiff(oldText, newText string) *Diff {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	edits := myersAlgorithm(oldLines, newLines)
	hunks := groupIntoHunks(edits, oldLines, newLines, 3) // 3 lines of context

	return &Diff{Hunks: hunks}
}

// myersAlgorithm implements the Myers diff algorithm
func myersAlgorithm(a, b []string) []Edit {
	n := len(a)
	m := len(b)

	// Handle empty cases
	if n == 0 && m == 0 {
		return []Edit{}
	}
	if n == 0 {
		// All inserts
		edits := make([]Edit, m)
		for i := 0; i < m; i++ {
			edits[i] = Edit{EditInsert, b[i]}
		}
		return edits
	}
	if m == 0 {
		// All deletes
		edits := make([]Edit, n)
		for i := 0; i < n; i++ {
			edits[i] = Edit{EditDelete, a[i]}
		}
		return edits
	}

	max := n + m

	// V array stores furthest reaching D-path
	v := make([]int, 2*max+1)
	trace := make([]map[int]int, 0)

	// Find the shortest edit script
	for d := 0; d <= max; d++ {
		// Save current V for backtracking
		vCopy := make(map[int]int)
		for k := -d; k <= d; k += 2 {
			vCopy[k] = v[k+max]
		}
		trace = append(trace, vCopy)

		for k := -d; k <= d; k += 2 {
			var x int

			// Choose whether to go down or right
			if k == -d || (k != d && v[k-1+max] < v[k+1+max]) {
				x = v[k+1+max]
			} else {
				x = v[k-1+max] + 1
			}

			y := x - k

			// Extend diagonal as far as possible
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[k+max] = x

			// Check if we've reached the end
			if x >= n && y >= m {
				return backtrack(a, b, trace, d)
			}
		}
	}

	// Shouldn't reach here
	return []Edit{}
}

// backtrack reconstructs the edit script from the trace
func backtrack(a, b []string, trace []map[int]int, d int) []Edit {
	edits := make([]Edit, 0)
	x := len(a)
	y := len(b)

	for d > 0 {
		v := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK]
		prevY := prevX - prevK

		// Add diagonal edits (equals)
		for x > prevX && y > prevY {
			x--
			y--
			edits = append([]Edit{{EditEqual, a[x]}}, edits...)
		}

		// Add vertical or horizontal edit
		if x == prevX {
			// Insert
			y--
			edits = append([]Edit{{EditInsert, b[y]}}, edits...)
		} else {
			// Delete
			x--
			edits = append([]Edit{{EditDelete, a[x]}}, edits...)
		}

		d--
	}

	// Add remaining equal lines from start
	for x > 0 && y > 0 {
		x--
		y--
		edits = append([]Edit{{EditEqual, a[x]}}, edits...)
	}

	return edits
}

// groupIntoHunks groups edits into hunks with context
func groupIntoHunks(edits []Edit, oldLines, newLines []string, context int) []Hunk {
	if len(edits) == 0 {
		return []Hunk{}
	}

	hunks := make([]Hunk, 0)
	var currentHunk *Hunk
	oldIdx := 0
	newIdx := 0
	contextBefore := 0
	contextAfter := 0

	for i, edit := range edits {
		switch edit.Type {
		case EditEqual:
			if currentHunk == nil {
				// Not in a hunk, count context
				contextBefore++
				if contextBefore > context {
					contextBefore = context
				}
			} else {
				// In a hunk, check if we should close it
				contextAfter++

				// Look ahead to see if there are more changes
				hasMoreChanges := false
				for j := i + 1; j < len(edits) && j < i+context+1; j++ {
					if edits[j].Type != EditEqual {
						hasMoreChanges = true
						break
					}
				}

				if !hasMoreChanges && contextAfter >= context {
					// Close current hunk
					hunks = append(hunks, *currentHunk)
					currentHunk = nil
					contextBefore = 0
					contextAfter = 0
				} else {
					currentHunk.Edits = append(currentHunk.Edits, edit)
					currentHunk.OldCount++
					currentHunk.NewCount++
				}
			}
			oldIdx++
			newIdx++

		case EditDelete, EditInsert:
			if currentHunk == nil {
				// Start new hunk
				currentHunk = &Hunk{
					OldStart: oldIdx - contextBefore,
					NewStart: newIdx - contextBefore,
					Edits:    make([]Edit, 0),
				}

				// Add context before
				for j := contextBefore; j > 0; j-- {
					if oldIdx-j >= 0 {
						currentHunk.Edits = append(currentHunk.Edits, Edit{EditEqual, oldLines[oldIdx-j]})
						currentHunk.OldCount++
						currentHunk.NewCount++
					}
				}
			}

			currentHunk.Edits = append(currentHunk.Edits, edit)
			contextAfter = 0

			if edit.Type == EditDelete {
				currentHunk.OldCount++
				oldIdx++
			} else {
				currentHunk.NewCount++
				newIdx++
			}
		}
	}

	// Close last hunk if exists
	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// splitLines splits text into lines, preserving line endings
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}

	lines := strings.Split(text, "\n")

	// Remove empty last line if text ended with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// ApplyHunk applies a hunk to text
func ApplyHunk(text string, hunk Hunk) (string, error) {
	lines := splitLines(text)
	result := make([]string, 0)

	// Add lines before hunk
	result = append(result, lines[:hunk.OldStart]...)

	// Apply hunk edits
	for _, edit := range hunk.Edits {
		switch edit.Type {
		case EditEqual, EditInsert:
			result = append(result, edit.Text)
		case EditDelete:
			// Skip deleted lines
		}
	}

	// Add lines after hunk
	if hunk.OldStart+hunk.OldCount < len(lines) {
		result = append(result, lines[hunk.OldStart+hunk.OldCount:]...)
	}

	return strings.Join(result, "\n"), nil
}

// Patch applies all hunks in a diff
func Patch(text string, diff *Diff) (string, error) {
	result := text

	for _, hunk := range diff.Hunks {
		var err error
		result, err = ApplyHunk(result, hunk)
		if err != nil {
			return "", err
		}
	}

	return result, nil
}
