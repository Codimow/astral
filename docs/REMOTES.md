# Working with Remotes

## Adding Remotes
asl remote add origin https://example.com/repo.git

## Cloning
asl clone https://example.com/repo.git

## Pushing
asl push origin main

## Pulling
asl pull origin main

## Fetching
asl fetch origin

## Authentication
Astral supports the following authentication methods:
- **None**: For public repositories
- **Basic Auth**: Using username and password
- **Token**: Using a bearer token (recommended)

Credentials can be provided via environment variables or interactive prompts.
