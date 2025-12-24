// Package mcp provides a Model Context Protocol (MCP) server for YaSwag.
// It enables AI assistants to interact with OpenAPI specifications through
// semantic search, schema exploration, and validation tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"

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
	var specPath string

	// Use in-memory data if available (from stdin)
	if len(s.specData) > 0 {
		data = s.specData
	} else if len(s.specPaths) > 0 {
		specPath = s.specPaths[0]
		data, err = os.ReadFile(specPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no specification configured")
	}

	var doc openapi.Document

	// Detect format by file extension
	isYAML := false
	if specPath != "" {
		ext := strings.ToLower(filepath.Ext(specPath))
		isYAML = ext == ".yaml" || ext == ".yml"
	}

	// Parse based on format
	if isYAML {
		err = yaml.Unmarshal(data, &doc)
	} else {
		err = json.Unmarshal(data, &doc)
	}

	if err != nil {
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

// operationEntry represents a method-operation pair
type operationEntry struct {
	Method string
	Op     *openapi.Operation
}

// getOperations returns all non-nil operations from a PathItem
func getOperations(pathItem *openapi.PathItem) []operationEntry {
	entries := []operationEntry{
		{"GET", pathItem.Get},
		{"POST", pathItem.Post},
		{"PUT", pathItem.Put},
		{"DELETE", pathItem.Delete},
		{"PATCH", pathItem.Patch},
		{"HEAD", pathItem.Head},
		{"OPTIONS", pathItem.Options},
	}
	var result []operationEntry
	for _, e := range entries {
		if e.Op != nil {
			result = append(result, e)
		}
	}
	return result
}

// getOperationsWithoutHeadOptions returns operations excluding HEAD and OPTIONS
func getOperationsWithoutHeadOptions(pathItem *openapi.PathItem) []operationEntry {
	entries := []operationEntry{
		{"GET", pathItem.Get},
		{"POST", pathItem.Post},
		{"PUT", pathItem.Put},
		{"DELETE", pathItem.Delete},
		{"PATCH", pathItem.Patch},
	}
	var result []operationEntry
	for _, e := range entries {
		if e.Op != nil {
			result = append(result, e)
		}
	}
	return result
}

// filterOperation checks if an operation matches the given filters
func filterOperation(op *openapi.Operation, methodFilter, tagFilter, method string) bool {
	if methodFilter != "" && !strings.EqualFold(method, methodFilter) {
		return false
	}
	if tagFilter != "" && !hasTag(op.Tags, tagFilter) {
		return false
	}
	return true
}

// searchResult represents a search result with relevance score
type searchResult struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Summary     string   `json:"summary"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Score       float64  `json:"relevance_score"`
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

	queryTerms := strings.Fields(strings.ToLower(query))
	results := searchEndpointsInDoc(doc, queryTerms, methodFilter, tagFilter)

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

// searchEndpointsInDoc searches for endpoints matching the query terms
func searchEndpointsInDoc(doc *openapi.Document, queryTerms []string, methodFilter, tagFilter string) []searchResult {
	var results []searchResult
	for path, pathItem := range doc.Paths {
		for _, entry := range getOperations(pathItem) {
			if !filterOperation(entry.Op, methodFilter, tagFilter, entry.Method) {
				continue
			}
			score := calculateRelevanceScore(queryTerms, path, entry.Op)
			if score > 0 {
				results = append(results, searchResult{
					Path:        path,
					Method:      entry.Method,
					Summary:     entry.Op.Summary,
					Description: entry.Op.Description,
					Tags:        entry.Op.Tags,
					Score:       score,
				})
			}
		}
	}
	return results
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

// endpointInfo represents basic endpoint information
type endpointInfo struct {
	Path    string   `json:"path"`
	Method  string   `json:"method"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags,omitempty"`
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

	endpoints := listEndpointsInDoc(doc, methodFilter, tagFilter, pathPattern)
	sortEndpoints(endpoints)

	if len(endpoints) == 0 {
		return mcp.NewToolResultText("No endpoints found matching the filters."), nil
	}

	output, _ := json.MarshalIndent(endpoints, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d endpoints:\n\n%s", len(endpoints), string(output))), nil
}

// listEndpointsInDoc collects endpoints from the document with filtering
func listEndpointsInDoc(doc *openapi.Document, methodFilter, tagFilter, pathPattern string) []endpointInfo {
	var endpoints []endpointInfo
	for path, pathItem := range doc.Paths {
		if pathPattern != "" && !matchPathPattern(path, pathPattern) {
			continue
		}
		for _, entry := range getOperations(pathItem) {
			if !filterOperation(entry.Op, methodFilter, tagFilter, entry.Method) {
				continue
			}
			endpoints = append(endpoints, endpointInfo{
				Path:    path,
				Method:  entry.Method,
				Summary: entry.Op.Summary,
				Tags:    entry.Op.Tags,
			})
		}
	}
	return endpoints
}

// sortEndpoints sorts endpoints by path and method
func sortEndpoints(endpoints []endpointInfo) {
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})
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

// schemaSearchResult represents a schema search result
type schemaSearchResult struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type,omitempty"`
	Properties  []string `json:"properties,omitempty"`
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

	results := searchSchemasInDoc(doc.Components.Schemas, query, hasProperty)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	if len(results) == 0 {
		return mcp.NewToolResultText("No schemas found matching your query."), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d schemas:\n\n%s", len(results), string(output))), nil
}

// searchSchemasInDoc searches schemas matching the query
func searchSchemasInDoc(schemas map[string]*openapi.Schema, query, hasProperty string) []schemaSearchResult {
	var results []schemaSearchResult
	queryLower := strings.ToLower(query)

	for name, schema := range schemas {
		if !schemaMatchesQuery(name, schema, queryLower) {
			continue
		}
		if hasProperty != "" && !schemaHasProperty(schema, hasProperty) {
			continue
		}
		results = append(results, buildSchemaResult(name, schema))
	}
	return results
}

// schemaMatchesQuery checks if a schema matches the search query
func schemaMatchesQuery(name string, schema *openapi.Schema, queryLower string) bool {
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(schema.Description)
	return strings.Contains(nameLower, queryLower) || strings.Contains(descLower, queryLower)
}

// buildSchemaResult builds a schema search result
func buildSchemaResult(name string, schema *openapi.Schema) schemaSearchResult {
	propNames := getSortedPropertyNames(schema)
	schemaType := getSchemaType(schema)
	return schemaSearchResult{
		Name:        name,
		Description: schema.Description,
		Type:        schemaType,
		Properties:  propNames,
	}
}

// getSortedPropertyNames returns sorted property names from a schema
func getSortedPropertyNames(schema *openapi.Schema) []string {
	var propNames []string
	for propName := range schema.Properties {
		propNames = append(propNames, propName)
	}
	sort.Strings(propNames)
	return propNames
}

// getSchemaType returns the first type of a schema or empty string
func getSchemaType(schema *openapi.Schema) string {
	if len(schema.Type) > 0 {
		return schema.Type[0]
	}
	return ""
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

	example, errMsg := generateEndpointExample(op, exampleType, statusCode, doc)
	if errMsg != "" {
		return mcp.NewToolResultText(errMsg), nil
	}

	if example == nil {
		return mcp.NewToolResultText("Could not generate example (no schema found)."), nil
	}

	output, _ := json.MarshalIndent(example, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Generated %s example for %s %s:\n\n%s", exampleType, method, path, string(output))), nil
}

// generateEndpointExample generates example for request or response
func generateEndpointExample(op *openapi.Operation, exampleType, statusCode string, doc *openapi.Document) (any, string) {
	if exampleType == "request" {
		return generateRequestExample(op, doc)
	}
	return generateResponseExample(op, statusCode, doc)
}

// generateRequestExample generates example from request body
func generateRequestExample(op *openapi.Operation, doc *openapi.Document) (any, string) {
	if op.RequestBody == nil {
		return nil, "No request body defined for this endpoint."
	}
	return extractExampleFromContent(op.RequestBody.Content, doc), ""
}

// generateResponseExample generates example from response
func generateResponseExample(op *openapi.Operation, statusCode string, doc *openapi.Document) (any, string) {
	resp, ok := op.Responses[statusCode]
	if !ok {
		return nil, fmt.Sprintf("No response defined for status code %s", statusCode)
	}
	return extractExampleFromContent(resp.Content, doc), ""
}

// extractExampleFromContent extracts example from content map
func extractExampleFromContent(content map[string]openapi.MediaType, doc *openapi.Document) any {
	for _, mediaType := range content {
		if mediaType.Schema != nil {
			return generateSchemaExample(mediaType.Schema, doc)
		}
	}
	return nil
}

// generateSchemaExample generates example data from a schema
func generateSchemaExample(schema *openapi.Schema, doc *openapi.Document) any {
	if schema == nil {
		return nil
	}

	if resolved := resolveSchemaRef(schema, doc); resolved != nil {
		return generateSchemaExample(resolved, doc)
	}

	if schema.Example != nil {
		return schema.Example
	}

	if len(schema.Type) == 0 {
		return nil
	}

	return generateExampleByType(schema.Type[0], schema, doc)
}

// resolveSchemaRef resolves a $ref to the actual schema, returns nil if not a ref
func resolveSchemaRef(schema *openapi.Schema, doc *openapi.Document) *openapi.Schema {
	if schema.Ref == "" {
		return nil
	}
	refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	if doc.Components != nil && doc.Components.Schemas != nil {
		if refSchema, ok := doc.Components.Schemas[refName]; ok {
			return refSchema
		}
	}
	return nil
}

// generateExampleByType generates an example value based on schema type
func generateExampleByType(schemaType string, schema *openapi.Schema, doc *openapi.Document) any {
	switch schemaType {
	case "string":
		return generateStringExample(schema)
	case "integer":
		return 1
	case "number":
		return 1.0
	case "boolean":
		return true
	case "array":
		return generateArrayExample(schema, doc)
	case "object":
		return generateObjectExample(schema, doc)
	default:
		return nil
	}
}

// generateArrayExample generates an example array
func generateArrayExample(schema *openapi.Schema, doc *openapi.Document) []any {
	if schema.Items != nil {
		return []any{generateSchemaExample(schema.Items, doc)}
	}
	return []any{}
}

// generateObjectExample generates an example object
func generateObjectExample(schema *openapi.Schema, doc *openapi.Document) map[string]any {
	obj := make(map[string]any)
	for propName, propSchema := range schema.Properties {
		obj[propName] = generateSchemaExample(propSchema, doc)
	}
	return obj
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

// relatedEndpoint represents a related endpoint
type relatedEndpoint struct {
	Path    string `json:"path"`
	Method  string `json:"method"`
	Summary string `json:"summary,omitempty"`
}

// handleFindRelated finds related endpoints
func (s *Server) handleFindRelated(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, _ := req.RequireString("path")

	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resource := extractResource(path)
	if resource == "" {
		return mcp.NewToolResultText("Invalid path"), nil
	}

	related := findRelatedEndpoints(doc, path, resource)
	sortRelatedEndpoints(related)

	if len(related) == 0 {
		return mcp.NewToolResultText("No related endpoints found."), nil
	}

	output, _ := json.MarshalIndent(related, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Found %d related endpoints:\n\n%s", len(related), string(output))), nil
}

// extractResource extracts the resource name from a path
func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// findRelatedEndpoints finds endpoints with the same resource
func findRelatedEndpoints(doc *openapi.Document, currentPath, resource string) []relatedEndpoint {
	var related []relatedEndpoint
	for p, pathItem := range doc.Paths {
		if p == currentPath || !pathHasResource(p, resource) {
			continue
		}
		for _, entry := range getOperationsWithoutHeadOptions(pathItem) {
			related = append(related, relatedEndpoint{
				Path:    p,
				Method:  entry.Method,
				Summary: entry.Op.Summary,
			})
		}
	}
	return related
}

// pathHasResource checks if a path starts with the given resource
func pathHasResource(path, resource string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) > 0 && parts[0] == resource
}

// sortRelatedEndpoints sorts related endpoints by path and method
func sortRelatedEndpoints(related []relatedEndpoint) {
	sort.Slice(related, func(i, j int) bool {
		if related[i].Path == related[j].Path {
			return related[i].Method < related[j].Method
		}
		return related[i].Path < related[j].Path
	})
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

// securityAnalysis represents the result of security analysis
type securityAnalysis struct {
	GlobalSecurity       []openapi.SecurityRequirement      `json:"global_security,omitempty"`
	SecuritySchemes      map[string]*openapi.SecurityScheme `json:"security_schemes,omitempty"`
	ProtectedEndpoints   int                                `json:"protected_endpoints"`
	UnprotectedEndpoints int                                `json:"unprotected_endpoints"`
	EndpointsBySecurity  map[string][]string                `json:"endpoints_by_security"`
}

// handleAnalyzeSecurity analyzes security requirements
func (s *Server) handleAnalyzeSecurity(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	doc, err := s.loadSpec()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	analysis := analyzeDocSecurity(doc)
	output, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// analyzeDocSecurity performs security analysis on a document
func analyzeDocSecurity(doc *openapi.Document) securityAnalysis {
	analysis := securityAnalysis{
		GlobalSecurity:      doc.Security,
		EndpointsBySecurity: make(map[string][]string),
	}

	if doc.Components != nil {
		analysis.SecuritySchemes = doc.Components.SecuritySchemes
	}

	hasGlobalSecurity := len(doc.Security) > 0

	for path, pathItem := range doc.Paths {
		for _, entry := range getOperationsWithoutHeadOptions(pathItem) {
			endpoint := fmt.Sprintf("%s %s", entry.Method, path)
			analyzeEndpointSecurity(&analysis, entry.Op, endpoint, doc.Security, hasGlobalSecurity)
		}
	}

	return analysis
}

// analyzeEndpointSecurity analyzes security for a single endpoint
func analyzeEndpointSecurity(analysis *securityAnalysis, op *openapi.Operation, endpoint string, globalSecurity []openapi.SecurityRequirement, hasGlobalSecurity bool) {
	switch {
	case len(op.Security) > 0:
		analysis.ProtectedEndpoints++
		addSecuritySchemes(analysis, op.Security, endpoint)
	case hasGlobalSecurity:
		analysis.ProtectedEndpoints++
		addSecuritySchemes(analysis, globalSecurity, endpoint)
	default:
		analysis.UnprotectedEndpoints++
		analysis.EndpointsBySecurity["none"] = append(analysis.EndpointsBySecurity["none"], endpoint)
	}
}

// addSecuritySchemes adds security scheme entries for an endpoint
func addSecuritySchemes(analysis *securityAnalysis, security []openapi.SecurityRequirement, endpoint string) {
	for _, sec := range security {
		for schemeName := range sec {
			analysis.EndpointsBySecurity[schemeName] = append(analysis.EndpointsBySecurity[schemeName], endpoint)
		}
	}
}
