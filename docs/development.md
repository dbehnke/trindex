# Development Setup

This document covers local development setup for Trindex.

## macOS with Colima

If you're using macOS and Docker Desktop is not installed, you can use [Colima](https://github.com/abiosoft/colima) as a lightweight alternative.

### Installing Colima

```bash
brew install colima
```

### Starting Colima

Start Colima with sufficient resources for running Postgres and integration tests:

```bash
colima start --cpu 4 --memory 8 --disk 50
```

### Setting DOCKER_HOST

Colima uses a non-default Docker socket location. Set the environment variable:

```bash
export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
```

Add this to your `~/.zshrc` or `~/.bash_profile` to make it permanent.

### Verifying Docker Works

```bash
docker ps
```

You should see an empty container list (or running containers if any).

### Running Integration Tests

With Colima running and `DOCKER_HOST` set:

```bash
task test:integration:mac
```

This will:
1. Check that Colima is running
2. Run integration tests with Testcontainers

### Troubleshooting

**Error: Cannot connect to Docker daemon**

Colima is not running. Start it with:
```bash
colima start
```

**Error: Colima status shows "Stopped"**

```bash
colima status
```

If stopped, start it with the command above.

**Integration tests timeout**

Increase Colima resources:
```bash
colima stop
colima start --cpu 4 --memory 8 --disk 50
```

**Tests fail with "Docker not available"**

Ensure `DOCKER_HOST` is set correctly:
```bash
echo $DOCKER_HOST
# Should output: unix:///Users/YOUR_USERNAME/.colima/default/docker.sock
```

**Ryuk reaper on macOS**

Testcontainers uses a container called "Ryuk" to automatically clean up test containers. With the proper Docker socket configuration, Ryuk works correctly on macOS with Colima and will automatically clean up test containers after each test.

If you experience Ryuk-related issues, ensure both environment variables are set:

```bash
export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="/var/run/docker.sock"
```

These are automatically configured when using `task test:integration:mac`.

## Linux / Docker Desktop

If you're using Linux or Docker Desktop on macOS, no special setup is needed. Docker should work out of the box.

### Running Integration Tests

```bash
task test:integration
```

## All Platforms

### Running Unit Tests Only

To skip integration tests (no Docker required):

```bash
task test:short
```

### Running All Tests

```bash
task test:all
```

This runs both unit tests and integration tests.
