// Package mcp provides a Model Context Protocol (MCP) server for YaSwag.
// It enables AI assistants to interact with OpenAPI specifications through
// semantic search, schema exploration, and validation tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/fathurrohman26/yaswag/pkg/openapi"
	"github.com/fathurrohman26/yaswag/pkg/validator"
)

// Server wraps the MCP server with OpenAPI spec paths
type Server struct {
	mcpServer *server.MCPServer
	specPaths []string
	specData  []byte // in-memory spec data (for stdin)
}

// NewServer creates a new MCP server for OpenAPI interactions
func NewServer(specPaths []string) *Server {
	s := &Server{
		specPaths: specPaths,
	}

	// Create MCP server with tool capabilities
	s.mcpServer = server.NewMCPServer(
		"yaswag",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	// Register all tools
	s.registerTools()

	return s
}

// NewServerWithData creates a new MCP server with in-memory spec data
func NewServerWithData(data []byte) *Server {
	s := &Server{
		specData: data,
	}

	s.mcpServer = server.NewMCPServer(
		"yaswag",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	s.registerTools()

	return s
}

// ValidateSpec validates the spec data or file before starting the server
func ValidateSpec(data []byte) (*validator.ValidationResult, error) {
	v := validator.New()
	return v.Validate(data)
}

// ValidateSpecFile validates a spec file before starting the server
func ValidateSpecFile(path string) (*validator.ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}
	return ValidateSpec(data)
}

// Run starts the MCP server using stdio transport
func (s *Server) Run() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers all available tools with the MCP server
func (s *Server) registerTools() {
	// Search endpoints tool
	s.mcpServer.AddTool(
		mcp.NewTool("search_endpoints",
			mcp.WithDescription("Search for API endpoints using natural language or keywords. Supports semantic matching against endpoint paths, summaries, descriptions, tags, and operation IDs."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query (e.g., 'user authentication', 'create order', 'GET /pets')"),
			),
			mcp.WithString("method",
				mcp.Description("Filter by HTTP method"),
				mcp.Enum("GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"),
			),
			mcp.WithString("tag",
				mcp.Description("Filter by tag name"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results to return (default: 10)"),
			),
		),
		s.handleSearchEndpoints,
	)

	// List endpoints tool
	s.mcpServer.AddTool(
		mcp.NewTool("list_endpoints",
			mcp.WithDescription("List all API endpoints with optional filtering by method, tag, or path pattern."),
			mcp.WithString("method",
				mcp.Description("Filter by HTTP method"),
				mcp.Enum("GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"),
			),
			mcp.WithString("tag",
				mcp.Description("Filter by tag name"),
			),
			mcp.WithString("path_pattern",
				mcp.Description("Filter by path pattern (supports wildcards like /users/*)"),
			),
		),
		s.handleListEndpoints,
	)

	// Get endpoint details tool
	s.mcpServer.AddTool(
		mcp.NewTool("get_endpoint",
			mcp.WithDescription("Get detailed information about a specific endpoint including parameters, request body, responses, and security requirements."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("The endpoint path (e.g., /users/{id})"),
			),
			mcp.WithString("method",
				mcp.Required(),
				mcp.Description("The HTTP method"),
				mcp.Enum("GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"),
			),
		),
		s.handleGetEndpoint,
	)

	// Search schemas tool
	s.mcpServer.AddTool(
		mcp.NewTool("search_schemas",
			mcp.WithDescription("Search for schema definitions by name, description, or properties."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query for schema name or description"),
			),
			mcp.WithString("has_property",
				mcp.Description("Filter schemas that have a specific property name"),
			),
		),
		s.handleSearchSchemas,
	)

	// Get schema tool
	s.mcpServer.AddTool(
		mcp.NewTool("get_schema",
			mcp.WithDescription("Get detailed schema definition by name, including all properties, types, and validation rules."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The schema name (e.g., 'User', 'Order')"),
			),
			mcp.WithBoolean("resolve_refs",
				mcp.Description("Whether to resolve $ref references inline"),
			),
		),
		s.handleGetSchema,
	)

	// Validate spec tool
	s.mcpServer.AddTool(
		mcp.NewTool("validate_spec",
			mcp.WithDescription("Validate an OpenAPI specification and report any errors or warnings."),
			mcp.WithString("spec_path",
				mcp.Description("Path to the spec file (optional, uses loaded specs if not provided)"),
			),
		),
		s.handleValidateSpec,
	)

	// Get spec info tool
	s.mcpServer.AddTool(
		mcp.NewTool("get_spec_info",
			mcp.WithDescription("Get general information about the OpenAPI specification including title, version, servers, and tags."),
		),
		s.handleGetSpecInfo,
	)

	// Generate example tool
	s.mcpServer.AddTool(
		mcp.NewTool("generate_example",
			mcp.WithDescription("Generate example request or response data for an endpoint based on its schema."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("The endpoint path"),
			),
			mcp.WithString("method",
				mcp.Required(),
				mcp.Description("The HTTP method"),
				mcp.Enum("GET", "POST", "PUT", "DELETE", "PATCH"),
			),
			mcp.WithString("type",
				mcp.Description("Generate 'request' or 'response' example (default: response)"),
				mcp.Enum("request", "response"),
			),
			mcp.WithString("status_code",
				mcp.Description("Response status code for response examples (default: 200)"),
			),
		),
		s.handleGenerateExample,
	)

	// Find related endpoints tool
	s.mcpServer.AddTool(
		mcp.NewTool("find_related",
			mcp.WithDescription("Find endpoints related to a given endpoint (same resource, similar operations)."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("The endpoint path to find related endpoints for"),
			),
		),
		s.handleFindRelated,
	)

	// List tags tool
	s.mcpServer.AddTool(
		mcp.NewTool("list_tags",
			mcp.WithDescription("List all tags defined in the specification with their descriptions and endpoint counts."),
		),
		s.handleListTags,
	)

	// Analyze security tool
	s.mcpServer.AddTool(
		mcp.NewTool("analyze_security",
			mcp.WithDescription("Analyze security requirements across the API, listing authentication methods and protected endpoints."),
		),
		s.handleAnalyzeSecurity,
	)
}

// Helper function to load spec from paths
func (s *Server) loadSpec() (*openapi.Document, error) {
	var data []byte
	var err error

	// Use in-memory data if available (from stdin)
	if len(s.specData) > 0 {
		data = s.specData
	} else if len(s.specPaths) > 0 {
		data, err = os.ReadFile(s.specPaths[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read spec: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no specification configured")
	}

	var doc openapi.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	return &doc, nil
}

// Helper to get string argument with default
func getString(req mcp.CallToolRequest, name, defaultVal string) string {
	if v, err := req.RequireString(name); err == nil {
		return v
	}
	return defaultVal
}

// Helper to get int argument with default
func getInt(req mcp.CallToolRequest, name string, defaultVal int) int {
	if v, err := req.RequireFloat(name); err == nil {
		return int(v)
	}
	return defaultVal
}

// handleSearchEndpoints implements semantic search for endpoints
func (s *Server) handleSearchEndpoints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, _ := req.RequireString("query")
	methodFilter := getString(req, "method", "")
	tagFilter := getString(req, "tag", "")
	limit := getInt(req, "limit", 10)

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type result struct {
		Path        string   `json:"path"`
		Method      string   `json:"method"`
		Summary     string   `json:"summary"`
		Description string   `json:"description,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		Score       float64  `json:"relevance_score"`
	}

	var results []result
	queryLower := strings.ToLower(query)
	queryTerms := strings.Fields(queryLower)

	for path, pathItem := range doc.Paths {
		operations := map[string]*openapi.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
		}

		for method, op := range operations {
			if op == nil {
				continue
			}

			if methodFilter != "" && !strings.EqualFold(method, methodFilter) {
				continue
			}

			if tagFilter != "" && !hasTag(op.Tags, tagFilter) {
				continue
			}

			score := calculateRelevanceScore(queryTerms, path, op)
			if score > 0 {
				results = append(results, result{
					Path:        path,
					Method:      method,
					Summary:     op.Summary,
					Description: op.Description,
					Tags:        op.Tags,
					Score:       score,
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No endpoints found matching your query."), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d matching endpoints:\n\n%s", len(results), string(output))), nil
}

// hasTag checks if a tag exists in the list (case-insensitive)
func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, target) {
			return true
		}
	}
	return false
}

// calculateRelevanceScore calculates how relevant an endpoint is to search terms
func calculateRelevanceScore(queryTerms []string, path string, op *openapi.Operation) float64 {
	score := 0.0
	pathLower := strings.ToLower(path)
	summaryLower := strings.ToLower(op.Summary)
	descLower := strings.ToLower(op.Description)
	opIDLower := strings.ToLower(op.OperationID)

	for _, term := range queryTerms {
		if strings.Contains(pathLower, term) {
			score += 3.0
		}
		if strings.Contains(opIDLower, term) {
			score += 2.5
		}
		if strings.Contains(summaryLower, term) {
			score += 2.0
		}
		if strings.Contains(descLower, term) {
			score += 1.0
		}
		for _, tag := range op.Tags {
			if strings.Contains(strings.ToLower(tag), term) {
				score += 1.5
			}
		}
	}

	return score
}

// handleListEndpoints lists all endpoints with optional filtering
func (s *Server) handleListEndpoints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	methodFilter := getString(req, "method", "")
	tagFilter := getString(req, "tag", "")
	pathPattern := getString(req, "path_pattern", "")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type endpoint struct {
		Path    string   `json:"path"`
		Method  string   `json:"method"`
		Summary string   `json:"summary"`
		Tags    []string `json:"tags,omitempty"`
	}

	var endpoints []endpoint

	for path, pathItem := range doc.Paths {
		if pathPattern != "" && !matchPathPattern(path, pathPattern) {
			continue
		}

		operations := map[string]*openapi.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
		}

		for method, op := range operations {
			if op == nil {
				continue
			}

			if methodFilter != "" && !strings.EqualFold(method, methodFilter) {
				continue
			}

			if tagFilter != "" && !hasTag(op.Tags, tagFilter) {
				continue
			}

			endpoints = append(endpoints, endpoint{
				Path:    path,
				Method:  method,
				Summary: op.Summary,
				Tags:    op.Tags,
			})
		}
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	if len(endpoints) == 0 {
		return mcp.NewToolResultText("No endpoints found matching the filters."), nil
	}

	output, _ := json.MarshalIndent(endpoints, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d endpoints:\n\n%s", len(endpoints), string(output))), nil
}

// matchPathPattern checks if a path matches a pattern with wildcards
func matchPathPattern(path, pattern string) bool {
	if pattern == "" {
		return true
	}
	pattern = strings.ReplaceAll(pattern, "*", "")
	return strings.Contains(strings.ToLower(path), strings.ToLower(pattern))
}

// handleGetEndpoint gets detailed information about a specific endpoint
func (s *Server) handleGetEndpoint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, _ := req.RequireString("path")
	method, _ := req.RequireString("method")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pathItem, ok := doc.Paths[path]
	if !ok {
		return mcp.NewToolResultText(fmt.Sprintf("Endpoint not found: %s", path)), nil
	}

	op := getOperation(pathItem, method)
	if op == nil {
		return mcp.NewToolResultText(fmt.Sprintf("Method %s not found for path %s", method, path)), nil
	}

	type endpointDetail struct {
		Path        string                        `json:"path"`
		Method      string                        `json:"method"`
		Summary     string                        `json:"summary,omitempty"`
		Description string                        `json:"description,omitempty"`
		OperationID string                        `json:"operationId,omitempty"`
		Tags        []string                      `json:"tags,omitempty"`
		Parameters  []*openapi.Parameter          `json:"parameters,omitempty"`
		RequestBody *openapi.RequestBody          `json:"requestBody,omitempty"`
		Responses   map[string]*openapi.Response  `json:"responses,omitempty"`
		Security    []openapi.SecurityRequirement `json:"security,omitempty"`
		Deprecated  bool                          `json:"deprecated,omitempty"`
	}

	params := pathItem.Parameters
	params = append(params, op.Parameters...)

	detail := endpointDetail{
		Path:        path,
		Method:      strings.ToUpper(method),
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationID,
		Tags:        op.Tags,
		Parameters:  params,
		RequestBody: op.RequestBody,
		Responses:   op.Responses,
		Security:    op.Security,
		Deprecated:  op.Deprecated,
	}

	output, _ := json.MarshalIndent(detail, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// getOperation returns the operation for a given method
func getOperation(pathItem *openapi.PathItem, method string) *openapi.Operation {
	switch strings.ToUpper(method) {
	case "GET":
		return pathItem.Get
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	case "DELETE":
		return pathItem.Delete
	case "PATCH":
		return pathItem.Patch
	case "HEAD":
		return pathItem.Head
	case "OPTIONS":
		return pathItem.Options
	default:
		return nil
	}
}

// handleSearchSchemas searches for schema definitions
func (s *Server) handleSearchSchemas(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, _ := req.RequireString("query")
	hasProperty := getString(req, "has_property", "")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if doc.Components == nil || doc.Components.Schemas == nil {
		return mcp.NewToolResultText("No schemas defined in the specification."), nil
	}

	type schemaResult struct {
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Type        string   `json:"type,omitempty"`
		Properties  []string `json:"properties,omitempty"`
	}

	var results []schemaResult
	queryLower := strings.ToLower(query)

	for name, schema := range doc.Components.Schemas {
		nameLower := strings.ToLower(name)
		descLower := strings.ToLower(schema.Description)

		if !strings.Contains(nameLower, queryLower) && !strings.Contains(descLower, queryLower) {
			continue
		}

		if hasProperty != "" && !schemaHasProperty(schema, hasProperty) {
			continue
		}

		var propNames []string
		for propName := range schema.Properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		schemaType := ""
		if len(schema.Type) > 0 {
			schemaType = schema.Type[0]
		}

		results = append(results, schemaResult{
			Name:        name,
			Description: schema.Description,
			Type:        schemaType,
			Properties:  propNames,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	if len(results) == 0 {
		return mcp.NewToolResultText("No schemas found matching your query."), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d schemas:\n\n%s", len(results), string(output))), nil
}

// schemaHasProperty checks if a schema has a specific property
func schemaHasProperty(schema *openapi.Schema, propName string) bool {
	for name := range schema.Properties {
		if strings.EqualFold(name, propName) {
			return true
		}
	}
	return false
}

// handleGetSchema gets detailed schema definition
func (s *Server) handleGetSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.RequireString("name")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if doc.Components == nil || doc.Components.Schemas == nil {
		return mcp.NewToolResultText("No schemas defined in the specification."), nil
	}

	schema, ok := doc.Components.Schemas[name]
	if !ok {
		for schemaName, s := range doc.Components.Schemas {
			if strings.EqualFold(schemaName, name) {
				schema = s
				name = schemaName
				ok = true
				break
			}
		}
	}

	if !ok {
		return mcp.NewToolResultText(fmt.Sprintf("Schema '%s' not found.", name)), nil
	}

	output, _ := json.MarshalIndent(map[string]any{
		"name":   name,
		"schema": schema,
	}, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// handleValidateSpec validates the OpenAPI specification
func (s *Server) handleValidateSpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	specPath := getString(req, "spec_path", "")

	paths := s.specPaths
	if specPath != "" {
		paths = []string{specPath}
	}

	if len(paths) == 0 {
		return mcp.NewToolResultText("No specification file to validate."), nil
	}

	data, err := os.ReadFile(paths[0])
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read spec: %v", err)), nil
	}

	v := validator.New()
	result, err := v.Validate(data)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Validation error: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// handleGetSpecInfo gets general specification information
func (s *Server) handleGetSpecInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type specInfo struct {
		OpenAPIVersion string           `json:"openapi_version"`
		Title          string           `json:"title"`
		Version        string           `json:"version"`
		Description    string           `json:"description,omitempty"`
		Servers        []openapi.Server `json:"servers,omitempty"`
		Tags           []openapi.Tag    `json:"tags,omitempty"`
		EndpointCount  int              `json:"endpoint_count"`
		SchemaCount    int              `json:"schema_count"`
	}

	endpointCount := countEndpoints(doc)
	schemaCount := 0
	if doc.Components != nil && doc.Components.Schemas != nil {
		schemaCount = len(doc.Components.Schemas)
	}

	info := specInfo{
		OpenAPIVersion: doc.OpenAPI,
		Title:          doc.Info.Title,
		Version:        doc.Info.Version,
		Description:    doc.Info.Description,
		Servers:        doc.Servers,
		Tags:           doc.Tags,
		EndpointCount:  endpointCount,
		SchemaCount:    schemaCount,
	}

	output, _ := json.MarshalIndent(info, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// countEndpoints counts the number of operations in the spec
func countEndpoints(doc *openapi.Document) int {
	count := 0
	for _, pathItem := range doc.Paths {
		if pathItem.Get != nil {
			count++
		}
		if pathItem.Post != nil {
			count++
		}
		if pathItem.Put != nil {
			count++
		}
		if pathItem.Delete != nil {
			count++
		}
		if pathItem.Patch != nil {
			count++
		}
		if pathItem.Head != nil {
			count++
		}
		if pathItem.Options != nil {
			count++
		}
	}
	return count
}

// handleGenerateExample generates example data for an endpoint
func (s *Server) handleGenerateExample(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, _ := req.RequireString("path")
	method, _ := req.RequireString("method")
	exampleType := getString(req, "type", "response")
	statusCode := getString(req, "status_code", "200")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pathItem, ok := doc.Paths[path]
	if !ok {
		return mcp.NewToolResultText(fmt.Sprintf("Endpoint not found: %s", path)), nil
	}

	op := getOperation(pathItem, method)
	if op == nil {
		return mcp.NewToolResultText(fmt.Sprintf("Method %s not found for path %s", method, path)), nil
	}

	var example any

	if exampleType == "request" {
		if op.RequestBody == nil {
			return mcp.NewToolResultText("No request body defined for this endpoint."), nil
		}
		for _, mediaType := range op.RequestBody.Content {
			if mediaType.Schema != nil {
				example = generateSchemaExample(mediaType.Schema, doc)
				break
			}
		}
	} else {
		resp, ok := op.Responses[statusCode]
		if !ok {
			return mcp.NewToolResultText(fmt.Sprintf("No response defined for status code %s", statusCode)), nil
		}
		for _, mediaType := range resp.Content {
			if mediaType.Schema != nil {
				example = generateSchemaExample(mediaType.Schema, doc)
				break
			}
		}
	}

	if example == nil {
		return mcp.NewToolResultText("Could not generate example (no schema found)."), nil
	}

	output, _ := json.MarshalIndent(example, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Generated %s example for %s %s:\n\n%s", exampleType, method, path, string(output))), nil
}

// generateSchemaExample generates example data from a schema
func generateSchemaExample(schema *openapi.Schema, doc *openapi.Document) any {
	if schema == nil {
		return nil
	}

	if schema.Ref != "" {
		refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		if doc.Components != nil && doc.Components.Schemas != nil {
			if refSchema, ok := doc.Components.Schemas[refName]; ok {
				return generateSchemaExample(refSchema, doc)
			}
		}
		return nil
	}

	if schema.Example != nil {
		return schema.Example
	}

	if len(schema.Type) == 0 {
		return nil
	}

	switch schema.Type[0] {
	case "string":
		return generateStringExample(schema)
	case "integer":
		return 1
	case "number":
		return 1.0
	case "boolean":
		return true
	case "array":
		if schema.Items != nil {
			return []any{generateSchemaExample(schema.Items, doc)}
		}
		return []any{}
	case "object":
		obj := make(map[string]any)
		for propName, propSchema := range schema.Properties {
			obj[propName] = generateSchemaExample(propSchema, doc)
		}
		return obj
	default:
		return nil
	}
}

// generateStringExample generates example string based on format
func generateStringExample(schema *openapi.Schema) string {
	if len(schema.Enum) > 0 {
		if s, ok := schema.Enum[0].(string); ok {
			return s
		}
	}
	switch schema.Format {
	case "date":
		return "2024-01-15"
	case "date-time":
		return "2024-01-15T10:30:00Z"
	case "email":
		return "user@example.com"
	case "uri":
		return "https://example.com"
	case "uuid":
		return "550e8400-e29b-41d4-a716-446655440000"
	default:
		return "string"
	}
}

// handleFindRelated finds related endpoints
func (s *Server) handleFindRelated(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, _ := req.RequireString("path")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return mcp.NewToolResultText("Invalid path"), nil
	}

	resource := parts[0]

	type relatedEndpoint struct {
		Path    string `json:"path"`
		Method  string `json:"method"`
		Summary string `json:"summary,omitempty"`
	}

	var related []relatedEndpoint

	for p, pathItem := range doc.Paths {
		if p == path {
			continue
		}

		pParts := strings.Split(strings.Trim(p, "/"), "/")
		if len(pParts) > 0 && pParts[0] == resource {
			operations := map[string]*openapi.Operation{
				"GET":    pathItem.Get,
				"POST":   pathItem.Post,
				"PUT":    pathItem.Put,
				"DELETE": pathItem.Delete,
				"PATCH":  pathItem.Patch,
			}

			for method, op := range operations {
				if op != nil {
					related = append(related, relatedEndpoint{
						Path:    p,
						Method:  method,
						Summary: op.Summary,
					})
				}
			}
		}
	}

	sort.Slice(related, func(i, j int) bool {
		if related[i].Path == related[j].Path {
			return related[i].Method < related[j].Method
		}
		return related[i].Path < related[j].Path
	})

	if len(related) == 0 {
		return mcp.NewToolResultText("No related endpoints found."), nil
	}

	output, _ := json.MarshalIndent(related, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d related endpoints:\n\n%s", len(related), string(output))), nil
}

// handleListTags lists all tags with descriptions
func (s *Server) handleListTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type tagInfo struct {
		Name          string `json:"name"`
		Description   string `json:"description,omitempty"`
		EndpointCount int    `json:"endpoint_count"`
	}

	tagCounts := make(map[string]int)
	for _, pathItem := range doc.Paths {
		operations := []*openapi.Operation{
			pathItem.Get, pathItem.Post, pathItem.Put,
			pathItem.Delete, pathItem.Patch, pathItem.Head, pathItem.Options,
		}
		for _, op := range operations {
			if op == nil {
				continue
			}
			for _, tag := range op.Tags {
				tagCounts[tag]++
			}
		}
	}

	var tags []tagInfo

	for _, tag := range doc.Tags {
		tags = append(tags, tagInfo{
			Name:          tag.Name,
			Description:   tag.Description,
			EndpointCount: tagCounts[tag.Name],
		})
		delete(tagCounts, tag.Name)
	}

	for tagName, count := range tagCounts {
		tags = append(tags, tagInfo{
			Name:          tagName,
			EndpointCount: count,
		})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	if len(tags) == 0 {
		return mcp.NewToolResultText("No tags defined in the specification."), nil
	}

	output, _ := json.MarshalIndent(tags, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// handleAnalyzeSecurity analyzes security requirements
func (s *Server) handleAnalyzeSecurity(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type securityAnalysis struct {
		GlobalSecurity       []openapi.SecurityRequirement      `json:"global_security,omitempty"`
		SecuritySchemes      map[string]*openapi.SecurityScheme `json:"security_schemes,omitempty"`
		ProtectedEndpoints   int                                `json:"protected_endpoints"`
		UnprotectedEndpoints int                                `json:"unprotected_endpoints"`
		EndpointsBySecurity  map[string][]string                `json:"endpoints_by_security"`
	}

	analysis := securityAnalysis{
		GlobalSecurity:      doc.Security,
		EndpointsBySecurity: make(map[string][]string),
	}

	if doc.Components != nil {
		analysis.SecuritySchemes = doc.Components.SecuritySchemes
	}

	hasGlobalSecurity := len(doc.Security) > 0

	for path, pathItem := range doc.Paths {
		operations := map[string]*openapi.Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"DELETE": pathItem.Delete,
			"PATCH":  pathItem.Patch,
		}

		for method, op := range operations {
			if op == nil {
				continue
			}

			endpoint := fmt.Sprintf("%s %s", method, path)

			switch {
			case len(op.Security) > 0:
				analysis.ProtectedEndpoints++
				for _, sec := range op.Security {
					for schemeName := range sec {
						analysis.EndpointsBySecurity[schemeName] = append(
							analysis.EndpointsBySecurity[schemeName], endpoint)
					}
				}
			case hasGlobalSecurity:
				analysis.ProtectedEndpoints++
				for _, sec := range doc.Security {
					for schemeName := range sec {
						analysis.EndpointsBySecurity[schemeName] = append(
							analysis.EndpointsBySecurity[schemeName], endpoint)
					}
				}
			default:
				analysis.UnprotectedEndpoints++
				analysis.EndpointsBySecurity["none"] = append(
					analysis.EndpointsBySecurity["none"], endpoint)
			}
		}
	}

	output, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
