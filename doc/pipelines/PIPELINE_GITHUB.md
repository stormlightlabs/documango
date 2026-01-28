# GitHub Markdown Pipeline

Ingest documentation directly from GitHub repositories: READMEs, docs/ folders, and markdown throughout the repo.

## API Endpoints

### Repository Metadata

Get repository info including the default branch:

```http
GET https://api.github.com/repos/{owner}/{repo}
```

Response includes `default_branch` (e.g., "main" or "master") needed for constructing URLs.

### Raw Content (Preferred for Public Repos)

Fetch file contents directly without rate limiting:

```http
GET https://raw.githubusercontent.com/{owner}/{repo}/{ref}/{path}
```

Where `{ref}` is a branch name, tag, or commit SHA. Returns the raw file content with `Content-Type: text/plain`. Returns HTTP 404 for invalid refs or paths.

### Contents API

Fetch file with metadata:

```http
GET https://api.github.com/repos/{owner}/{repo}/contents/{path}
```

Default response is JSON with base64-encoded content:

```json
{
  "name": "README.md",
  "path": "docs/README.md",
  "sha": "abc123...",
  "size": 14822,
  "type": "file",
  "encoding": "base64",
  "content": "IyBUaXRsZQ0K...",
  "download_url": "https://raw.githubusercontent.com/..."
}
```

For raw content directly, add the Accept header:

```http
Accept: application/vnd.github.raw
```

Use `?ref={branch}` query parameter to specify a branch, tag, or commit SHA.

### Directory Listing

Request a directory path via the Contents API to get an array of entries:

```json
[
  {"name": "getting-started.md", "path": "docs/getting-started.md", "type": "file", "sha": "..."},
  {"name": "guides", "path": "docs/guides", "type": "dir", "sha": "..."}
]
```

### Repository Tree (Full Discovery)

Get the complete file tree in one request:

```http
GET https://api.github.com/repos/{owner}/{repo}/git/trees/{ref}?recursive=1
```

Response contains a flat list of all files and directories:

```json
{
  "sha": "abc123...",
  "truncated": false,
  "tree": [
    {"path": "README.md", "mode": "100644", "type": "blob", "sha": "...", "size": 1234},
    {"path": "docs/intro.md", "mode": "100644", "type": "blob", "sha": "...", "size": 567},
    {"path": "docs/guides", "mode": "040000", "type": "tree", "sha": "..."}
  ]
}
```

For very large repositories (e.g., Linux kernel with 72,000+ entries), `truncated` will be `true` and the tree will be incomplete. In this case, fall back to cloning the repository locally.

## Discovery Strategy

Find markdown files in a repository by fetching the tree and filtering:

1. Fetch repository metadata to get `default_branch`
2. Fetch the tree with `?recursive=1`
3. Filter entries where `type` is `"blob"` and `path` ends with `.md` or `.markdown`

Priority locations to check:

- Root: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `LICENSE.md`
- Documentation folders: `docs/`, `doc/`, `documentation/`
- Nested package docs in monorepos: `packages/*/README.md`

## Processing

Markdown files require minimal transformation since they're already in the target format.

### Title Extraction

Extract document title using this priority:

1. First H1 heading in the document (`# Title`)
2. `title` field from YAML front matter
3. Filename without extension (e.g., `getting-started.md` becomes "Getting Started")

### Front Matter Extraction

Many documentation sites use YAML front matter between `---` delimiters:

```markdown
---
title: Installation Guide
description: How to install the package
sidebar_position: 1
---

# Installation
...
```

Parse this for metadata: title, description, tags, and ordering hints.

### Link Resolution

Convert relative links to absolute references:

| Original | Resolved |
|----------|----------|
| `./other-doc.md` | Internal document reference |
| `../README.md` | Internal document reference (if in scope) |
| `../src/file.go` | External link or remove |
| `#section-anchor` | Preserved as anchor link |
| `https://example.com` | Preserved as external link |

## Rate Limiting

Unauthenticated API requests are limited to 60 per hour. Rate limit info is returned in response headers:

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1769615282
X-RateLimit-Used: 15
```

Check current limits via:

```http
GET https://api.github.com/rate_limit
```

Strategies for staying within limits:

- Use raw.githubusercontent.com URLs for fetching content (not rate limited)
- Use the tree API to discover files in one request instead of traversing directories
- Cache the repository tree and SHA to detect when re-fetching is needed
- For repositories exceeding the tree API limit (~100k entries), clone locally

## Clone-Based Fallback

When the tree API returns `truncated: true` or rate limits are exhausted:

1. Shallow clone: `git clone --depth 1 --single-branch {url}`
2. Walk the filesystem for `.md` and `.markdown` files
3. Process files locally
4. Clean up the clone

## Dependencies

- `net/http` for API requests
- SHA comparison for incremental updates
