# YaSwag MCP Server

Model Context Protocol (MCP) server for AI-powered OpenAPI specification interaction. This enables AI assistants like Claude to semantically search, explore, and analyze your API specifications.

## Features

- **Semantic Search** - Natural language search across endpoints, schemas, and documentation
- **Schema Explorer** - Navigate and understand complex API schemas
- **Validation** - Validate OpenAPI specifications
- **Security Analysis** - Analyze authentication and authorization patterns
- **Example Generation** - Generate request/response examples from schemas

## Quick Start

### 1. Build YaSwag

```bash
go install github.com/fathurrohman26/yaswag/cmd/yaswag@latest
```

Or build from source:

```bash
make build
```

### 2. Configure Claude Code

Add to your `.claude/settings.json`:

```json
{
  "mcpServers": {
    "yaswag": {
      "command": "yaswag",
      "args": ["mcp", "./openapi.json"]
    }
  }
}
```

Or with multiple spec files:

```json
{
  "mcpServers": {
    "yaswag": {
      "command": "yaswag",
      "args": ["mcp", "./api/v1/openapi.yaml", "./api/v2/openapi.yaml"]
    }
  }
}
```

### 3. Use in Claude Code

Once configured, Claude can use tools like:

```md
"Search for user authentication endpoints"
"Show me the User schema"
"List all POST endpoints"
"Validate my OpenAPI spec"
```

## Available Tools

### search_endpoints

Search for API endpoints using natural language or keywords.

**Parameters:**

- `query` (required): Search query (e.g., "user authentication", "create order")
- `method`: Filter by HTTP method (GET, POST, PUT, DELETE, PATCH)
- `tag`: Filter by tag name
- `limit`: Maximum results to return (default: 10)

**Example:**

```md
Search for endpoints related to "user login"
```

### list_endpoints

List all API endpoints with optional filtering.

**Parameters:**

- `method`: Filter by HTTP method
- `tag`: Filter by tag name
- `path_pattern`: Filter by path pattern (supports wildcards)

**Example:**

```md
List all POST endpoints tagged with "users"
```

### get_endpoint

Get detailed information about a specific endpoint.

**Parameters:**

- `path` (required): The endpoint path (e.g., /users/{id})
- `method` (required): The HTTP method

**Example:**

```md
Get details for GET /users/{id}
```

### search_schemas

Search for schema definitions by name, description, or properties.

**Parameters:**

- `query` (required): Search query for schema name or description
- `has_property`: Filter schemas that have a specific property

**Example:**

```md
Search schemas with "email" property
```

### get_schema

Get detailed schema definition by name.

**Parameters:**

- `name` (required): The schema name (e.g., "User", "Order")
- `resolve_refs`: Whether to resolve $ref references inline

**Example:**

```md
Get the User schema definition
```

### validate_spec

Validate an OpenAPI specification.

**Parameters:**

- `spec_path`: Path to the spec file (optional, uses loaded specs if not provided)

**Example:**

```md
Validate my OpenAPI specification
```

### get_spec_info

Get general information about the OpenAPI specification.

**Returns:**

- OpenAPI version
- API title and version
- Servers
- Tags
- Endpoint and schema counts

### generate_example

Generate example request or response data for an endpoint.

**Parameters:**

- `path` (required): The endpoint path
- `method` (required): The HTTP method
- `type`: Generate "request" or "response" example (default: response)
- `status_code`: Response status code (default: 200)

**Example:**

```md
Generate a request example for POST /users
```

### find_related

Find endpoints related to a given endpoint (same resource, similar operations).

**Parameters:**

- `path` (required): The endpoint path to find related endpoints for

**Example:**

```md
Find endpoints related to /users/{id}
```

### list_tags

List all tags with their descriptions and endpoint counts.

### analyze_security

Analyze security requirements across the API.

**Returns:**

- Global security settings
- Security schemes defined
- Protected vs unprotected endpoint counts
- Endpoints grouped by security scheme

## Protocol Details

The MCP server uses JSON-RPC 2.0 over stdio for communication. It implements:

- **Protocol Version**: 2024-11-05
- **Transport**: stdio (stdin/stdout)
- **Capabilities**: Tools, Resources

### Supported Methods

| Method | Description |
|--------|-------------|
| `initialize` | Initialize the MCP session |
| `tools/list` | List available tools |
| `tools/call` | Call a tool with arguments |
| `resources/list` | List available resources (spec files) |
| `resources/read` | Read a resource content |
| `ping` | Health check |

## Programmatic Usage

```go
package main

import (
    "github.com/fathurrohman26/yaswag/pkg/mcp"
)

func main() {
    // Create server with spec files
    server := mcp.NewServer([]string{"./openapi.json"})

    // Run the server (blocks, reads from stdin)
    if err := server.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Architecture

```md
┌─────────────────┐      JSON-RPC       ┌─────────────────┐
│   AI Assistant  │ ◄─────────────────► │   MCP Server    │
│  (Claude Code)  │      over stdio     │    (yaswag)     │
└─────────────────┘                     └────────┬────────┘
                                                 │
                                                 ▼
                                        ┌─────────────────┐
                                        │  OpenAPI Specs  │
                                        │  (JSON/YAML)    │
                                        └─────────────────┘
```

## Semantic Search

The semantic search implementation uses a weighted scoring system:

| Match Location | Weight |
|---------------|--------|
| Path | 3.0 |
| Operation ID | 2.5 |
| Summary | 2.0 |
| Tags | 1.5 |
| Description | 1.0 |

Results are sorted by relevance score, with exact matches scoring highest.

## Tips for Best Results

1. **Be specific**: "user authentication login endpoint" works better than "auth"
2. **Use filters**: Combine search with method/tag filters for precise results
3. **Check schemas**: Use `get_schema` to understand request/response structures
4. **Validate first**: Run `validate_spec` to ensure your spec is valid before searching
