# SourceVault Agentic TODOs

This file is structured to be easily parsed and executed by an AI assistant.
To trigger work, you can prompt: **"Implement task [ID] from the TODO list."**

---

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

---

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

### [SV-005] Implement User Data Model
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/db/users.go`
- `internal/registry/sync.go`
**Acceptance Criteria**:
1. Create the `users` SQL table migration in the database core.
2. Implement CRUD interfaces (Create, Get, Update, Delete) for users.
3. Implement `SaveUserMetadata(user)` and `RemoveUserMetadata(user)` in `internal/registry/sync.go` to write/delete `Users/{uuid}.yaml` in the registry worktree and commit the change.

---

### [SV-006] Implement Repository Data Model
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/db/repositories.go`
- `internal/registry/sync.go`
**Acceptance Criteria**:
1. Create the `repositories` SQL table migration (referencing the `users` table).
2. Implement CRUD interfaces for repositories.
3. Implement `SaveRepositoryMetadata(repo)` and `RemoveRepositoryMetadata(repo)` in `internal/registry/sync.go` to write/delete `Repositories/{uuid}.yaml` and commit the change.

---

### [SV-004] Implement CA Subcommand
**Status**: `[ ]` Pending
**Context / Files**:
- `cmd/sourcevault/ca.go` (new)
- `internal/crypto` (new)
**Acceptance Criteria**:
1. Create a `sourcevault ca` subcommand with nested commands: `create`, `rotate`, `revoke`, `unseal`, and `seal`.
2. Implement key generation for SSH Certificate Authorities, supporting **both** Ed25519 (default) and RSA-4096 (with SHA-256/SHA-512 signatures) to ensure full FIPS compliance.
3. Save public/private keypairs securely within the `RootDir`, encrypted with a passphrase using `ssh.MarshalPrivateKeyWithPassphrase`.
4. Implement an "Unseal" mechanism (similar to HashiCorp Vault) where the decrypted `ssh.Signer` is temporarily held in a thread-safe memory structure (`sync.RWMutex`), allowing users to self-sign keys without providing the CA password repeatedly.
5. Provide an RPC or internal API to interface with the in-memory signer.

---

### [SV-007] Implement System Registry (Git-First Bootstrap)
**Status**: `[ ]` Pending
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
> Registry sync functions (`SaveUserMetadata`, `SaveRepositoryMetadata`, etc.) are implemented incrementally alongside each data model task (SV-005, SV-006, etc.) — not here.
