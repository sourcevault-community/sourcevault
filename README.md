# SourceVault
The Federated Code Collaboration Platform

SourceVault is an open-source decentralized Git hosting and collaboration platform designed for self-hosting, data sovereignty, and seamless federation across instances.

## What is SourceVault?

SourceVault combines the promise of future federated collaboration with a security-first, Git-first engine.

- **Federated collaboration**: Planned support for a custom federation API with ActivityPub compatibility.
- **Open source**: 100% open source with no vendor lock-in.
- **Data sovereignty**: Your code, your data, your rules. No tracking, no surprises.
- **Secure Git access**: Built around OpenSSH and native SSH Certificate Authority logic.

## SourceVault Architecture

This repository implements SourceVault as a single, self-contained Go binary.

- A unified Go application with built-in web UI, Git hosting, and federation support.
- Designed for easy self-hosting, strong security, and a Git-first user experience.
- Includes repository management, authentication, access control, and metadata storage.

## Features

- Planned federated Git collaboration via a custom federation API with ActivityPub compatibility.
- Self-hosted deployment with a single Go binary.
- Open source and vendor-neutral.
- Data sovereignty: your code, your data, your rules.
- Built-in Git server functionality with secure repository access.

## Status

🚀 Coming Soon

SourceVault is under active development and aims to provide a better way to host and collaborate on Git repositories with a federated, open-source foundation.

## Learn More

- Homepage: https://sourcevault.dev
- About: https://sourcevault.dev/about.html
- Contribute: https://sourcevault.dev/contribute.html

## Configuration

SourceVault is configured via environment variables. For local development, you can use a `sourcevault.env` file in the project root. A template is provided in `sourcevault.env.sample`.

| Variable | Description | Default |
|----------|-------------|---------|
| `SOURCEVAULT_BASE_DIR` | Root directory for data and logs | `/home/sourcevault` |
| `SOURCEVAULT_LOG_FILE` | Path to log file (relative to BaseDir) | `(stdout)` |
| `SOURCEVAULT_LOG_LEVEL` | Log verbosity level: ERROR, WARN, INFO, DEBUG | `INFO` |
| `SOURCEVAULT_DEBUG`    | Enable verbose diagnostic output | `true` |
| `SOURCEVAULT_WEB_ENABLED` | Enable the administrative web server | `false` |
| `SOURCEVAULT_WEB_HOST` | Web server bind address | `127.0.0.1` |
| `SOURCEVAULT_WEB_PORT` | Web server bind port | `8080` |

## Development

### Prerequisites
- Go 1.26 or later

### Building and Running
```bash
# Build the binary
make build

# Run the application
make run

# Run tests
make test
```

## License

This repository is licensed under the terms of the included `LICENSE` file.
