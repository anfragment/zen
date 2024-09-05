# Project structure
This document describes the file structure of the project.
- `.github`: Contains GitHub Actions workflows and issue templates.
- `assets`: Contains assets used in README.md and other documents (no source files).
- `build`: Contains build configuration and artifacts. Read more in [build/README.md](../../build/README.md).
- `docs/internal`: Contains internal documentation for contributors.
- `frontend`: Contains the frontend code. Read more in [frontend/README.md](../../frontend/README.md).
- `internal`: Contains the backend Go packages.
- `proxy`: Contains proxy exclusions for backward compatibility purposes. Read more in [proxy/README.md](../../proxy/README.md).
- `scripts`: Node.js scripts for manifest file upload. Might be used for other purposes in the future.
- `tasks`: Contains platform-specific build-related [taskfiles](https://taskfile.dev).
- `main.go`: The main entry point of the application.
- `golangci.yml`: Configuration file for [golangci-lint](https://golangci-lint.run).
- `Taskfile.yml`: Main [Taskfile](https://taskfile.dev) for common development tasks.
- `wails.json`: [Wails](https://wails.io) configuration file.
