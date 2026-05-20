# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **CA Bootstrap & Restoration Logic** (`internal/crypto`): Implemented `EnsureCA()` to automatically manage the application's Certificate Authority during startup.
    - If local CA files are missing, it attempts to restore the currently active CA (including encrypted private key) from the system registry.
    - If no CA exists in either the registry or on disk, it force-generates a new CA and registers it as the active authority.
    - Supports automatic unsealing on startup if `SOURCEVAULT_CA_PASSPHRASE` is provided in the environment.
- **Registry Active CA Tracking** (`internal/registry`): Updated the system registry to securely store encrypted CA private keys and track the authoritative CA via `ActiveCA.yaml`. This ensures consistent CA state across multi-node or restored environments.
- **SSH Certificate Signing** (`cmd/sourcevault ca sign`): Implemented the ability to sign SSH public keys with the unsealed local CA. Supports user and host certificates, customizable principals, and validity periods. Automatically outputs the signed certificate to `[key]-cert.pub`.
- **Crypto & Registry Unit Tests**: Developed a comprehensive test suite for `internal/crypto` and `internal/registry`.
    - `internal/crypto`: Verified Ed25519/RSA CA key generation, in-memory unsealing/sealing, certificate signing logic, and OpenSSH-compatible KRL production.
    - `internal/registry`: Implemented integration tests with a temporary Git repository to verify CA metadata persistence and revocation workflows.
- **RPC Bridge for CLI-Server Communication** (`internal/rpc`): Implemented a Unix Domain Socket RPC server to allow the CLI to communicate with the running SourceVault daemon.
- **Enhanced CA CLI**: Updated `ca status`, `ca unseal`, `ca seal`, and `ca sign` subcommands to prefer RPC communication with the active server, enabling live state management.

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
