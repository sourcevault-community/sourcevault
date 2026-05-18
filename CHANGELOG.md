# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Improved code documentation and readability in `internal/config/config.go`, `internal/version/version.go`, and `cmd/sourcevault/main.go` by adding comprehensive inline comments to structs, functions, and application lifecycle logic.

### Fixed
- Fixed invalid backslash line continuation in `cmd/sourcevault/main.go` and refactored the `slog.Info` call for better readability.

### Added
- Implemented environment variable configuration loading in `internal/config/config.go` using `godotenv` to parse `.env` files.
- Added comprehensive inline documentation explaining the configuration loading and override flow.
- Documented `SOURCEVAULT_SSH_ENABLED`, `SOURCEVAULT_SSH_HOST`, and `SOURCEVAULT_SSH_PORT` in `README.md` and `sourcevault.env.sample` to match new `SshConfig` structures.
- Added inline GoDoc comments to `cmd/sourcevault/main.go` for the configuration output block.
- Embedded `WebConfig` and `SshConfig` within the main `Config` structure for hierarchical configuration management.
- Created `internal/config/config.go` with `Config`, `WebConfig`, and `SshConfig` structures.
- Added comprehensive GoDoc comments to all configuration structs in `internal/config`.
- Integrated `internal/version` package into `cmd/sourcevault/main.go` to display application build metadata (version, git commit, build date, architecture) on startup.
- Fully implemented `printUsage` in `cmd/sourcevault/main.go` using `lipgloss` styling and `strings.Builder` to output a formatted CLI help menu.
- Defined the initial list of `commands` (`help`, `start`) for the CLI interface.
- Added `lipgloss` dependency for rich terminal UI formatting.
- Defined initial UI styling tokens (`titleStyle`, `commandStyle`, etc.) in `cmd/sourcevault/main.go`.
- Implemented `printUsage` function with accompanying GoDoc comments.
- Added comprehensive GoDoc comments to `cmd/sourcevault/main.go` functions (`main` and `run`).
- Created `internal/version/version.go` to track application metadata (`AppName`, `AppVersion`, `GitCommit`, `GitBranch`, `BuildDate`, `Architecture`).
- Added standard `Makefile` with targets for build, install, uninstall, run, clean, and test. Injects version info via `LDFLAGS`.
- Added `.ai-rules.md` and `.cursorrules` to strictly enforce coding standards, changelog maintenance, commit hygiene, and pre-commit test execution for AI assistants.
- Initialized base Go project structure and module (`sourcevault`).
- Added initial `cmd/sourcevault/main.go` entrypoint with standard license `HEADER`.
- Added standard `LICENSE`, `HEADER`, and `README.md` imported from previous PRJ.
