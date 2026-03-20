# GitHub Lens

A lightweight, local web dashboard that aggregates open Issues and Pull Requests across all repositories of one or more GitHub organizations — so you never have to click through dozens of repos again.

**Stack:** Go (backend) · Svelte + DaisyUI (frontend) · SQLite (local cache)

---

## The Problem

Navigating through multiple repositories across GitHub organizations to track open Issues and PRs is tedious. Existing tools are either too complex, cloud-hosted, or require paid subscriptions for something that should be simple.

## The Solution

GitHub Lens runs locally, pulls Issues and PRs via the GitHub API, caches them in SQLite, and presents them in a fast, searchable, beautiful web interface.

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Browser                       │
│         Svelte + DaisyUI (port 5173)            │
└──────────────────┬──────────────────────────────┘
                   │ HTTP/JSON
┌──────────────────▼──────────────────────────────┐
│               Go Backend (port 8080)            │
│                                                 │
│  ┌────────────┐  ┌───────────┐  ┌─────────────┐ │
│  │ API Router │  │ Sync Svc  │  │ Config Mgr  │ │
│  └─────┬──────┘  └─────┬─────┘  └──────┬──────┘ │
│        │               │               │        │
│  ┌─────▼───────────────▼───────────────▼──────┐ │
│  │            SQLite (local cache)            │ │
│  └─────────────────────┬──────────────────────┘ │
└────────────────────────┼────────────────────────┘
                         │ GitHub REST API v3
┌────────────────────────▼────────────────────────┐
│                  GitHub API                     │
└─────────────────────────────────────────────────┘
```

### Go Backend

| Package | Responsibility |
|---|---|
| `cmd/github-lens` | Entrypoint, starts HTTP server, serves embedded frontend |
| `internal/config` | Load/validate YAML config (orgs, repos, token) |
| `internal/github` | GitHub API client — fetch issues, PRs, rate-limit handling, pagination |
| `internal/store` | SQLite access layer — upsert issues/PRs, full-text search |
| `internal/sync` | Orchestrates fetching from GitHub and writing to the store |
| `internal/api` | HTTP handlers — REST endpoints for the frontend |

### Svelte Frontend

| Area | Details |
|---|---|
| Framework | SvelteKit (static adapter, SPA mode) |
| UI Library | DaisyUI (Tailwind CSS) |
| Build | Embedded into the Go binary via `embed.FS` for single-binary distribution |

---

## Configuration

A single `config.yaml` file in the working directory (or `~/.config/github-lens/config.yaml`).

> **Warning:** Never commit `config.yaml` to version control — it contains your GitHub token. The `.gitignore` already excludes it. Use `config.example.yaml` as a template.

```yaml
# GitHub personal access token (read-only scope: repo, read:org)
github_token: "ghp_xxxxxxxxxxxxxxxxxxxx"

# Organizations to track
organizations:
  - name: "my-org"
    # Optional: only include specific repos (omit for all repos)
    include_repos:
      - "backend-api"
      - "frontend-app"
    # Optional: exclude specific repos
    exclude_repos:
      - "archived-thing"

  - name: "another-org"
    # No filters = all repositories

# Server settings
server:
  port: 8080

# Sync settings
sync:
  # Auto-sync interval (0 = manual only)
  interval: "15m"
  # Max concurrent API requests
  concurrency: 5
```

The token can also be provided via the `GITHUB_TOKEN` environment variable (takes precedence over config).

---

## Data Model (SQLite)

```sql
CREATE TABLE items (
    id              INTEGER PRIMARY KEY,
    github_id       INTEGER NOT NULL UNIQUE,
    type            TEXT NOT NULL CHECK(type IN ('issue', 'pr')),
    state           TEXT NOT NULL CHECK(state IN ('open', 'closed', 'merged')),
    title           TEXT NOT NULL,
    body            TEXT,
    url             TEXT NOT NULL,           -- HTML URL for opening in browser
    number          INTEGER NOT NULL,
    org             TEXT NOT NULL,
    repo            TEXT NOT NULL,
    author          TEXT NOT NULL,
    author_avatar   TEXT,
    labels          TEXT,                    -- comma-separated for FTS compatibility
    assignees       TEXT,                    -- JSON array
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    synced_at       DATETIME NOT NULL
);

CREATE VIRTUAL TABLE items_fts USING fts5(title, body, labels, repo, org, author);

CREATE TABLE sync_log (
    id          INTEGER PRIMARY KEY,
    org         TEXT NOT NULL,
    repo        TEXT NOT NULL,
    started_at  DATETIME NOT NULL,
    finished_at DATETIME,
    status      TEXT,                        -- success, error
    items_count INTEGER DEFAULT 0,
    error       TEXT
);
```

---

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/items` | List items (supports query params below) |
| `GET` | `/api/items/:id` | Get single item detail |
| `POST` | `/api/sync` | Trigger a full sync (returns `202 Accepted`, runs async) |
| `GET` | `/api/sync/status` | Current sync status (running, last sync time) |
| `GET` | `/api/config/orgs` | List configured organizations |
| `GET` | `/api/repos` | List all tracked repositories (distinct org/repo pairs) |
| `GET` | `/api/labels` | List all labels across all items |
| `GET` | `/api/authors` | List all authors across all items |
| `GET` | `/api/stats` | Dashboard stats (counts by org, repo, type) |

### Error Responses

All endpoints return errors in a consistent JSON envelope:

```json
{
  "error": "descriptive message",
  "code": "NOT_FOUND"
}
```

Standard HTTP status codes are used (`400`, `404`, `500`, etc.).

### Query Parameters for `/api/items`

| Param | Type | Example | Description |
|---|---|---|---|
| `q` | string | `q=auth+bug` | Full-text search across title, body, labels |
| `type` | string | `type=pr` | Filter by `issue` or `pr` |
| `state` | string | `state=open` | Filter by `open`, `closed`, or `merged` |
| `org` | string | `org=my-org` | Filter by organization |
| `repo` | string | `repo=backend-api` | Filter by repository |
| `author` | string | `author=octocat` | Filter by author |
| `label` | string | `label=bug` | Filter by label |
| `sort` | string | `sort=updated_at` | Sort field (default: `updated_at`) |
| `order` | string | `order=desc` | Sort order: `asc` or `desc` |
| `page` | int | `page=2` | Page number (default: 1) |
| `per_page` | int | `per_page=50` | Items per page (default: 25, max: 100) |

---

## Frontend UI

### Pages & Components

**Dashboard (`/`)**
- Stats bar: total open issues, total open PRs, number of repos tracked
- Quick filter chips for orgs
- Combined table of all open items, sorted by last updated
- Sync button with spinner + "last synced X minutes ago" indicator

**Table View (main component)**
- DaisyUI `table` with columns: Type (icon), Title, Repo, Author (avatar + name), Labels (badges), Updated
- Sortable column headers (click to sort)
- Row click opens the GitHub URL in a new tab
- Inline preview drawer (slide-in panel) showing issue/PR body in markdown on click of an expand icon
- Pagination controls at the bottom

**Search & Filters Bar**
- DaisyUI `input` with search icon — debounced full-text search
- DaisyUI `select` dropdowns for: Organization, Repository, Type (Issue/PR), State (Open/Closed/Merged), Label, Author
- State filter defaults to "Open" — showing only open items on first load
- Active filters shown as DaisyUI `badge` chips with dismiss button
- "Clear all filters" link

**Sync Status**
- DaisyUI `button` with sync icon — triggers manual sync
- During sync: loading spinner + progress toast
- After sync: success/error toast notification

### Theme

- DaisyUI theme: `corporate` (clean, professional) with dark mode toggle (`dark` theme)
- Responsive layout using Tailwind breakpoints

### Component Tree

```
App
├── Navbar
│   ├── Logo + Title
│   ├── ThemeToggle (light/dark)
│   └── SyncButton + LastSyncIndicator
├── StatsBar
│   ├── StatCard (Open Issues)
│   ├── StatCard (Open PRs)
│   └── StatCard (Repos Tracked)
├── FilterBar
│   ├── SearchInput
│   ├── OrgSelect
│   ├── RepoSelect
│   ├── TypeSelect
│   ├── StateSelect
│   ├── LabelSelect
│   ├── AuthorSelect
│   └── ActiveFilterBadges
├── ItemsTable
│   ├── TableHeader (sortable columns)
│   ├── TableRow[]
│   │   ├── TypeIcon (issue/pr)
│   │   ├── Title + Labels (badges)
│   │   ├── Repo (org/repo)
│   │   ├── Author (avatar + name)
│   │   ├── Updated (relative time)
│   │   └── ExpandButton
│   └── Pagination
└── DetailDrawer (slide-in panel)
    ├── ItemHeader (title, number, state badge)
    ├── ItemMeta (author, created, updated, assignees)
    ├── MarkdownBody
    └── OpenOnGitHubButton
```

---

## Project Structure

```
github-lens/
├── cmd/
│   └── github-lens/
│       └── main.go                 # Entrypoint
├── internal/
│   ├── api/
│   │   ├── handler.go              # HTTP handlers
│   │   ├── middleware.go            # CORS, logging
│   │   └── router.go               # Route definitions
│   ├── config/
│   │   └── config.go               # YAML config loading
│   ├── github/
│   │   ├── client.go               # GitHub API client
│   │   └── types.go                # API response types
│   ├── store/
│   │   ├── sqlite.go               # SQLite operations
│   │   ├── migrations.go           # Schema migrations
│   │   └── search.go               # FTS5 search queries
│   └── sync/
│       └── sync.go                 # Sync orchestration
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── Navbar.svelte
│   │   │   │   ├── StatsBar.svelte
│   │   │   │   ├── FilterBar.svelte
│   │   │   │   ├── ItemsTable.svelte
│   │   │   │   ├── TableRow.svelte
│   │   │   │   ├── DetailDrawer.svelte
│   │   │   │   ├── SyncButton.svelte
│   │   │   │   └── ThemeToggle.svelte
│   │   │   ├── api.ts              # Backend API client
│   │   │   ├── types.ts            # TypeScript types
│   │   │   └── stores.ts           # Svelte stores (filters, items)
│   │   ├── routes/
│   │   │   └── +page.svelte        # Main (and only) page
│   │   └── app.html
│   ├── static/
│   ├── svelte.config.js
│   ├── tailwind.config.js          # DaisyUI plugin config
│   ├── package.json
│   └── vite.config.ts
├── .github/
│   └── workflows/
│       ├── ci.yaml                  # Lint + test on every push/PR
│       ├── release.yaml             # Build, sign, SBOM, publish on tag
│       └── codeql.yaml              # GitHub CodeQL security scanning
├── .goreleaser.yaml                 # GoReleaser configuration
├── config.example.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Build & Run

```bash
# Prerequisites: Go 1.22+, Node.js 20+

# Clone and configure
git clone <repo-url> && cd github-lens
cp config.example.yaml config.yaml
# Edit config.yaml with your token + orgs

# Build everything (frontend + backend → single binary)
make build

# Run
./github-lens
# → open http://localhost:8080

# Development (hot-reload)
make dev
# → frontend on :5173 (proxied to backend on :8080)
```

### Makefile Targets

| Target | Description |
|---|---|
| `make build` | Build frontend, embed into Go binary, compile |
| `make dev` | Run backend + frontend dev server concurrently |
| `make sync` | Trigger a one-time sync via CLI flag |
| `make clean` | Remove build artifacts and local DB |

---

## CI/CD & Release Pipeline

All pipelines are GitHub Actions workflows in `.github/workflows/`. The goal: every push is linted and tested, every tag produces signed, SBOM-enriched release artifacts.

> **Note:** The YAML snippets below are reference copies. The source of truth is in `.github/workflows/` and `.goreleaser.yaml`.

### Overview

```
  push / PR to main                       push tag v*
        │                                      │
        ▼                                      ▼
  ┌───────────┐                        ┌──────────────┐
  │  ci.yaml  │                        │ release.yaml │
  └─────┬─────┘                        └──────┬───────┘
        │                                     │
  ┌─────▼──────────────┐        ┌─────────────▼─────────────────────┐
  │ Lint & Check       │        │ Native Build Matrix               │
  │  • golangci-lint   │        │  linux/amd64   (ubuntu-latest)    │
  │  • go vet          │        │  linux/arm64   (ubuntu-24.04-arm) │
  │  • govulncheck     │        │  darwin/arm64  (macos-latest)     │
  │  • eslint + svelte │        │  darwin/amd64  (macos-13)         │
  │    -check          │        │  windows/amd64 (windows-latest)   │
  ├────────────────────┤        ├───────────────────────────────────┤
  │ Test               │        │ Per-platform steps                │
  │  • go test ./...   │        │  • go build (native, no cross)    │
  │  • vitest          │        │  • macOS: codesign + notarize     │
  ├────────────────────┤        │  • Upload archive as artifact     │
  │ Build (verify)     │        └───────────────┬───────────────────┘
  │  • make build      │                        │
  └────────────────────┘        ┌────────────────▼──────────────────┐
                                │ Publish Release                   │
                                │  • Download all archives          │
                                │  • SHA-256 checksums              │
                                │  • Cosign keyless (Sigstore)      │
                                │  • SBOM (SPDX + CycloneDX)        │
                                │  • SLSA provenance attestation    │
                                │  • Create GitHub Release          │
                                └──────────────┬────────────────────┘
                                               ▼
                                  GitHub Release published
                                  with signed artifacts
```

### Workflow: `ci.yaml` — Lint, Test, Build

Runs on every push and pull request targeting `main`.

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  lint-go:
    name: Lint Go
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
      - run: go vet ./...
      - name: govulncheck
        uses: golang/govulncheck-action@v1
        with:
          go-version-file: go.mod

  lint-frontend:
    name: Lint Frontend
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: frontend
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
      - run: npm run lint
      - run: npm run check          # svelte-check (type checking)

  test-go:
    name: Test Go
    runs-on: ubuntu-latest
    needs: lint-go
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: go-coverage
          path: coverage.out

  test-frontend:
    name: Test Frontend
    runs-on: ubuntu-latest
    needs: lint-frontend
    defaults:
      run:
        working-directory: frontend
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
      - run: npm run test

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [test-go, test-frontend]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: make build
      - name: Verify binary runs
        run: ./github-lens --version
```

### Workflow: `release.yaml` — Native Build, Sign, SBOM, Publish

Runs when a version tag (`v*`) is pushed. Each target platform builds on its own native runner — no cross-compilation. Darwin binaries are codesigned with an Apple Developer ID certificate and notarized via Apple's notary service. All release artifacts are additionally signed with [Sigstore](https://www.sigstore.dev/) (keyless).

#### Required Secrets

| Secret | Purpose |
|---|---|
| `APPLE_CERTIFICATE_P12` | Base64-encoded .p12 Developer ID Application certificate |
| `APPLE_CERTIFICATE_PASSWORD` | Password for the .p12 file |
| `APPLE_SIGNING_IDENTITY` | Certificate CN, e.g. `Developer ID Application: Name (TEAMID)` |
| `APPLE_ID` | Apple ID email for notarytool |
| `APPLE_ID_PASSWORD` | App-specific password for notarytool |
| `APPLE_TEAM_ID` | Apple Developer Team ID |

```yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write
  id-token: write
  attestations: write

jobs:
  build:
    name: Build (${{ matrix.goos }}/${{ matrix.goarch }})
    strategy:
      fail-fast: false
      matrix:
        include:
          - { goos: linux,   goarch: amd64, runner: ubuntu-latest }
          - { goos: linux,   goarch: arm64, runner: ubuntu-24.04-arm }
          - { goos: darwin,  goarch: arm64, runner: macos-latest }
          - { goos: darwin,  goarch: amd64, runner: macos-13 }
          - { goos: windows, goarch: amd64, runner: windows-latest }
    runs-on: ${{ matrix.runner }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json

      - name: Build frontend
        working-directory: frontend
        run: npm ci && npm run build

      - name: Build binary
        env:
          CGO_ENABLED: "0"
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        shell: bash
        run: |
          VERSION=${GITHUB_REF_NAME#v}
          EXT=$([[ "${{ matrix.goos }}" == "windows" ]] && echo ".exe" || echo "")
          go build -trimpath \
            -ldflags "-s -w \
              -X main.version=$VERSION \
              -X main.commit=$GITHUB_SHA \
              -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            -o github-lens${EXT} ./cmd/github-lens

      # macOS: codesign with Developer ID + notarize via Apple
      - name: macOS codesign
        if: matrix.goos == 'darwin'
        env:
          CERTIFICATE_P12: ${{ secrets.APPLE_CERTIFICATE_P12 }}
          CERTIFICATE_PASSWORD: ${{ secrets.APPLE_CERTIFICATE_PASSWORD }}
          SIGNING_IDENTITY: ${{ secrets.APPLE_SIGNING_IDENTITY }}
        run: |
          echo "$CERTIFICATE_P12" | base64 --decode > cert.p12
          KEYCHAIN_PW=$(openssl rand -hex 16)
          security create-keychain -p "$KEYCHAIN_PW" build.keychain
          security default-keychain -s build.keychain
          security unlock-keychain -p "$KEYCHAIN_PW" build.keychain
          security import cert.p12 -k build.keychain \
            -P "$CERTIFICATE_PASSWORD" -T /usr/bin/codesign
          security set-key-partition-list -S apple-tool:,apple: \
            -s -k "$KEYCHAIN_PW" build.keychain

          codesign --force --options runtime \
            --sign "$SIGNING_IDENTITY" --timestamp github-lens

          rm -f cert.p12

      - name: macOS notarize
        if: matrix.goos == 'darwin'
        env:
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_ID_PASSWORD: ${{ secrets.APPLE_ID_PASSWORD }}
          APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
        run: |
          zip -j github-lens-notarize.zip github-lens
          xcrun notarytool submit github-lens-notarize.zip \
            --apple-id "$APPLE_ID" \
            --password "$APPLE_ID_PASSWORD" \
            --team-id "$APPLE_TEAM_ID" \
            --wait
          rm github-lens-notarize.zip

      - name: Create archive
        shell: bash
        run: |
          VERSION=${GITHUB_REF_NAME#v}
          ARCHIVE=github-lens_${VERSION}_${{ matrix.goos }}_${{ matrix.goarch }}
          if [[ "${{ matrix.goos }}" == "windows" ]]; then
            zip ${ARCHIVE}.zip github-lens.exe README.md LICENSE config.example.yaml
          else
            tar czf ${ARCHIVE}.tar.gz github-lens README.md LICENSE config.example.yaml
          fi

      - uses: actions/upload-artifact@v4
        with:
          name: archive-${{ matrix.goos }}-${{ matrix.goarch }}
          path: github-lens_*
          retention-days: 1

  release:
    name: Publish Release
    needs: build
    runs-on: ubuntu-latest
    permissions:
      contents: write
      id-token: write
      attestations: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/download-artifact@v4
        with:
          pattern: archive-*
          path: dist
          merge-multiple: true

      # Checksums
      - name: Generate checksums
        working-directory: dist
        run: sha256sum github-lens_* > checksums.txt

      # Cosign keyless signing (Sigstore OIDC)
      - uses: sigstore/cosign-installer@v3

      - name: Sign artifacts with Cosign
        working-directory: dist
        run: |
          for f in *.tar.gz *.zip checksums.txt; do
            cosign sign-blob --yes \
              --output-signature "${f}.sig" \
              --output-certificate "${f}.pem" "${f}"
          done

      # SBOM generation
      - uses: anchore/sbom-action/download-syft@v0

      - name: Generate SBOMs
        run: |
          syft dir:. -o spdx-json=dist/github-lens.spdx.json
          syft dir:. -o cyclonedx-json=dist/github-lens.cdx.json

      - name: Sign SBOMs
        working-directory: dist
        run: |
          for f in *.spdx.json *.cdx.json; do
            cosign sign-blob --yes \
              --output-signature "${f}.sig" \
              --output-certificate "${f}.pem" "${f}"
          done

      # SLSA provenance
      - name: Attest build provenance
        uses: actions/attest-build-provenance@v2
        with:
          subject-path: |
            dist/*.tar.gz
            dist/*.zip

      # Create GitHub Release with all artifacts
      - name: Create release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh release create "$GITHUB_REF_NAME" dist/* \
            --title "$GITHUB_REF_NAME" \
            --generate-notes
```

### Workflow: `codeql.yaml` — Security Scanning

```yaml
name: CodeQL

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: "0 6 * * 1"    # Weekly Monday 06:00 UTC

permissions:
  security-events: write
  contents: read

jobs:
  analyze:
    name: Analyze
    runs-on: ubuntu-latest
    strategy:
      matrix:
        language: [go, javascript]
    steps:
      - uses: actions/checkout@v4
      - uses: github/codeql-action/init@v3
        with:
          languages: ${{ matrix.language }}
      - uses: github/codeql-action/autobuild@v3
      - uses: github/codeql-action/analyze@v3
```

### GoReleaser Configuration (`.goreleaser.yaml`) — Local Builds

GoReleaser is available for local snapshot builds (`goreleaser build --single-target --snapshot --clean`). The CI release pipeline uses native runners with `go build` directly (see above).

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: github-lens
    main: ./cmd/github-lens
    binary: github-lens
    env:
      - CGO_ENABLED=0
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: >-
      github-lens_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    files:
      - README.md
      - LICENSE
      - config.example.yaml
```

### Supply Chain Security Summary

| Measure | Tool | What It Provides |
|---|---|---|
| **Code scanning** | CodeQL | SAST for Go + JS — catches vulnerabilities before merge |
| **Dependency vulnerabilities** | govulncheck | Checks Go deps against the Go vulnerability database |
| **macOS code signing** | Apple `codesign` | Developer ID signature — Gatekeeper trusts the binary without warnings |
| **macOS notarization** | Apple `notarytool` | Apple scans the binary for malware and records approval — required for distribution outside the App Store |
| **Artifact signing** | Cosign (keyless) | Cryptographic proof that artifacts were built by this CI — no key management needed |
| **SBOM** | Syft | Full software bill of materials in SPDX and CycloneDX formats |
| **Build provenance** | SLSA (GitHub attestations) | Verifiable proof of *where* and *how* the binary was built |
| **Checksums** | SHA-256 | Checksums for all archives, generated in the release job |
| **Linting** | golangci-lint, eslint | Catches code quality issues and potential bugs early |
| **Race detection** | `go test -race` | Catches concurrency bugs in CI |
| **Native builds** | GitHub Actions matrix | Each platform builds on its own runner — no cross-compilation, no toolchain mismatches |

### Verifying a Release

Users can verify the integrity and provenance of downloaded artifacts:

```bash
# Verify Cosign signature (keyless — uses Sigstore transparency log)
cosign verify-blob \
  --signature github-lens_1.0.0_linux_amd64.tar.gz.sig \
  --certificate github-lens_1.0.0_linux_amd64.tar.gz.pem \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  github-lens_1.0.0_linux_amd64.tar.gz

# Verify SHA-256 checksum
sha256sum -c checksums.txt

# Verify GitHub build provenance attestation
gh attestation verify github-lens_1.0.0_linux_amd64.tar.gz \
  --owner <org>

# macOS: verify code signature (Gatekeeper does this automatically)
codesign --verify --verbose=2 github-lens
spctl --assess --verbose=2 github-lens

# Inspect SBOM
cat github-lens.spdx.json | jq '.packages | length'
```

---

## Key Design Decisions

1. **Single binary** — The Svelte frontend is compiled and embedded into the Go binary via `embed.FS`. No separate web server needed. Download, configure, run.

2. **SQLite + FTS5 (pure Go)** — Uses `modernc.org/sqlite` (pure-Go, no CGO) for truly portable cross-compilation. Lightweight, zero-config, plenty fast for this use case. Full-text search via FTS5 virtual tables means instant search without external dependencies.

3. **Background sync with manual trigger** — Configurable auto-sync interval, plus a manual sync button. The UI always reads from the local cache for instant response.

4. **GitHub REST API v3** — Simpler than GraphQL for this use case. Handles pagination and rate limiting gracefully with exponential backoff.

5. **DaisyUI** — Provides polished, accessible components out of the box without writing custom CSS. Theme switching is trivial.

---

## Future Ideas (Out of Scope for v1)

- Keyboard shortcuts (j/k navigation, `/` to focus search)
- Notification badges for new items since last visit
- PR review status and CI check indicators
- Bookmark/pin specific issues
- Export filtered results as CSV
- GitHub webhook support for real-time updates instead of polling
