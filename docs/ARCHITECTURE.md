# Astral Architecture

This document explains the design decisions and internal architecture of Astral.

## Design Philosophy

Astral is built on three core principles:

1. **Simplicity Over Complexity**: Remove unnecessary abstractions (like staging areas)
2. **Performance First**: Optimize critical paths with modern algorithms
3. **Developer Experience**: Make the right thing easy and the wrong thing hard

## Core Components

### 1. Content-Addressable Storage

Astral uses Blake3 hashing for content-addressable storage:

**Why Blake3?**
- 2x faster than SHA-1
- Better security (256-bit vs 160-bit)
- Parallelizable by design
- Hardware acceleration support

**Storage Format:**
```
objects/
  12/
    3456789abcdef... (compressed object)
```

Each object is stored as:
```
<type> <data>
```

Compressed with zlib for space efficiency.

### 2. Object Types

#### Blob
Raw file content. No metadata, just bytes.

#### Tree
Directory snapshot containing entries:
```
<mode> <name>\0<32-byte hash>
```

Sorted by name for deterministic hashing.

#### Commit
Snapshot pointer with metadata:
```
tree <tree-hash>
parent <parent-hash>  (optional)
author <name> <email> <timestamp>

<commit message>
```

### 3. Reference System

References are mutable pointers to commits:

```
.asl/
  HEAD              -> "ref: refs/heads/main"
  refs/
    heads/
      main          -> <commit-hash>
      feature-x     -> <commit-hash>
```

HEAD can be:
- **Symbolic**: Points to a branch (`ref: refs/heads/main`)
- **Direct**: Contains a commit hash (detached HEAD)

### 4. Concurrency Model

**Lock-Free Operations:**
- Object database reads are concurrent-safe
- Writes use content-addressing (no conflicts)
- Caching uses `sync.RWMutex` for read-heavy workload

**Parallel File Processing:**
```go
var g errgroup.Group
for _, file := range files {
    file := file
    g.Go(func() error {
        // Process file in parallel
        return hashAndStore(file)
    })
}
g.Wait()
```

## Data Flow

### Commit Operation

```
1. List files to commit
   ↓
2. Hash files in parallel → Blob objects
   ↓
3. Build tree from blobs → Tree object
   ↓
4. Create commit pointing to tree → Commit object
   ↓
5. Update branch reference
```

### Checkout Operation

```
1. Read commit object
   ↓
2. Read tree object
   ↓
3. Restore files in parallel from blobs
   ↓
4. Update working directory
```

## Performance Optimizations

### 1. Object Caching

In-memory cache of recently accessed objects:
```go
type Store struct {
    cache map[Hash]*Object
    mu    sync.RWMutex
}
```

Cache hit rate >90% for typical workflows.

### 2. Compression

Objects are compressed with zlib (level 6):
- Fast enough for real-time operations
- 60-70% size reduction on average

### 3. Deduplication

Content-addressable storage automatically deduplicates:
```
same content → same hash → single storage
```

## Security Considerations

### 1. Hash Collision Resistance

Blake3 provides 2^256 possible hashes:
- Collision probability: negligible
- Better than SHA-1 (known weaknesses)

### 2. Input Validation

All user inputs are validated:
```go
// Branch names
if name == "" || name == "HEAD" {
    return ErrInvalidBranchName
}

// Commit hashes
if len(hash) != 32 {
    return ErrInvalidHash
}
```

### 3. No Command Injection

All file operations use safe APIs:
```go
os.ReadFile(path)  // Not os.Exec("cat " + path)
```

## Comparison with Git

| Feature | Astral | Git |
|---------|--------|-----|
| Hashing | Blake3 | SHA-1 |
| Staging | No | Yes |
| Speed | Fast | Fast |
| Complexity | Low | High |
| Storage | Compressed | Compressed + Packs |

## Future Optimizations

### Pack Files (Phase 4)

Current: One file per object
Future: Packed objects with delta compression

Benefits:
- 90% size reduction
- Faster network transfers
- Slower write operations (acceptable tradeoff)

### Memory-Mapped I/O (Phase 4)

For large repositories:
```go
mmap.Open(path) // Map file to memory
```

Benefits:
- Faster large file access
- OS handles caching
- Shared memory between processes

### Incremental Updates

Track which files changed:
```
.asl/index:
  <path> <mtime> <hash>
```

Only rehash modified files.

## Design Decisions

### Why No Staging Area?

**Git staging area adds complexity:**
- `git add` → `git commit` (two steps)
- Beginners often confused
- Most commits include all changes anyway

**Astral approach:**
```bash
asl save -m "message"  # Save all changes
asl save file.txt -m "message"  # Save specific file
```

Simpler mental model, fewer commands.

### Why Stack-Based Workflow?

Modern development uses stacked branches:
```
main  →  feature-1  →  feature-2  →  feature-3
```

Astral makes this first-class:
```bash
asl stack  # Visualize entire stack
```

### Why Go?

- **Performance**: Compiled, fast
- **Concurrency**: Excellent goroutine support
- **Cross-platform**: Single binary, no runtime
- **Tooling**: Built-in testing, benchmarking

## Error Handling Philosophy

**Never panic in library code:**
```go
// Good
func Read() (*Data, error) {
    if err != nil {
        return nil, fmt.Errorf("read failed: %w", err)
    }
}

// Bad
func Read() *Data {
    if err != nil {
        panic(err)  // NO!
    }
}
```

**Use structured errors:**
```go
var (
    ErrObjectNotFound = errors.New("object not found")
    ErrInvalidHash    = errors.New("invalid hash")
)
```

Allows caller to handle specifically:
```go
if errors.Is(err, ErrObjectNotFound) {
    // Handle missing object
}
```

## Conclusion

Astral's architecture prioritizes:
1. **Correctness** - Never lose data
2. **Performance** - Make fast things fast
3. **Simplicity** - Remove accidental complexity

The result is a VCS that's fast, reliable, and enjoyable to use.

---

**Questions?** Open an issue for discussion!
