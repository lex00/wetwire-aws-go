# Kiro CLI Integration

Use Kiro CLI with wetwire-aws for AI-assisted infrastructure design in corporate AWS environments.

## Prerequisites

- Go 1.23+ installed
- Kiro CLI installed ([installation guide](https://kiro.dev/docs/cli/installation/))
- AWS Builder ID or GitHub/Google account (for Kiro authentication)

---

## Step 1: Install wetwire-aws

### Option A: Using Go (recommended)

```bash
go install github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest
```

### Option B: Pre-built binaries

Download from [GitHub Releases](https://github.com/lex00/wetwire-aws-go/releases):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/lex00/wetwire-aws-go/releases/latest/download/wetwire-aws-darwin-arm64
chmod +x wetwire-aws-darwin-arm64
sudo mv wetwire-aws-darwin-arm64 /usr/local/bin/wetwire-aws

# macOS (Intel)
curl -LO https://github.com/lex00/wetwire-aws-go/releases/latest/download/wetwire-aws-darwin-amd64
chmod +x wetwire-aws-darwin-amd64
sudo mv wetwire-aws-darwin-amd64 /usr/local/bin/wetwire-aws

# Linux (x86-64)
curl -LO https://github.com/lex00/wetwire-aws-go/releases/latest/download/wetwire-aws-linux-amd64
chmod +x wetwire-aws-linux-amd64
sudo mv wetwire-aws-linux-amd64 /usr/local/bin/wetwire-aws
```

### Verify installation

```bash
wetwire-aws --version
```

---

## Step 2: Install Kiro CLI

```bash
# Install Kiro CLI
curl -fsSL https://cli.kiro.dev/install | bash

# Verify installation
kiro-cli --version

# Sign in (opens browser)
kiro-cli login
```

---

## Step 3: Configure Kiro for wetwire-aws

Run the design command with `--provider kiro` to auto-configure:

```bash
# Create a project directory
mkdir my-infra && cd my-infra

# Initialize Go module
go mod init my-infra

# Run design with Kiro provider (auto-installs configs on first run)
wetwire-aws design --provider kiro "Create an S3 bucket"
```

This automatically installs:

| File | Purpose |
|------|---------|
| `~/.kiro/agents/wetwire-aws-runner.json` | Kiro agent configuration |
| `.kiro/mcp.json` | Project MCP server configuration |

### Manual configuration (optional)

The MCP server is provided as a subcommand `wetwire-aws mcp`. If you prefer to configure manually:

**~/.kiro/agents/wetwire-aws-runner.json:**
```json
{
  "name": "wetwire-aws-runner",
  "description": "Infrastructure code generator using wetwire-aws",
  "prompt": "You are an infrastructure design assistant...",
  "model": "claude-sonnet-4",
  "mcpServers": {
    "wetwire": {
      "command": "wetwire-aws",
      "args": ["mcp"],
      "cwd": "/path/to/your/project"
    }
  },
  "tools": ["*"]
}
```

**.kiro/mcp.json:**
```json
{
  "mcpServers": {
    "wetwire": {
      "command": "wetwire-aws",
      "args": ["mcp"],
      "cwd": "/path/to/your/project"
    }
  }
}
```

> **Note:** The `cwd` field ensures MCP tools resolve paths correctly in your project directory. When using `wetwire-aws design --provider kiro`, this is configured automatically.

---

## Step 4: Run Kiro with wetwire design

### Using the wetwire-aws CLI

```bash
# Start Kiro design session
wetwire-aws design --provider kiro "Create a serverless API with Lambda and DynamoDB"
```

This launches Kiro CLI with the wetwire-aws-runner agent and your prompt.

### Using Kiro CLI directly

```bash
# Start chat with wetwire-aws-runner agent
kiro-cli chat --agent wetwire-aws-runner

# Or with an initial prompt
kiro-cli chat --agent wetwire-aws-runner "Create an S3 bucket with versioning"
```

---

## Available MCP Tools

The wetwire-aws MCP server exposes three tools to Kiro:

| Tool | Description | Example |
|------|-------------|---------|
| `wetwire_init` | Initialize a new project | `wetwire_init(path="./myapp")` |
| `wetwire_lint` | Lint code for issues | `wetwire_lint(path="./infra/...")` |
| `wetwire_build` | Generate CloudFormation template | `wetwire_build(path="./infra/...", format="json")` |

---

## Example Session

```
$ wetwire-aws design --provider kiro "Create an S3 bucket with versioning and encryption"

Installed Kiro agent config: ~/.kiro/agents/wetwire-aws-runner.json
Installed project MCP config: .kiro/mcp.json
Starting Kiro CLI design session...

> I'll help you create an S3 bucket with versioning and encryption enabled.

Let me initialize the project and create the infrastructure code.

[Calling wetwire_init...]
[Calling wetwire_lint...]
[Calling wetwire_build...]

I've created the following files:
- infra/storage.go

The S3 bucket includes:
- Versioning enabled
- Server-side encryption with AES-256
- Public access blocked

Would you like me to add any additional configurations?
```

---

## Workflow

The Kiro agent follows this workflow:

1. **Explore** - Understand your requirements
2. **Plan** - Design the infrastructure architecture
3. **Implement** - Generate Go code using wetwire-aws patterns
4. **Lint** - Run `wetwire_lint` to check for issues
5. **Build** - Run `wetwire_build` to generate CloudFormation template

---

## Deploying Generated Templates

After Kiro generates your infrastructure code:

```bash
# Build the CloudFormation template
wetwire-aws build ./infra > template.json

# Deploy with AWS CLI
aws cloudformation deploy \
  --template-file template.json \
  --stack-name my-stack \
  --capabilities CAPABILITY_IAM

# Or use SAM CLI for serverless
sam deploy --template-file template.json --stack-name my-stack --guided
```

---

## Troubleshooting

### MCP server not found

```
Mcp error: -32002: No such file or directory
```

**Solution:** Ensure `wetwire-aws` is in your PATH:

```bash
which wetwire-aws

# If not found, add to PATH or reinstall
go install github.com/lex00/wetwire-aws-go/cmd/wetwire-aws@latest
```

### Kiro CLI not found

```
kiro-cli not found in PATH
```

**Solution:** Install Kiro CLI:

```bash
curl -fsSL https://cli.kiro.dev/install | bash
```

### Authentication issues

```
Error: Not authenticated
```

**Solution:** Sign in to Kiro:

```bash
kiro-cli login
```

---

## Known Limitations

### Automated Testing

When using `wetwire-aws test --provider kiro`, tests run in non-interactive mode (`--no-interactive`). This means:

- The agent runs autonomously without waiting for user input
- Persona simulation is limited - all personas behave similarly
- The agent won't ask clarifying questions

For true persona simulation with multi-turn conversations, use the Anthropic provider:

```bash
wetwire-aws test --provider anthropic --persona expert "Create an S3 bucket"
```

### Interactive Design Mode

Interactive design mode (`wetwire-aws design --provider kiro`) works fully as expected:

- Real-time conversation with the agent
- Agent can ask clarifying questions
- Lint loop executes as specified in the agent prompt

---

## See Also

- [CLI Reference](CLI.md) - Full wetwire-aws CLI documentation
- [Quick Start](QUICK_START.md) - Getting started with wetwire-aws
- [Kiro CLI Installation](https://kiro.dev/docs/cli/installation/) - Official installation guide
- [Kiro CLI Docs](https://kiro.dev/docs/cli/) - Official Kiro documentation
