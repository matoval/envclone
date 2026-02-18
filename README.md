# envclone

Containerized dev environments with sidecar services. Same environment on Linux and macOS.

## Setup

```bash
# Build envclone
go build -o envclone .

# Install prerequisites (nerdctl, rootless containerd, buildkit)
./envclone setup
```

### What `setup` installs

| Platform | Components |
|----------|-----------|
| Linux | nerdctl-full, rootless containerd, buildkit |
| macOS | Lima (via Homebrew), creates a Lima VM with containerd |

## Quick Start

```bash
# Initialize a new project
cd ~/my-project
envclone init

# Edit .devcontainer/devcontainer.json to your needs, then:
envclone up
envclone shell
```

## Commands

| Command | Description |
|---------|-------------|
| `envclone setup` | Install all prerequisites |
| `envclone init` | Create `.devcontainer/devcontainer.json` in current directory |
| `envclone up` | Build image (if Dockerfile), start containers |
| `envclone down` | Stop and remove all containers for the project |
| `envclone shell` | Open a bash shell in the dev container |
| `envclone exec <cmd>` | Run a command in the dev container |
| `envclone status` | Show running containers for the project |
| `envclone code` | Open VS Code connected to the dev container via SSH |
| `envclone ssh-config` | Print SSH config block for VS Code Remote-SSH |

## Configuration

### Using a base image

```json
{
  "name": "my-project",
  "image": "fedora:45",
  "remoteUser": "root"
}
```

### Using a Dockerfile

```json
{
  "name": "my-project",
  "build": {
    "dockerfile": "Dockerfile"
  },
  "remoteUser": "root"
}
```

`.devcontainer/Dockerfile`:

```dockerfile
FROM fedora:45
RUN dnf install -y git gcc golang && dnf clean all
```

### Workspace mount

By default, envclone mounts the current directory to `/workspace` in the container. You can customize both sides:

```json
{
  "workspaceFolder": "/home/user/projects/my-app",
  "workspaceMount": "/workspace"
}
```

- `workspaceFolder` — host path to mount (defaults to current directory if empty)
- `workspaceMount` — path inside the container (defaults to `/workspace`)

### Sidecar services

Add services that share a network namespace with your dev container. All services are reachable at `localhost` from the dev shell.

```json
{
  "name": "my-app",
  "image": "fedora:45",
  "services": [
    {
      "name": "postgres",
      "image": "postgres:16",
      "env": ["POSTGRES_PASSWORD=dev"]
    },
    {
      "name": "redis",
      "image": "redis:7"
    }
  ]
}
```

### Lifecycle commands

```json
{
  "postCreateCommand": "dnf install -y vim",
  "postStartCommand": "echo 'ready'"
}
```

### Additional mounts

Mount host paths into the container. Use `${localEnv:VAR}` to reference host environment variables:

```json
{
  "mounts": [
    "${localEnv:HOME}/.gitconfig:/root/.gitconfig",
    "${localEnv:HOME}/.ssh:/root/.ssh"
  ]
}
```

## VS Code Integration

Open VS Code connected to your dev container with a single command:

```bash
envclone up
envclone code
```

This sets up SSH in the container, injects your public key, updates `~/.ssh/config`, and launches VS Code — all in one step. Requires an SSH key in `~/.ssh/` (generate one with `ssh-keygen -t ed25519` if needed).

### Persisting VS Code extensions and auth

By default, VS Code installs extensions fresh into each container. To persist extensions and their authentication across `envclone down`/`up` cycles, mount `~/.vscode-server` from the host:

```json
{
  "mounts": [
    "${localEnv:HOME}/.vscode-server:/root/.vscode-server"
  ]
}
```

Install your extensions once — they'll be there on every subsequent container.

### Manual setup

```bash
envclone up
envclone ssh-config >> ~/.ssh/config
code --remote ssh-remote+envclone-my-project /workspace
```

## Architecture

envclone uses a shared network namespace (pause container) pattern — the same approach Kubernetes uses for pods. All containers in an environment share the same network stack, so services are reachable at `localhost`.

```
┌─────────────────────────────────────┐
│         shared network namespace    │
│  ┌──────────┐ ┌────────┐ ┌───────┐ │
│  │   dev    │ │postgres│ │ redis │ │
│  │container │ │        │ │       │ │
│  └──────────┘ └────────┘ └───────┘ │
└─────────────────────────────────────┘
       │
   -v /project:/workspace
       │
   host filesystem
```
