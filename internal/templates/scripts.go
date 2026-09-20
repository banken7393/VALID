package templates

// ScriptSeedFiles maps paths under .valid/scripts/ to seed contents.
func ScriptSeedFiles() map[string]string {
	return map[string]string{
		"README.md":               scriptsREADME,
		"example_hello/meta.json": exampleHelloMeta,
		"example_hello/run.sh":    exampleHelloRun,
	}
}

const scriptsREADME = `# Project scripts (MCP tools)

Each subdirectory with a ` + "`meta.json`" + ` becomes an MCP tool node on ` + "`valid mcp`" + `.
The folder name is the tool name the model can call at will (or when you ask).

## Add a script

1. Create ` + "`.valid/scripts/<tool_id>/`" + ` (` + "`tool_id`" + `: lowercase letters, digits, underscore; start with a letter).
2. Add ` + "`meta.json`" + `:

` + "```json" + `
{
  "description": "What the AI should know — when to run this and what it does",
  "command": ["bash", "run.sh"],
  "timeout_sec": 120,
  "cwd": "script",
  "enabled": true
}
` + "```" + `

3. Add your executable (` + "`run.sh`" + `, binary, etc.).
4. Reconnect MCP if the client caches ` + "`tools/list`" + ` (VALID reloads scripts on every list/call).

## Rules

- ` + "`description`" + ` is the MCP tool description (model-facing). Write it so the AI knows when to use the tool.
- ` + "`command`" + ` is argv only (no free-form shell string). Relative binaries resolve from the script folder.
- ` + "`cwd`" + `: ` + "`script`" + ` (default) or ` + "`repo`" + `.
- Reserved names cannot be used: ` + "`search_knowledge`" + `, ` + "`list_documents`" + `, ` + "`get_document`" + `, ` + "`upsert_document`" + `.
- Scripts run on **stdio MCP** by default. HTTP MCP is loopback + token and disables scripts unless ` + "`--allow-scripts-http`" + `.
- Set ` + "`\"enabled\": false`" + ` to keep a script on disk without exposing it.
- Optional call argument ` + "`args`" + `: string array appended to ` + "`command`" + `.
- Timeout is capped at 15 minutes.

Example tool already seeded: ` + "`example_hello`" + `.
`

const exampleHelloMeta = `{
  "description": "Smoke-test project script MCP node. Prints a hello line; optional args[0] is echoed.",
  "command": ["bash", "run.sh"],
  "timeout_sec": 30,
  "cwd": "script",
  "enabled": true
}
`

const exampleHelloRun = `#!/usr/bin/env bash
# VALID example project script — safe to delete once you add real scripts.
set -euo pipefail
msg="hello-from-valid-scripts"
if [[ $# -gt 0 ]]; then
  msg="${msg}:$*"
fi
echo "${msg}"
`
