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

### [SV-003] Implement SQLite Database Core
**Status**: `[ ]` Pending
**Context / Files**:
- New package: `internal/db`
**Acceptance Criteria**:
1. Create `internal/db/db.go` containing the connection pool setup and schema migrations.
2. Ensure the SQLite file is created inside the application's `RootDir`.
3. Break domain logic into separate files (e.g., `users.go`, `repositories.go`) rather than a monolithic file.
4. Implement basic CRUD interfaces for users and organizations.

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
