# YaSwag - Yet Another Swagger/OpenAPI Tool For Go

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/fathurrohman26/yaswag)
[![Go Report Card](https://goreportcard.com/badge/github.com/fathurrohman26/yaswag)](https://goreportcard.com/report/github.com/fathurrohman26/yaswag)
[![GitHub license](https://img.shields.io/github/license/fathurrohman26/yaswag)](https://github.com/fathurrohman26/yaswag/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/fathurrohman26/yaswag?style=flat-square)](https://github.com/fathurrohman26/yaswag/releases/latest)

## Overview

YaSwag is a tool for generating OpenAPI 3.x specifications from Go source code using custom annotations.

[![YaSwag Preview](./docs/screenshot.png)](./docs/screenshot.png)

> **Important:** YaSwag uses its own eccentric annotation syntax that is **NOT compatible** with [swag](https://github.com/swaggo/swag) or other OpenAPI generation tools. YaSwag is designed to be simple, readable, and easy to use with a unique syntax.

## Features

- Generate OpenAPI 3.x specifications (YAML/JSON) from Go source code with custom annotations.
- Built-in Swagger UI for serving and visualizing OpenAPI documentation.
- Built-in Swagger Editor for creating and editing OpenAPI specifications.
- MCP (Model Context Protocol) server for AI assistant integration with semantic search.
- Security audit for analyzing API specifications for security issues.
- Command-line interface (CLI) for generating, validating, formatting, serving, editing, and auditing OpenAPI specs.
- Support for API-level metadata, operations, parameters, request bodies, responses, security schemes, and data models.
- Automatic schema inference from Go struct tags (json tags) with optional `!field` overrides.
- Easy-to-read and write annotation syntax with `!` prefix.

### Supported OpenAPI Versions

YaSwag only supports **OpenAPI 3.x** specifications:

- OpenAPI 3.0.x
- OpenAPI 3.1.x
- OpenAPI 3.2.x

**Note:** Swagger 2.0 is **not supported**. If you have Swagger 2.0 specifications, please migrate them to OpenAPI 3.x before using YaSwag.

**Swagger UI Compatibility:** When serving OpenAPI 3.2.x specifications via Swagger UI, YaSwag automatically patches the version to 3.1.x since Swagger UI does not yet support rendering OpenAPI 3.2.x specifications.

## Installation (CLI)

YaSwag is built using Go v1.25.5 or higher. To install the YaSwag CLI tool, run the following command:

```bash
go install github.com/fathurrohman26/yaswag/cmd/yaswag@latest
```

Or you can build it from source:

```bash
git clone https://github.com/fathurrohman26/yaswag.git
cd yaswag
make build

# The binary will be located in the 'bin' directory
./bin/yaswag version
```

## Usage (CLI)

After installing the YaSwag CLI tool, you can use it to generate Swagger documentation for your Go projects. Here are some common commands:

```bash
yaswag generate - Generates OpenAPI documentation for your Go project.
yaswag validate - Validates your OpenAPI specification file.
yaswag format   - Formats your OpenAPI specification file.
yaswag serve    - Serves the OpenAPI documentation via Swagger UI.
yaswag editor   - Launch Swagger Editor for creating/editing specifications.
yaswag mcp      - Start MCP server for AI assistant integration.
yaswag audit    - Perform security audit on OpenAPI specification.
yaswag help     - Displays help information about YaSwag commands.
yaswag version  - Displays the current version of YaSwag.
```

### Generate

```bash
# generate (indentation is 2 spaces by default)
yaswag generate --source ./path/to/your/project --format <json|yaml> --output ./swagger.<json|yaml>

# pretty format (with indentation with N spaces, default to 4)
yaswag generate --source ./path/to/your/project --format <json|yaml> --output ./swagger.<json|yaml> --pretty=6

# output to stdout
yaswag generate --source ./path/to/your/project --format yaml
```

### Validate

```bash
# validate an existing OpenAPI specification (output to stdout)
yaswag validate --input ./swagger.<json|yaml>

# validate from stdin (pipe from generate)
yaswag generate --source ./path/to/your/project | yaswag validate
```

### Format

```bash
# format an existing OpenAPI specification (output to stdout)
yaswag format --input ./swagger.<json|yaml> --pretty=2

# format and save to file
yaswag format --input ./swagger.<json|yaml> --output ./swagger-formatted.<json|yaml> --pretty=2

# format from stdin and convert to JSON
yaswag generate --source ./path/to/your/project | yaswag format --format json --pretty 2
```

### Serve (Swagger UI)

```bash
# serve OpenAPI specification with Swagger UI
yaswag serve --input ./swagger.<json|yaml>

# serve OpenAPI specification with Swagger UI on custom port
yaswag serve --input ./swagger.<json|yaml> --port 8080

# serve OpenAPI specification from external URL with Swagger UI
yaswag serve --input https://example.com/path/to/swagger.yaml --port 8080

# serve OpenAPI specification from stdin (pipe from generate)
yaswag generate --source ./path/to/your/project | yaswag serve

# serve from stdin with custom port
yaswag generate --source ./path/to/your/project | yaswag serve --port 9090

# pipe any OpenAPI spec to serve
cat swagger.yaml | yaswag serve
```

### Editor (Swagger Editor)

```bash
# launch Swagger Editor with default template
yaswag editor

# launch Swagger Editor on custom port
yaswag editor --port 9090

# launch Swagger Editor with existing spec file
yaswag editor --input ./swagger.yaml

# launch Swagger Editor with spec from URL
yaswag editor --input https://petstore3.swagger.io/api/v3/openapi.json

# pipe spec to editor
yaswag generate --source ./path/to/your/project | yaswag editor

# pipe any OpenAPI spec to editor
cat swagger.yaml | yaswag editor
```

### MCP (AI Assistant Integration)

YaSwag includes a Model Context Protocol (MCP) server that enables AI assistants like Claude to interact with your OpenAPI specifications through semantic search, schema exploration, and validation.

```bash
# start MCP server with a spec file
yaswag mcp ./swagger.yaml

# start MCP server with multiple spec files
yaswag mcp ./api/v1/openapi.yaml ./api/v2/openapi.yaml

# skip validation before starting
yaswag mcp --skip-validation ./swagger.yaml
```

#### Claude Code Configuration

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

#### Available MCP Tools

| Tool | Description |
|------|-------------|
| `search_endpoints` | Search for endpoints using natural language |
| `list_endpoints` | List all endpoints with optional filtering |
| `get_endpoint` | Get detailed endpoint information |
| `search_schemas` | Search for schema definitions |
| `get_schema` | Get detailed schema definition |
| `validate_spec` | Validate the OpenAPI specification |
| `get_spec_info` | Get general specification information |
| `generate_example` | Generate example request/response data |
| `find_related` | Find related endpoints |
| `list_tags` | List all tags with endpoint counts |
| `analyze_security` | Analyze security requirements |

### Audit (Security Analysis)

YaSwag includes a security audit command that analyzes your OpenAPI specification for common security issues and best practices violations.

```bash
# audit an OpenAPI specification (text output)
yaswag audit --input ./swagger.yaml

# audit with JSON output
yaswag audit --input ./swagger.yaml --format json

# audit from URL
yaswag audit --input https://example.com/openapi.json

# audit from stdin (pipe from generate)
yaswag generate --source ./path/to/your/project | yaswag audit

# pipe any OpenAPI spec to audit
cat swagger.yaml | yaswag audit
```

#### Security Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `UNPROTECTED_WRITE` | WARNING | POST/PUT/DELETE/PATCH endpoints without security requirements |
| `API_KEY_IN_QUERY` | WARNING | API keys passed in query parameters (should use headers) |
| `OAUTH_HTTP` | ERROR | OAuth URLs using HTTP instead of HTTPS |
| `DEPRECATED_NO_SECURITY` | INFO | Deprecated endpoints without security requirements |
| `SCOPE_NOT_DEFINED` | WARNING | OAuth scopes used but not defined in security scheme |

#### Exit Codes

- `0` - No ERROR-level issues found
- `1` - ERROR-level issues found

#### Sample Output

```bash
Security Audit Report
=====================

Summary
-------
Total Endpoints: 10
Protected: 8 (80%)
Unprotected: 2 (20%)

Findings (2 issues)
-------------------

[WARNING] UNPROTECTED_WRITE - Unprotected write operation
  Location: POST /users
  Message: POST endpoint has no security requirement
  Recommendation: Add authentication/authorization requirement to protect write operations

[WARNING] API_KEY_IN_QUERY - API key in query parameter
  Location: SecurityScheme 'apiKeyQuery'
  Message: API key 'apiKeyQuery' is passed in query parameter
  Recommendation: Use header-based API key for better security (prevents logging in URLs)

Security Schemes
----------------
- bearerAuth (http): 8 endpoints

Coverage by Tag
---------------
- users: 4/5 protected (80%)
- products: 4/5 protected (80%)
```

### Help

```bash
# show help
yaswag help

# module specific help
yaswag generate --help
yaswag validate --help
yaswag format --help
yaswag serve --help
yaswag editor --help
yaswag mcp --help
yaswag audit --help

# show version
yaswag version
```

## Usage (Annotation)

YaSwag uses its own **eccentric annotation syntax** with the `!` prefix. This syntax is designed to be concise, readable, and easy to write.

### API-Level Annotations

Define your API metadata in a comment block (typically in your `main.go` file):

```go
package main

// Swagger Petstore - OpenAPI 3.0
//
// !api 3.0.3
// !info "Swagger Petstore - OpenAPI 3.0" v1.0.27 "This is a sample Pet Store Server based on the OpenAPI 3.0 specification."
// !contact "API Support" <apiteam@swagger.io>
// !license Apache-2.0 https://www.apache.org/licenses/LICENSE-2.0.html
// !tos https://swagger.io/terms/
// !externalDocs https://swagger.io "Find out more about Swagger"
// !link "The Pet Store repository" https://github.com/swagger-api/swagger-petstore
// !link "The source API definition" https://github.com/swagger-api/swagger-petstore/blob/master/src/main/resources/openapi.yaml
// !security petstore_auth:oauth2:implicit "OAuth2 authentication" https://petstore3.swagger.io/oauth/authorize
// !scope petstore_auth write:pets "modify pets in your account"
// !scope petstore_auth read:pets "read your pets"
// !security api_key:apiKey:header "API Key authentication"
// !server https://petstore3.swagger.io/api/v3 "Production server"
// !server http://localhost:8080/api/v3 "Local development"
// !tag pet "Everything about your Pets"
// !tag store "Access to Petstore orders"
// !tag user "Operations about user"
func main() {}
```

### Operation-Level Annotations

Define operations on your handler functions:

```go
// GetUsers retrieves a list of users.
//
// !GET /users -> getUsers "Retrieve a list of users" #users
// !secure api_key petstore_auth
// !query limit:integer "The number of users to return" default=10
// !query offset:integer "The offset for pagination" default=0
// !ok User[] "Successful response"
// !error 500 ErrorResponse "Internal server error"
func GetUsers() {}

// CreateUser creates a new user.
//
// !POST /users -> createUser "Create a new user" #users
// !secure api_key
// !body CreateUserRequest "User data to create" required
// !ok 201 User "User created successfully"
// !error 400 ErrorResponse "Bad request"
func CreateUser() {}

// GetUserByID retrieves a user by ID.
//
// !GET /users/{id} -> getUserById "Retrieve a user by ID" #users
// !path id:integer "The user ID" required
// !ok User "Successful response"
// !error 404 ErrorResponse "User not found"
func GetUserByID() {}
```

### Schema Annotations

Define your models with `!model`. Field annotations (`!field`) are **optional** - YaSwag automatically infers schema from Go struct tags:

```go
// User represents a user of the system.
// !model "A user of the system"
type User struct {
    ID    int    `json:"id"`              // Auto: required (no omitempty), type inferred as integer
    Name  string `json:"name"`            // Auto: required, type inferred as string
    Email string `json:"email,omitempty"` // Auto: optional (has omitempty)

    // Use !field to add descriptions, examples, or override inferred values
    // !field "User's age in years" example=25
    Age int `json:"age,omitempty"`

    Internal string `json:"-"` // Excluded from schema (json:"-")
}

// ErrorResponse represents a standard error response.
// !model "Standard error response"
type ErrorResponse struct {
    // !field code:integer "Error code" required
    Code int `json:"code"`

    // !field message:string "Error message" required
    Message string `json:"message"`
}
```

## Annotation Reference

### API-Level Annotation Syntax

| Annotation | Syntax | Description |
|------------|--------|-------------|
| `!api` | `!api <version>` | Set the OpenAPI version (e.g., `3.0.3`, `3.1.0`) |
| `!info` | `!info "Title" vVersion "Description"` | Set API info (title, version, description) |
| `!contact` | `!contact "Name" <email> (url)` | Set contact information |
| `!license` | `!license Name URL` | Set license information |
| `!tos` | `!tos URL` | Set terms of service URL |
| `!server` | `!server URL "Description"` | Add a server URL |
| `!tag` | `!tag name "Description"` | Define an API tag |
| `!externalDocs` | `!externalDocs URL "Description"` | Set external documentation URL |
| `!link` | `!link "Label" URL` | Add a link to the description |

### Security Annotations Syntax

| Annotation | Syntax | Description |
|------------|--------|-------------|
| `!security` | `!security name:type:location "Description" URL` | Define a security scheme |
| `!scope` | `!scope securityName scopeName "Description"` | Add OAuth2 scope to a security scheme |
| `!secure` | `!secure securityName1 securityName2` | Apply security schemes to an operation |

#### Security Types

- `apiKey` - API Key authentication (location: `header`, `query`, or `cookie`)
- `oauth2` - OAuth 2.0 authentication (flow: `implicit`, `password`, `clientCredentials`, `authorizationCode`)
- `http` - HTTP authentication (scheme: `bearer`, `basic`)
- `openIdConnect` - OpenID Connect authentication

#### Security Examples

```go
// API Key in header
// !security api_key:apiKey:header "API Key authentication"

// OAuth2 with implicit flow
// !security petstore_auth:oauth2:implicit "OAuth2 authentication" https://example.com/oauth/authorize
// !scope petstore_auth write:pets "modify pets in your account"
// !scope petstore_auth read:pets "read your pets"

// HTTP Bearer token
// !security bearer_auth:http:bearer "Bearer token authentication"

// Apply to operation
// !secure api_key petstore_auth
```

### Operation Annotations

| Annotation | Syntax | Description |
|------------|--------|-------------|
| `!METHOD` | `!GET /path -> operationId "Summary" #tags` | Define an operation (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD) |
| `!query` | `!query name:type "Description" default=value required` | Add a query parameter |
| `!path` | `!path name:type "Description" required` | Add a path parameter |
| `!header` | `!header name:type "Description"` | Add a header parameter |
| `!body` | `!body SchemaRef "Description" required` | Add a request body |
| `!ok` | `!ok [status] SchemaRef "Description"` | Add a success response (default status: 200) |
| `!error` | `!error [status] SchemaRef "Description"` | Add an error response (default status: 500) |
| `!secure` | `!secure securityName1 securityName2` | Apply security requirements |

### Field and Model Annotations

| Annotation | Syntax | Description |
|------------|--------|-------------|
| `!model` | `!model "Description"` | Mark a struct as an OpenAPI schema |
| `!field` | `!field name:type "Description" required example=value` | (Optional) Describe a field in the schema |

#### Schema Inference Rules

Fields are automatically inferred from Go struct tags:

| Condition | Result |
|-----------|--------|
| `json:"fieldName"` | Field included, name from tag, **required** |
| `json:"fieldName,omitempty"` | Field included, name from tag, **optional** |
| `json:"-"` | Field **excluded** from schema |
| No json tag | Field included with Go field name |
| Inline comment (non-`!` prefix) | Used as field description |
| `!field` annotation | Overrides inferred values |

### Types

YaSwag supports the following types for parameters and fields:

- `string` - String type
- `integer` - 32-bit integer
- `int64` - 64-bit integer
- `number` - Floating point number
- `float` - 32-bit float
- `double` - 64-bit double
- `boolean` - Boolean type
- `array` - Array type
- `object` - Object type

### Array Schema References

For responses and request bodies that return arrays, use either syntax:

```go
// Suffix notation
// !ok User[] "List of users"

// Prefix notation
// !ok []User "List of users"
```

### Tags

Use hashtag notation to assign tags to operations:

```go
// Single tag
// !GET /users -> getUsers "Get users" #users

// Multiple tags
// !GET /admin/users -> getAdminUsers "Get admin users" #users #admin
```

## Complete Example

See the complete example in `examples/complete/main.go` which demonstrates the full Swagger Petstore API:

```go
package main

// Swagger Petstore - OpenAPI 3.0
//
// !api 3.0.3
// !info "Swagger Petstore - OpenAPI 3.0" v1.0.27 "This is a sample Pet Store Server based on the OpenAPI 3.0 specification. You can find out more about Swagger at [https://swagger.io](https://swagger.io)."
// !contact "" <apiteam@swagger.io>
// !license Apache-2.0 https://www.apache.org/licenses/LICENSE-2.0.html
// !tos https://swagger.io/terms/
// !externalDocs https://swagger.io "Find out more about Swagger"
// !link "The Pet Store repository" https://github.com/swagger-api/swagger-petstore
// !link "The source API definition for the Pet Store" https://github.com/swagger-api/swagger-petstore/blob/master/src/main/resources/openapi.yaml
// !security petstore_auth:oauth2:implicit "OAuth2 authentication" https://petstore3.swagger.io/oauth/authorize
// !scope petstore_auth write:pets "modify pets in your account"
// !scope petstore_auth read:pets "read your pets"
// !security api_key:apiKey:header "API Key authentication"
// !server https://petstore3.swagger.io/api/v3 "Petstore server"
// !server http://localhost:8080/api/v3 "Local development"
// !tag pet "Everything about your Pets"
// !tag store "Access to Petstore orders"
// !tag user "Operations about user"
func main() {}

// UpdatePet updates an existing pet by ID.
//
// !PUT /pet -> updatePet "Update an existing pet" #pet
// !secure petstore_auth api_key
// !body Pet "Update an existent pet in the store" required
// !ok Pet "Successful operation"
// !error 400 ApiResponse "Invalid ID supplied"
// !error 404 ApiResponse "Pet not found"
// !error 405 ApiResponse "Validation exception"
func UpdatePet() {}

// Pet represents a pet in the store.
// !model "A pet for sale in the pet store"
type Pet struct {
    // !field id:int64 "Unique identifier for the pet" example=10
    ID int64 `json:"id,omitempty"`

    // !field name:string "Name of the pet" required example="doggie"
    Name string `json:"name"`

    // !field status:string "Pet status in the store" example="available"
    Status string `json:"status,omitempty"`
}
```

Generate and serve the OpenAPI specification:

```bash
# Generate the specification
yaswag generate --source ./examples/complete --output ./swagger.yaml

# Serve with Swagger UI
yaswag serve --input ./swagger.yaml

# Or pipe directly
yaswag generate --source ./examples/complete | yaswag serve
```

## Swagger UI Features

The built-in Swagger UI includes:

- Dark/Light theme toggle
- Responsive design for mobile and desktop
- Validation badge with auto-validation
- OAuth2 authentication with scope selection
- Beautiful API documentation header with:
  - Version badge
  - OAS version indicator
  - External documentation links
  - Terms of service, contact, and license links

## Swagger Editor Features

The built-in Swagger Editor includes:

- Split-pane layout (editor + live preview)
- Syntax highlighting with Monaco-style editor
- Download as YAML or JSON
- Load existing specs from file, URL, or stdin
- Default template for new APIs

## License

MIT License
