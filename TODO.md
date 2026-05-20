# SourceVault Agentic TODOs

This file is structured to be easily parsed and executed by an AI assistant.
To trigger work, you can prompt: **"Implement task [ID] from the TODO list."**

**Status Legend:**
- `[ ]` Pending — not yet started
- `[/]` In Progress — currently being implemented
- `[~]` Testing — implemented, awaiting confirmation it works in production
- `[x]` Completed — confirmed working

---

## Milestone: v0.1.0 — Foundation

> Core infrastructure: database layer, system registry, and base data models.
> This milestone establishes the data layer that all future features depend on.

### [SV-003] Implement Database Abstraction Core & Migrations
**Status**: `[x]` Completed
**Context / Files**:
- `internal/config/config.go`
- `sourcevault.env.sample`
- New package: `internal/db`
**Acceptance Criteria**:
1. Add a `DatabaseConfig` block to `config.go` supporting `Driver` (default "sqlite") and `DSN` (default "sourcevault.db").
2. Create `internal/db/db.go` to handle the connection pool (`database/sql`), structuring the initialization to support a `switch` statement for future drivers (postgres/mysql).
3. Ensure the SQLite database file (`sourcevault.db`) is automatically created inside the application's `RootDir` if the driver is "sqlite".
4. Build a lightweight migration engine that is dialect-aware (to support different `CREATE TABLE` syntax later on).

---

### [SV-007] Implement System Registry (Git-First Bootstrap)
**Status**: `[~]` Testing
**Context / Files**:
- `internal/registry` (new package)
- `cmd/sourcevault/start.go`
- `internal/config/config.go`
- Reference: `/Users/jovens/Developer/OLD/projects/sourcevault-ssh/src/db/sync.go`

**Background**:
The system registry is a bare Git repository that acts as the application's source of truth for
configuration data (users, volumes, repos, orgs, CA metadata). Because it defines where all
dynamic volumes live, it must be bootstrapped at a fixed, known path before any other service
reads configuration data.

**Registry Layout (worktree)**:
```
registry/
├── system.git/        ← bare repo (canonical source of truth, push target over SSH)
└── worktree/          ← checked-out clone (what SourceVault reads at runtime)
    ├── Users/         ← {uuid}.yaml per user
    ├── Volumes/       ← {uuid}.yaml per volume definition
    ├── Repositories/  ← {uuid}.yaml per repo
    ├── Organizations/ ← {uuid}.yaml per org
    └── CertificateAuthority/ ← CA metadata only (NO private keys — those live in RootDir/ca/)
```

**Acceptance Criteria**:
1. Add `registry/` to the directory provisioning list in `start.go`.
2. Create `internal/registry/registry.go` implementing `EnsureRegistry(cfg)`:
   - If `registry/system.git` does not exist → `git init --bare`.
   - If `registry/worktree` does not exist → `git clone registry/system.git worktree`.
   - If worktree exists → `git fetch origin && git reset --hard origin/main` (force-sync, never merge).
3. Pre-create the five top-level directories inside the worktree (`Users/`, `Volumes/`, `Repositories/`, `Organizations/`, `CertificateAuthority/`) with a `.gitkeep` so they are tracked in the bare repo from the start.
4. Call `registry.EnsureRegistry(cfg)` in `start.go` **after** filesystem provisioning but **before** database migrations.
5. Add `SOURCEVAULT_REGISTRY_BRANCH` config option (default: `main`) to support non-standard branch names.

> [!NOTE]
> The registry is the **source of truth**. The database is a queryable cache of it.
> Sync functions always flow **registry → DB** (recovery only). Normal writes go to **both** simultaneously — registry first, then DB.

---

## Milestone: v0.2.0 — Security Core

> CA infrastructure must be in place before SSH authentication is useful.
> This milestone provides the cryptographic foundation the SSH server depends on.

### [SV-004] Implement Local CA Management
**Status**: `[x]` Completed
**Context / Files**:
- `cmd/sourcevault/ca.go`
- `internal/crypto` (new files: `bootstrap.go`, `ca.go`, `signer.go`, `krl.go`)
- `internal/registry/sync.go` (extended)
- `internal/db/ca.go` (new)
**Acceptance Criteria**:
1. Create a `sourcevault ca` subcommand with nested commands: `create`, `rotate`, `revoke`, `unseal`, `seal`, `status`, and `sign`.
2. Implement key generation supporting **both** Ed25519 (default) and RSA-4096. Key type and parameters respect the active Crypto Policy (SV-010).
3. Save encrypted public/private keypairs locally in `RootDir/data/ca/` and backup metadata (including encrypted private key) to the system registry.
4. Implement an **automatic bootstrap logic** during server startup:
   - Perform a **full sync** of CA metadata from the Git registry to the SQLite database.
   - If missing locally, restore the **active CA** files from the database cache.
5. Track the authoritative CA in the registry via `ActiveCA.yaml` to prevent regression.
6. Support automatic "Unseal" on startup if `SOURCEVAULT_CA_PASSPHRASE` is provided in the environment.
7. Implement **Database Caching**: cache all CA metadata (UUID, fingerprint, algorithm, status, sealed key) in the SQLite database for fast querying and restoration.
8. Implement `sourcevault ca unseal` with **automated discovery** (no path flag required).
9. Implement `sourcevault ca sign [pubkey]` to issue signed SSH certificates using the unsealed CA via the RPC bridge or local process.
10. Implement KRL (Key Revocation List) generation for OpenSSH compatibility.

---

### [SV-008] Federated CA Trust (Decentralized Node Peering)
**Status**: `[ ]` Pending
**Context / Files**:
- `cmd/sourcevault/ca.go` (extend)
- `internal/crypto` (extend)
- `internal/registry/sync.go`
- `internal/config/config.go`
**Background**:
For decentralized deployments, multiple SourceVault nodes need to mutually trust each other's CAs so that a user certificate issued by Node A is accepted by Node B. SSH Certificates are **required** for decentralized hosts — plain key auth is not accepted between federated nodes.

**Acceptance Criteria**:
1. Add `sourcevault ca import` subcommand to import a foreign CA's public key from a file or URL.
2. Add `sourcevault ca trust` / `sourcevault ca revoke-trust` to manage the trusted-CA list.
3. Store trusted foreign CA public keys in `registry/worktree/CertificateAuthority/trusted/{uuid}.yaml`, including origin node identifier, fingerprint, algorithm, and import date.
4. On startup, the SSH server (SV-002) must load all trusted CA public keys so it can validate certificates issued by any trusted node.
5. Ensure revoked CAs are immediately rejected — the in-memory trusted CA list must be reloadable without restart.

---

### [SV-009] SSH Key Trust Management (Non-Certificate Auth)
**Status**: `[ ]` Pending
**Context / Files**:
- `cmd/sourcevault/key.go` (new)
- `internal/db/users.go` (extend)
- `internal/registry/sync.go`
**Background**:
Users who are not part of a decentralized setup, or who prefer not to use SSH certificates, may authenticate with a plain trusted SSH public key. This is the fallback/standalone auth method.

**Acceptance Criteria**:
1. Add a `sourcevault key` subcommand with: `add`, `list`, `revoke`.
2. Trusted public keys are stored per-user in the database and written to `registry/worktree/Users/{uuid}.yaml` as a `trusted_keys` list (fingerprints + full public key).
3. Support configurable **expiration** for trusted keys (e.g. `--valid-for 8760h` or `--expires 2027-01-01`). Expired keys are automatically rejected by the SSH server at auth time.
4. The SSH server (SV-002) must consult the trusted key store at auth time, rejecting expired or revoked keys.
5. Plain SSH key auth must be **disabled** for any user connecting from a federated/decentralized node — those connections require a valid SSH certificate from a trusted CA.

---

### [SV-010] Crypto Policy Configuration (FIPS & Cipher Control)
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/config/config.go`
- `internal/crypto` (new or extend)
- `sourcevault.env.sample`
**Background**:
Operators on FIPS-compliant infrastructure must be able to restrict the allowed cryptographic primitives. This policy is enforced globally — any CA creation, key import, or SSH negotiation that violates the policy is rejected.

**Acceptance Criteria**:
1. Add a `CryptoPolicy` block to `config.go` with the following fields:
   - `AllowedKeyTypes []string` (e.g. `["rsa", "ed25519"]`; FIPS mode: `["rsa"]`)
   - `MinRSABits int` (default: `3072`; FIPS minimum: `4096`)
   - `AllowedMACs []string` (allowed SSH MAC algorithms)
   - `AllowedCiphers []string` (allowed SSH symmetric ciphers)
   - `AllowedKEX []string` (allowed key exchange algorithms)
   - `MaxCAValidityDays int` (maximum CA certificate validity in days)
   - `MaxKeyValidityDays int` (maximum user key validity in days)
   - `FIPSMode bool` (shorthand: sets all of the above to FIPS-compliant values)
2. Expose all fields via `SOURCEVAULT_CRYPTO_*` environment variables.
3. Implement a `crypto.ValidatePolicy(cfg)` function called at startup that returns an error if the configured policy is self-contradictory (e.g. `FIPSMode=true` but `AllowedKeyTypes` includes `ed25519`).
4. The SSH server (SV-002) must pass the allowed ciphers/MACs/KEX lists to the `ssh.ServerConfig` during negotiation.
5. CA creation (SV-004) and key import (SV-009) must validate the requested key type and parameters against the active policy before proceeding.

---

## Milestone: v0.3.0 — Git Operations

> The SSH server is the core product feature. It depends on the CA (v0.2.0) for
> certificate-based authentication and the data models (v0.1.0) for access control.

### [SV-002] Implement Internal SSH Server Skeleton
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/config/config.go` (SshConfig)
- `cmd/sourcevault/start.go`
- New package: `internal/ssh`
**Acceptance Criteria**:
1. Create a basic SSH server using `golang.org/x/crypto/ssh` or `github.com/gliderlabs/ssh`.
2. Bind the server to the Host and Port specified in `cfg.Ssh`.
3. Wire the server into the `errgroup` in `start.go` so it starts alongside the Web and Metrics servers.
4. Ensure it respects `ctx.Done()` for graceful shutdown.

---

### [SV-005] Implement User Data Model
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/db/users.go`
- `internal/registry/sync.go`
**Acceptance Criteria**:
1. Create the `users` SQL table migration in the database core.
2. Implement CRUD interfaces (Create, Get, Update, Delete) for users. Every write operation must:
   - Write `Users/{uuid}.yaml` to the registry worktree and commit.
   - Write the same data to the SQLite database.
   Both writes should be treated as a single logical operation — if the registry write fails, the DB write should not proceed.
3. Implement `SyncUsersFromRegistry()` in `internal/registry/sync.go` to read all `Users/*.yaml` files and reconstruct the `users` table. This is the **recovery path only** (e.g. after DB corruption).

---

### [SV-006] Implement Repository Data Model
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/db/repositories.go`
- `internal/registry/sync.go`
**Acceptance Criteria**:
1. Create the `repositories` SQL table migration (referencing the `users` table).
2. Implement CRUD interfaces for repositories. Every write operation must:
   - Write `Repositories/{uuid}.yaml` to the registry worktree and commit.
   - Write the same data to the SQLite database.
   Both writes should be treated as a single logical operation — if the registry write fails, the DB write should not proceed.
3. Implement `SyncRepositoriesFromRegistry()` in `internal/registry/sync.go` to read all `Repositories/*.yaml` files and reconstruct the `repositories` table. This is the **recovery path only**.

---

### [SV-011] Implement Restricted SSH Shell & Git Session Handler
**Status**: `[ ]` Pending
**Depends on**: SV-002 (SSH server), SV-004 (CA/signing), SV-005 (user model), SV-006 (repo model)
**Context / Files**:
- `internal/ssh/shell.go` (new)
- `internal/ssh/git.go` (new)
- `internal/ssh/sanitizer.go` (new)
**Background**:
When a user connects over SSH they land in a restricted SourceVault shell — not a system shell. The shell handles two modes:
- **Interactive mode**: a custom prompt for administrative operations (key management, CA status, repo listing, etc.).
- **Git mode**: non-interactive execution of git transfer commands triggered by the client (`git clone`, `git push`, `git fetch`).

The shell must make it extremely difficult to escape to system commands.

**Acceptance Criteria**:
1. Implement a command allowlist sanitizer in `internal/ssh/sanitizer.go`:
   - Only `git-upload-pack`, `git-receive-pack`, and `git-upload-archive` may be executed.
   - All other exec requests are rejected and logged at `Warn`.
   - Input is validated against a strict regex — no shell metacharacters (`|`, `;`, `&`, `$`, backticks, etc.) are permitted.
   - Repository paths are validated to be within `RootDir/volumes/` and must not contain `..` traversal sequences.
2. Implement `internal/ssh/git.go` for non-interactive git session execution:
   - Parse the SSH exec request (e.g. `git-upload-pack '/org/repo.git'`).
   - Validate command and path through the sanitizer.
   - Execute the allowed git binary with the sanitized absolute path, piping stdin/stdout/stderr to the SSH channel.
3. Implement `internal/ssh/shell.go` for the interactive prompt:
   - Displayed when a user connects without an exec request (no command).
   - Supports a defined set of built-in commands: `help`, `whoami`, `repos`, `keys`, `exit`.
   - All other input is rejected with `"command not permitted"`.
   - Prompt displays the authenticated username.
4. The SSH server (SV-002) routes sessions: exec requests → git handler, no-command sessions → interactive shell.
5. Log every session open/close, every allowed command, and every rejected command at `Info`/`Warn`.

---

## Milestone: v1.0.0 — Production Ready

> First stable public release. Scope TBD — add tasks here as the earlier milestones complete.

---

## Backlog

> Nice-to-have features that are not blocking any milestone.

### [SV-001] Implement Auto-Update Mechanism
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/config/config.go`
- `cmd/sourcevault/start.go`
- New package: `internal/updater`
**Acceptance Criteria**:
1. Implement a `go-selfupdate` wrapper in `internal/updater` to poll for new GitHub releases.
2. Add an `AutoUpdate` toggle flag to `config.go` and `sourcevault.env.sample`.
3. If enabled, start a background goroutine in `start.go` to check for updates daily.
4. Expose a `sourcevault update` cobra command for manual execution.
5. Provide logging and graceful restart instructions upon binary replacement.
