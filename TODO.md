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
**Status**: `[ ]` Pending
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
**Acceptance Criteria**:
1. Create the `users` SQL table migration in the database core.
2. Implement CRUD interfaces (Create, Get, Update, Delete) for users.

---

### [SV-006] Implement Repository Data Model
**Status**: `[ ]` Pending
**Context / Files**:
- `internal/db/repositories.go`
**Acceptance Criteria**:
1. Create the `repositories` SQL table migration (referencing the `users` table).
2. Implement CRUD interfaces for repositories.

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
