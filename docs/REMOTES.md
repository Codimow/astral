# Working with Remotes

Astral supports distributed version control through remote repositories. This document explains how to work with remotes.

## Remote Management

### Adding a Remote

Add a remote repository to your local repository:

```bash
asl remote add <name> <url>
```

Example:
```bash
asl remote add origin https://example.com/repo.git
```

### Listing Remotes

List all configured remotes:

```bash
asl remote list
```

Output format: `<name>    <url>`

### Removing a Remote

Remove a remote from your repository:

```bash
asl remote remove <name>
```

Example:
```bash
asl remote remove origin
```

## Cloning

Clone a remote repository into a new directory:

```bash
asl clone <url> [directory]
```

Examples:
```bash
# Clone into default directory
asl clone https://example.com/repo.git

# Clone into specific directory
asl clone https://example.com/repo.git myproject
```

**What happens during clone:**
1. Creates a new directory
2. Initializes a new Astral repository
3. Adds the remote as 'origin'
4. Fetches all objects and refs from remote
5. Creates remote-tracking branches (refs/remotes/origin/*)
6. Checks out the default branch

## Fetching

Download objects and refs from a remote repository without merging:

```bash
asl fetch [remote]
```

Examples:
```bash
# Fetch from default remote (origin)
asl fetch

# Fetch from specific remote
asl fetch origin
```

**What happens during fetch:**
1. Connects to remote
2. Lists all remote refs
3. Performs smart graph traversal to download only missing objects
4. Updates remote-tracking branches (refs/remotes/<remote>/*)

## Pushing

Upload local commits to a remote repository:

```bash
asl push [remote] [branch]
```

Examples:
```bash
# Push current branch to origin
asl push

# Push specific branch to origin
asl push origin main

# Force push (dangerous!)
asl push --force

# Push all branches
asl push --all
```

**What happens during push:**
1. Resolves remote and branch
2. Checks for fast-forward (unless --force)
3. Calculates which objects need to be pushed
4. Uploads objects in batched, gzip-compressed request
5. Updates remote ref

**Fast-forward checks:**
- By default, pushes are rejected if they're not fast-forward
- Use `--force` to override (be careful!)
- Fast-forward means your local branch contains all commits from remote

## Pulling

Fetch from remote and merge into current branch:

```bash
asl pull [remote] [branch]
```

Examples:
```bash
# Pull from default remote into current branch
asl pull

# Pull specific branch from origin
asl pull origin main
```

**What happens during pull:**
1. Performs fetch (see above)
2. Merges remote-tracking branch into current branch
3. May result in fast-forward or three-way merge
4. May result in conflicts that need resolution

**Conflicts:**
If merge conflicts occur:
```bash
# Fix conflicts in your editor
# Then continue the merge
asl merge --continue

# Or abort the merge
asl merge --abort
```

## Authentication

Astral supports multiple authentication methods:

### None (Public Repositories)
```bash
# No authentication needed for public repos
asl clone https://example.com/public-repo.git
```

### Basic Authentication
Set environment variables:
```bash
export ASL_AUTH_USERNAME="your-username"
export ASL_AUTH_PASSWORD="your-password"
```

### Token Authentication (Recommended)
Set environment variable:
```bash
export ASL_AUTH_TOKEN="your-token"
```

Token auth is preferred for security over basic auth.

## Protocol Details

Astral uses HTTP/HTTPS for remote operations with the following endpoints:

- `GET /info/refs` - List all refs and their hashes
- `GET /objects/{hash}` - Fetch a single object
- `POST /objects/` - Upload multiple objects (batched, gzip-compressed)
- `GET /refs/heads/{branch}` - Get a specific ref
- `POST /refs/heads/{branch}` - Update a specific ref

### Object Transfer

**Fetch Strategy:**
- Client-side graph walking
- Downloads objects breadth-first from tips
- Stops when encountering existing objects
- Assumes if we have an object, we have its history

**Push Strategy:**
- Server-side validation of fast-forward
- Batched object upload with gzip compression
- Graph traversal to find all needed objects
- Stops at objects known to exist on remote

## Remote-Tracking Branches

Remote branches are stored as:
```
refs/remotes/<remote>/<branch>
```

Example:
- Remote: `origin`
- Branch: `main`
- Tracking ref: `refs/remotes/origin/main`

These refs are updated during fetch and pull operations and can be merged into local branches.

## Configuration

Remote configurations are stored in `.asl/config/config` file:

```ini
[remote "origin"]
    url = https://example.com/repo.git
    fetch = +refs/heads/*:refs/remotes/origin/*
```

## Best Practices

1. **Fetch regularly** to stay up-to-date with remote changes
2. **Use meaningful remote names** (origin for main repo, upstream for fork source)
3. **Avoid force-pushing** to shared branches
4. **Use token auth** instead of password auth for better security
5. **Check status** before pushing to avoid conflicts

## Common Workflows

### Contributing to a Project
```bash
# Clone the repository
asl clone https://example.com/project.git
cd project

# Create a feature branch
asl branch feature-x
asl switch feature-x

# Make changes and commit
asl save -m "Add feature X"

# Push your branch
asl push origin feature-x
```

### Staying Up-to-Date
```bash
# Fetch latest changes
asl fetch origin

# Merge into your branch
asl pull origin main
```

### Syncing Fork
```bash
# Add upstream remote
asl remote add upstream https://example.com/original.git

# Fetch from upstream
asl fetch upstream

# Merge upstream changes
asl pull upstream main

# Push to your fork
asl push origin main
```
