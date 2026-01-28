# GitHub Markdown Pipeline

Ingest documentation directly from GitHub repositories: READMEs, docs/ folders, wikis.

## Source Acquisition

Two approaches:

**GitHub API** (authenticated, rate-limited):

```text
GET https://api.github.com/repos/{owner}/{repo}/contents/{path}
Accept: application/vnd.github.raw+json
```

**Raw URLs** (simple, no auth for public repos):

```text
https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}
```

## Discovery

Find markdown files in a repository:

1. Fetch repository tree via API
2. Filter for `.md` and `.markdown` extensions
3. Common locations: `README.md`, `docs/`, `wiki/`

## Processing

Markdown files require minimal transformation:

- Already in target format
- Normalize relative links to absolute or internal paths
- Extract front matter if present (YAML/TOML headers)
- Generate title from H1 or filename

## Link Resolution

GitHub-flavored links need conversion:

- `./other-doc.md` becomes internal reference
- `../src/file.go` becomes external link or stripped
- Anchor links preserved

## Agent Context

Extract from markdown:

- Document title (first H1 or filename)
- First paragraph as summary
- Table of contents from headers

## Use Cases

- Project documentation (README, CONTRIBUTING, etc.)
- Tutorial repositories
- Specification documents
- Any markdown-based docs not on a package registry

## Rate Limiting

GitHub API: 60 requests/hour unauthenticated, 5000/hour with token.

For large repositories, prefer cloning locally then processing.

## Dependencies

- `net/http` - API requests
- GitHub personal access token (for private repos or higher limits)
