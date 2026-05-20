# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **DB-Backed CA Restoration** (`internal/crypto`): Enhanced `EnsureCA()` to restore missing local CA files from the SQLite database cache after registry synchronization. This provides a fast-recovery path if `data/ca/` is cleared.
- **Full CA Metadata Sync** (`internal/registry`): Implemented a complete synchronization logic that seeds the SQLite database with the full history of CA metadata from the Git registry on every server startup.
- **Streamlined CA Unsealing** (`cmd/sourcevault ca unseal`): Refactored the `unseal` command to automatically discover the active CA UUID and file path from the system registry, removing the need for the manual `--key` flag.
- **CA Bootstrap Logic** (`internal/crypto`): Implemented automated management of the application's Certificate Authority during startup, ensuring local files are in sync with the registry's authoritative active CA.
- **Registry Active CA Tracking** (`internal/registry`): Updated the system registry to securely store encrypted CA private keys and track the authoritative CA via `ActiveCA.yaml`.
- **SSH Certificate Signing** (`cmd/sourcevault ca sign`): Implemented the ability to sign SSH public keys with the unsealed local CA via the CLI or RPC bridge.
- **RPC Bridge for CLI-Server Communication** (`internal/rpc`): Implemented a Unix Domain Socket RPC server to allow the CLI to communicate with the running SourceVault daemon.
- **Enhanced CA CLI**: Updated `ca status`, `ca unseal`, `ca seal`, and `ca sign` subcommands to prefer RPC communication with the active server, enabling live state management.
- **Crypto & Registry Unit Tests**: Developed a comprehensive test suite for `internal/crypto` and `internal/registry`.
    - `internal/crypto`: Verified Ed25519/RSA CA key generation, in-memory unsealing/sealing, certificate signing logic, and OpenSSH-compatible KRL production.
    - `internal/registry`: Implemented integration tests with a temporary Git repository to verify CA metadata persistence and revocation workflows.

## [0.1.0] - 2026-05-18

### Added
- **Git-First System Registry** (`internal/registry`): Implemented `EnsureRegistry()` to bootstrap a bare Git repository and checked-out worktree at `RootDir/registry/` on startup.
- **Agnostic Database Core**: Implemented a highly concurrent SQLite connection pool (via `mattn/go-sqlite3`) abstracted via `database/sql` for future Postgres/MySQL support.
- **Dialect-Aware Migrations**: Built a lightweight startup schema migration engine.
- **Structured Logging**: Implemented structured HTTP request logging middleware in the `internal/web` package.
- **Unit Testing Foundation**: Implemented comprehensive unit tests for core modules: `internal/config`, `internal/log`, `internal/version`, and the `main` application entry point.
- **Environment Configuration**: Implemented environment variable configuration loading in `internal/config/config.go` using `godotenv`.
- **Project Identity**: Created `internal/version/version.go` to track application metadata and added standard `Makefile`.
- **Foundation Structure**: Initialized base Go project structure, standard `LICENSE`, `HEADER`, and `README.md`.
- **Cobra CLI Integration**: Migrated CLI command routing and help generation to `github.com/spf13/cobra`.
- **CLI Refactoring**: Restructured `cmd/sourcevault/main.go` into `main.go`, `root.go`, and `start.go` for cleaner separation of concerns.
- **Improved Initialization**: Refactored `cmd/sourcevault` to load configuration and initialize logging via `rootCmd.PersistentPreRunE`.
- **Branded CLI Help**: Adapted custom `lipgloss` styling into Cobra's `SetHelpFunc` template for consistent visual branding.
- **Favicon Support**: Generated a `favicon.ico` from the SourceVault logo and added a `/favicon.ico` route to the web server.
- **UI Polish**: Added a copyright footer to the `coming_soon.html` landing page template.
