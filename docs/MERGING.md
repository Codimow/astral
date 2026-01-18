# Merging in Astral

Astral provides powerful branch merging capabilities with automatic conflict detection and resolution.

## Quick Start

### Basic Merge
```bash
asl merge feature-branch
```

This will merge `feature-branch` into your current branch.

## Merge Types

### Fast-Forward Merge
When your current branch hasn't diverged from the target branch, Astral performs a fast-forward merge (just moves the branch pointer forward).

```bash
# On main branch
asl branch feature
asl switch feature
# ... make changes ...
asl save -m "Add feature"

# Back to main (unchanged)
asl switch main
asl merge feature  # Fast-forward!
```

### Three-Way Merge
When branches have diverged, Astral creates a merge commit with two parent commits.

```bash
# Both branches have different commits
asl switch main
asl merge feature  # Creates merge commit
```

## Handling Conflicts

### When Conflicts Occur
```bash
$ asl merge feature-branch
✗ Merge conflict detected

Conflicted files:
  ✗ src/main.go

To resolve:
  1. Fix conflicts in each file
  2. Run: asl resolve <file>
  3. Run: asl merge --continue

Or abort with: asl merge --abort
```

### Resolving Conflicts

#### Option 1: Manual Resolution
```bash
# Edit the file and fix conflicts
vim conflicted-file.txt

# Remove conflict markers: <<<<<<< ======= >>>>>>>
# Keep the correct code

# Mark as resolved
asl resolve conflicted-file.txt

# Complete merge
asl merge --continue
```

#### Option 2: Use a Strategy
```bash
# Keep our changes for all conflicts
asl resolve --ours

# Keep their changes for all conflicts
asl resolve --theirs

# Then continue
asl merge --continue
```

### Aborting a Merge
```bash
# Cancel the merge and restore previous state
asl merge --abort
```

## Conflict Markers

Astral uses standard conflict markers:

```
our changes here
```

**How to resolve:**
1. Decide which version to keep (or combine both)
2. Remove the conflict markers (`<<<<<<<`, `|||||||`, `=======`, `>>>>>>>`)
3. Save the file
4. Run `asl resolve <file>`

## Advanced Options

### Force Merge Commit
Create a merge commit even when fast-forward is possible:
```bash
asl merge --no-ff feature
```

This is useful for maintaining a clear history of when features were merged.

### Only Fast-Forward
Fail if a three-way merge would be required:
```bash
asl merge --ff-only feature
# Error: cannot fast-forward
```

Useful in CI/CD where you want to ensure linear history.

## Checking Merge Status

```bash
$ asl status

Merging branch 'feature' into current branch
  (use "asl merge --abort" to cancel merge)

Unresolved conflicts:
  ✗ file1.txt (content)
  ✗ file2.go (content)

Auto-merged files:
  ● file3.txt

Next steps:
  1. Resolve conflicts in each file
  2. Mark as resolved: asl resolve <file>
  3. Complete merge: asl merge --continue
```

## Examples

### Example 1: Clean Merge
```bash
# Create and work on feature branch
asl branch new-feature
asl switch new-feature
echo "new code" > feature.txt
asl save -m "Add new feature"

# Merge back to main
asl switch main
asl merge new-feature
# ✓ Merge completed successfully
```

### Example 2: Merge with Conflicts
```bash
# Both branches modified same file
asl switch main
asl merge conflicting-branch

# ✗ Merge conflict detected
# Conflicted files:  ✗ shared.txt

# Resolve the conflict
vim shared.txt  # Fix conflicts manually
asl resolve shared.txt
asl merge --continue

# ✓ Merge completed
```

### Example 3: Abort Merge
```bash
# Start merge
asl merge feature

# Conflicts detected, but changed your mind
asl merge --abort

# ✓ Merge aborted
# Back to state before merge
```

## Best Practices

1. **Always check status before merging**
   ```bash
   asl status  # Make sure working tree is clean
   ```

2. **Create a backup branch before complex merges**
   ```bash
   asl branch backup-before-merge
   asl merge complex-feature
   ```

3. **Resolve conflicts in small chunks**
   - Resolve one file at a time
   - Test after each resolution
   - Mark as resolved incrementally

4. **Communicate with your team**
   - Coordinate merges of large features
   - Review conflicts together when unsure

## Troubleshooting

### "Branch not found"
Make sure the branch exists:
```bash
asl branch  # List all branches
```

### "Merge already in progress"
You have an unfinished merge:
```bash
asl status  # Check merge status
asl merge --continue  # Or --abort
```

### "Working directory has uncommitted changes"
Commit or stash your changes before merging:
```bash
asl save -m "WIP"  # Commit changes
# Then merge
```

## Related Commands

- `asl branch` - Create and list branches
- `asl switch` - Switch between branches
- `asl status` - View repository status
- `asl log` - View commit history
- `asl diff` - View changes

## Technical Notes

Astral uses a three-way merge algorithm:
1. Finds the lowest common ancestor (merge base)
2. Computes changes from base to each branch
3. Combines non-conflicting changes
4. Marks overlapping changes as conflicts

For more details, see `ARCHITECTURE.md`.
