package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/fathurrohman26/yaswag/pkg/openapi"
)

func TestNewServer(t *testing.T) {
	server := NewServer([]string{"test.json"})
	if server == nil {
		t.Fatal("expected server to be created")
	}
	if len(server.specPaths) != 1 {
		t.Errorf("expected 1 spec path, got %d", len(server.specPaths))
	}
	if server.mcpServer == nil {
		t.Error("expected MCP server to be initialized")
	}
}

func TestCalculateRelevanceScore(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		path     string
		opID     string
		summary  string
		tags     []string
		minScore float64
	}{
		{
			name:     "path match",
			query:    "users",
			path:     "/users",
			opID:     "listUsers",
			summary:  "List all users",
			tags:     []string{"users"},
			minScore: 8.0,
		},
		{
			name:     "no match",
			query:    "orders",
			path:     "/users",
			opID:     "listUsers",
			summary:  "List all users",
			tags:     []string{"users"},
			minScore: 0.0,
		},
		{
			name:     "partial match",
			query:    "list",
			path:     "/users",
			opID:     "listUsers",
			summary:  "List all users",
			tags:     []string{"users"},
			minScore: 4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &openapi.Operation{
				OperationID: tt.opID,
				Summary:     tt.summary,
				Tags:        tt.tags,
			}
			queryTerms := strings.Fields(strings.ToLower(tt.query))
			score := calculateRelevanceScore(queryTerms, tt.path, op)
			if score < tt.minScore {
				t.Errorf("expected score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}

func TestMatchPathPattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		match   bool
	}{
		{"/users", "", true},
		{"/users", "users", true},
		{"/users/{id}", "users", true},
		{"/users/{id}", "id", true},
		{"/orders", "users", false},
		{"/users/{id}", "/users", true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			result := matchPathPattern(tt.path, tt.pattern)
			if result != tt.match {
				t.Errorf("matchPathPattern(%s, %s) = %v, expected %v", tt.path, tt.pattern, result, tt.match)
			}
		})
	}
}

func TestHasTag(t *testing.T) {
	tests := []struct {
		tags   []string
		target string
		want   bool
	}{
		{[]string{"users", "admin"}, "users", true},
		{[]string{"users", "admin"}, "Users", true},
		{[]string{"users", "admin"}, "orders", false},
		{[]string{}, "users", false},
	}

	for _, tt := range tests {
		result := hasTag(tt.tags, tt.target)
		if result != tt.want {
			t.Errorf("hasTag(%v, %s) = %v, want %v", tt.tags, tt.target, result, tt.want)
		}
	}
}

func TestLoadSpec(t *testing.T) {
	// Test with no paths
	s := &Server{specPaths: []string{}}
	_, err := s.loadSpec()
	if err == nil {
		t.Error("expected error for empty paths")
	}

	// Test with non-existent file
	s = &Server{specPaths: []string{"/nonexistent/file.json"}}
	_, err = s.loadSpec()
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	// Test with valid spec
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.json")
	specContent := `{"openapi": "3.0.3", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	s = &Server{specPaths: []string{specPath}}
	doc, err := s.loadSpec()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Info.Title != "Test" {
		t.Errorf("expected title 'Test', got %s", doc.Info.Title)
	}
}

func TestSearchEndpointsHandler(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.json")
	specContent := `{
		"openapi": "3.0.3",
		"info": {"title": "Test API", "version": "1.0.0"},
		"paths": {
			"/users": {
				"get": {
					"summary": "List users",
					"operationId": "listUsers",
					"tags": ["users"],
					"responses": {"200": {"description": "OK"}}
				},
				"post": {
					"summary": "Create user",
					"operationId": "createUser",
					"tags": ["users"],
					"responses": {"201": {"description": "Created"}}
				}
			},
			"/orders": {
				"get": {
					"summary": "List orders",
					"operationId": "listOrders",
					"tags": ["orders"],
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer([]string{specPath})
	ctx := context.Background()

	// Test searching for "users"
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "users",
		"limit": float64(10),
	}

	result, err := s.handleSearchEndpoints(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("expected non-error result")
	}

	text := getResultText(result)
	if !strings.Contains(text, "/users") {
		t.Error("expected result to contain /users")
	}
}

func TestListEndpointsHandler(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.json")
	specContent := `{
		"openapi": "3.0.3",
		"info": {"title": "Test API", "version": "1.0.0"},
		"paths": {
			"/users": {
				"get": {"summary": "List users", "tags": ["users"], "responses": {"200": {"description": "OK"}}},
				"post": {"summary": "Create user", "tags": ["users"], "responses": {"201": {"description": "Created"}}}
			}
		}
	}`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer([]string{specPath})
	ctx := context.Background()

	req := mcp.CallToolRequest{}
	result, err := s.handleListEndpoints(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := getResultText(result)
	if !strings.Contains(text, "Found 2 endpoints") {
		t.Errorf("expected 2 endpoints, got: %s", text)
	}
}

func TestGetSpecInfoHandler(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.json")
	specContent := `{
		"openapi": "3.0.3",
		"info": {"title": "Test API", "version": "1.0.0", "description": "A test API"},
		"servers": [{"url": "http://localhost:8080"}],
		"tags": [{"name": "users", "description": "User operations"}],
		"paths": {
			"/users": {
				"get": {"summary": "List users", "responses": {"200": {"description": "OK"}}}
			}
		},
		"components": {
			"schemas": {
				"User": {"type": "object", "properties": {"id": {"type": "integer"}}}
			}
		}
	}`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer([]string{specPath})
	ctx := context.Background()

	req := mcp.CallToolRequest{}
	result, err := s.handleGetSpecInfo(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := getResultText(result)
	if !strings.Contains(text, "Test API") {
		t.Error("expected result to contain API title")
	}
	if !strings.Contains(text, "endpoint_count") {
		t.Error("expected result to contain endpoint count")
	}
	if !strings.Contains(text, "schema_count") {
		t.Error("expected result to contain schema count")
	}
}

func TestGetEndpointHandler(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.json")
	specContent := `{
		"openapi": "3.0.3",
		"info": {"title": "Test API", "version": "1.0.0"},
		"paths": {
			"/users/{id}": {
				"get": {
					"summary": "Get user by ID",
					"operationId": "getUser",
					"parameters": [{"name": "id", "in": "path", "required": true}],
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer([]string{specPath})
	ctx := context.Background()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"path":   "/users/{id}",
		"method": "GET",
	}

	result, err := s.handleGetEndpoint(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := getResultText(result)
	if !strings.Contains(text, "getUser") {
		t.Error("expected result to contain operation ID")
	}
	if !strings.Contains(text, "Get user by ID") {
		t.Error("expected result to contain summary")
	}
}

func TestGenerateStringExample(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"date", "2024-01-15"},
		{"date-time", "2024-01-15T10:30:00Z"},
		{"email", "user@example.com"},
		{"uri", "https://example.com"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			schema := &openapi.Schema{Format: tt.format}
			result := generateStringExample(schema)
			if result != tt.expected {
				t.Errorf("generateStringExample with format %s = %s, want %s", tt.format, result, tt.expected)
			}
		})
	}
}

func TestGetOperation(t *testing.T) {
	pathItem := &openapi.PathItem{
		Get:     &openapi.Operation{Summary: "get"},
		Post:    &openapi.Operation{Summary: "post"},
		Put:     &openapi.Operation{Summary: "put"},
		Delete:  &openapi.Operation{Summary: "delete"},
		Patch:   &openapi.Operation{Summary: "patch"},
		Head:    &openapi.Operation{Summary: "head"},
		Options: &openapi.Operation{Summary: "options"},
	}

	tests := []struct {
		method  string
		summary string
	}{
		{"GET", "get"},
		{"POST", "post"},
		{"PUT", "put"},
		{"DELETE", "delete"},
		{"PATCH", "patch"},
		{"HEAD", "head"},
		{"OPTIONS", "options"},
		{"UNKNOWN", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			op := getOperation(pathItem, tt.method)
			if tt.summary == "" {
				if op != nil {
					t.Errorf("expected nil for method %s", tt.method)
				}
			} else {
				if op == nil || op.Summary != tt.summary {
					t.Errorf("getOperation(%s) = %v, want summary %s", tt.method, op, tt.summary)
				}
			}
		})
	}
}

// Helper to get text from result
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text
		}
	}
	return ""
}
