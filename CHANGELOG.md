# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added a copyright footer to the `coming_soon.html` landing page template.
- Refactored `cmd/sourcevault` to load configuration and initialize logging via `rootCmd.PersistentPreRunE` so all future subcommands inherit this setup automatically.
- Migrated CLI command routing and help generation to `github.com/spf13/cobra`.
- Restructured `cmd/sourcevault/main.go` into `main.go`, `root.go`, and `start.go` for cleaner separation of concerns.
- Adapted custom `lipgloss` styling into Cobra's `SetHelpFunc` template for consistent visual branding.
- Implemented structured HTTP request logging middleware in the `internal/web` package to capture method, path, status, and duration.
- Added explicit error logging for web server template and asset loading failures.
- Implemented a wait mechanism in the `start` command to prevent immediate exit and allow for graceful shutdown.
- Implemented comprehensive unit tests for core modules: `internal/config`, `internal/log`, `internal/version`, and the `main` application entry point.
- Refactored `cmd/sourcevault/main.go` to use `io.Writer` parameters in `run` and `printUsage` for improved testability.
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
