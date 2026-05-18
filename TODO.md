# SourceVault TODOs

This file tracks planned features, refactoring goals, and technical debt.

## Features
- [ ] **Auto-Update Mechanism**: Implement a self-update feature using `go-selfupdate`.
  - Add an `AutoUpdate` toggle to `config.go`.
  - Create a background goroutine to poll for new GitHub releases.
  - Expose a manual `sourcevault update` command via the Cobra CLI.
  - Implement graceful restart instructions/logic upon successful binary replacement.

## Internal Systems
- [ ] **SSH Server**: Implement the internal SSH server to handle Git clone/push/pull requests over port 2222.
- [ ] **Database Setup**: Implement the SQLite repository logic (`internal/db` package with multiple domain files like `users.go`, `repositories.go`, etc.).
