# jctl - Jenkins Control Tool

> ✨ **Built entirely with [Kiro](https://kiro.dev)** - An AI-powered development environment

A powerful command-line interface for interacting with Jenkins CI/CD servers. Manage pipelines, view builds, access logs, and trigger builds directly from your terminal.

## Features

- 🚀 **Pipeline Management** - List and browse Jenkins pipelines and folders
- 📊 **Build Information** - View build history with status and timestamps
- 📝 **Log Streaming** - Access build logs with real-time streaming support
- 💬 **Interactive Input Handling** - Automatically detect and respond to pipeline input steps
- ⚡ **Build Triggering** - Start builds with parameters and follow progress
- 🔐 **Secure Authentication** - Support for API tokens and OAuth
- 👤 **Profile Management** - Configure multiple Jenkins instances and switch between them
- 🎨 **Multiple Output Formats** - Text, JSON, and YAML output options

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap robyt96/tap
brew install jctl
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/robyt96/jctl/releases):

- **Linux**: `jctl_<version>_linux_amd64.tar.gz` or `jctl_<version>_linux_arm64.tar.gz`
- **macOS**: `jctl_<version>_darwin_amd64.tar.gz` or `jctl_<version>_darwin_arm64.tar.gz`
- **Windows**: `jctl_<version>_windows_amd64.zip` or `jctl_<version>_windows_arm64.zip`

Extract and move the binary to your PATH:

```bash
# Linux/macOS
tar -xzf jctl_<version>_<os>_<arch>.tar.gz
sudo mv jctl /usr/local/bin/

# Windows
# Extract the zip and add jctl.exe to your PATH
```

### Build from Source

Requirements:
- Go 1.24.4 or later

```bash
git clone https://github.com/robyt96/jctl.git
cd jctl
go build -o jctl ./cmd/jctl
sudo mv jctl /usr/local/bin/
```

## Quick Start

### 1. Configure Your Jenkins Instance

Create a configuration file at `~/.jctl/config.yaml`:

```yaml
default_profile: production

profiles:
  production:
    jenkins:
      url: https://jenkins.example.com
      timeout: 30s
    auth:
      method: token
    output:
      format: text
      color: true

  staging:
    jenkins:
      url: https://jenkins-staging.example.com
      timeout: 30s
    auth:
      method: token
```

### 2. Authenticate

```bash
# Authenticate with API token (recommended)
jctl auth login

# Or specify profile
jctl auth login --profile staging
```

You'll be prompted to enter your Jenkins username and API token. Generate an API token from Jenkins at: `https://your-jenkins-url/user/<username>/configure`

### 3. Start Using jctl

```bash
# List all pipelines
jctl pipelines list

# List builds for a pipeline
jctl builds list my-pipeline

# View build logs
jctl logs my-pipeline 42

# Trigger a build
jctl trigger my-pipeline --param ENV=staging
```

## Usage

### Global Flags

These flags can be used with any command:

```bash
--profile string       Profile to use (default: default profile from config)
--jenkins-url string   Jenkins server URL (overrides config)
--config string        Config file path (default: ~/.jctl/config.yaml)
--output string        Output format: text, json, yaml (default: text)
--verbose              Enable verbose logging
--timeout duration     Request timeout duration
```

### Commands

#### Pipelines

List all available pipelines:

```bash
# List pipelines in root
jctl pipelines list

# List pipelines in a folder
jctl pipelines list my-folder

# Output as JSON
jctl pipelines list --output json
```

#### Builds

List builds for a specific pipeline:

```bash
# List builds
jctl builds list my-pipeline

# List builds for pipeline in folder
jctl builds list folder/subfolder/pipeline

# Output as JSON
jctl builds list my-pipeline --output json
```

#### Logs

View console logs for a build:

```bash
# View complete log
jctl logs my-pipeline 42

# Follow log output (stream in real-time)
jctl logs my-pipeline 42 --follow
```

**Interactive Input Handling**

When following logs with `--follow`, jctl automatically detects when a build is waiting for user input (such as approval steps) and prompts you interactively:

```bash
# Follow logs - will prompt for input when needed
jctl logs my-pipeline 42 --follow

# Example interaction:
# === Input Required ===
# Are you sure to proceed with this plan?
#
# Proceed? (y/n): y
# Input submitted. Continuing...
```

The tool supports:
- **Simple approvals**: Yes/No prompts for approval steps
- **Parameterized inputs**: Prompts for each required parameter with descriptions and default values
- **Automatic detection**: Checks for pending inputs every 2 seconds during log streaming
- **Graceful handling**: Allows aborting the build by typing 'n' or 'abort'

#### Trigger

Trigger a new build:

```bash
# Trigger without parameters
jctl trigger my-pipeline

# Trigger with parameters
jctl trigger my-pipeline --param ENV=staging --param VERSION=1.2.3

# Trigger and follow logs (with automatic input handling)
jctl trigger my-pipeline --param BRANCH=main --follow
```

When using `--follow` with the trigger command, jctl will:
1. Wait for the build to start
2. Stream logs in real-time
3. Automatically detect and prompt for any input steps
4. Display the final build status

#### Authentication

Manage authentication credentials:

```bash
# Login with API token (interactive)
jctl auth login

# Login with specific method
jctl auth login --method token

# Login with OAuth
jctl auth login --method oauth --client-id <id> --client-secret <secret>
```

#### Profiles

Manage multiple Jenkins instances:

```bash
# List all profiles
jctl profile list

# Show current profile details
jctl profile show

# Show specific profile
jctl profile show production

# Set default profile
jctl profile set-default staging
```

## Configuration

### Configuration File

The configuration file is located at `~/.jctl/config.yaml` by default. You can specify a different location with the `--config` flag.

Example configuration:

```yaml
default_profile: production

profiles:
  production:
    jenkins:
      url: https://jenkins.example.com
      timeout: 30s
    auth:
      method: token
      token_file: ~/.jctl/credentials.json
    output:
      format: text
      color: true
    defaults:
      pipeline: main-pipeline

  staging:
    jenkins:
      url: https://jenkins-staging.example.com
      timeout: 60s
    auth:
      method: oauth
    output:
      format: json
      color: false
```

### Environment Variables

You can override configuration values using environment variables:

- `JCTL_JENKINS_URL` - Jenkins server URL
- `JCTL_PROFILE` - Profile to use
- `JCTL_OUTPUT_FORMAT` - Output format (text, json, yaml)
- `JCTL_TIMEOUT` - Request timeout duration

### Configuration Precedence

Configuration values are applied in the following order (highest to lowest priority):

1. Command-line flags
2. Environment variables
3. Profile configuration
4. Default values

## Authentication

jctl supports two authentication methods:

### API Token (Recommended)

1. Generate an API token in Jenkins:
   - Navigate to `https://your-jenkins-url/user/<username>/configure`
   - Click "Add new Token" under API Token section
   - Copy the generated token

2. Login with jctl:
   ```bash
   jctl auth login --method token
   ```

3. Enter your username and token when prompted

### OAuth (not tested)

For Jenkins instances with OAuth configured:

```bash
jctl auth login --method oauth --client-id <id> --client-secret <secret>
```

This will open your browser for authentication.

## Examples

### Working with Folders

```bash
# List pipelines in a folder
jctl pipelines list DevOps/Production

# List builds for pipeline in folder
jctl builds list DevOps/Production/backend-service

# View logs
jctl logs DevOps/Production/backend-service 123
```

### Parameterized Builds

```bash
# Trigger with multiple parameters
jctl trigger my-pipeline \
  --param ENVIRONMENT=production \
  --param VERSION=2.1.0 \
  --param DEPLOY_REGION=us-east-1

# Trigger and follow logs
jctl trigger my-pipeline --param BRANCH=feature/new-feature --follow
```

### Interactive Input Handling

When a build requires manual approval or input, jctl will automatically prompt you:

```bash
# Trigger a deployment that requires approval
jctl trigger deploy-to-production --param VERSION=2.0.0 --follow

# Output:
# ✓ Build triggered successfully for pipeline: deploy-to-production
# Queue ID: 123
# Waiting for build to start...
# 
# ✓ Build #45 started
# Streaming logs...
# 
# [build output...]
# 
# === Input Required ===
# Are you sure you want to deploy version 2.0.0 to production?
# 
# Proceed? (y/n): y
# Input submitted. Continuing...
# 
# [continued build output...]
# 
# ✓ Build #45 completed with status: SUCCESS
```

For parameterized inputs:

```bash
# Build with multiple input parameters
jctl logs infrastructure-apply 67 --follow

# Output:
# === Input Required ===
# Please confirm the infrastructure changes
# 
# Please provide values for the following parameters:
# 
# REGION: Target AWS region
#   (default: us-east-1)
# Value: us-west-2
# 
# INSTANCE_COUNT: Number of instances to create
#   (default: 3)
# Value: 5
# 
# Submit these values? (y/n/abort): y
# Input submitted. Continuing...
```

### Multiple Profiles

```bash
# Use staging profile
jctl --profile staging pipelines list

# Trigger build on production
jctl --profile production trigger critical-pipeline --param HOTFIX=true

# Switch default profile
jctl profile set-default production
```

### JSON Output for Scripting

```bash
# Get pipeline list as JSON
jctl pipelines list --output json | jq '.[] | select(.name == "my-pipeline")'

# Get build status
jctl builds list my-pipeline --output json | jq '.[0].result'

# Parse build information
BUILD_STATUS=$(jctl builds list my-pipeline --output json | jq -r '.[0].result')
if [ "$BUILD_STATUS" = "SUCCESS" ]; then
  echo "Last build succeeded"
fi
```

## Troubleshooting

### Connection Issues

```bash
# Enable verbose logging
jctl --verbose pipelines list

# Test with explicit URL
jctl --jenkins-url https://jenkins.example.com pipelines list
```

### Authentication Errors

```bash
# Re-authenticate
jctl auth login

# Check profile configuration
jctl profile show

# Verify credentials file
cat ~/.jctl/credentials.json
```

### Pipeline Not Found

```bash
# List all available pipelines
jctl pipelines list

# Check folder structure
jctl pipelines list folder-name
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/robyt96/jctl/issues)
- **Documentation**: [Wiki](https://github.com/robyt96/jctl/wiki)

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [GoReleaser](https://goreleaser.com/) - Release automation
