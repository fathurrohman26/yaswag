package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fathurrohman26/yaswag/pkg/openapi"
)

// testHelper provides common test utilities
type testHelper struct {
	t      *testing.T
	tmpDir string
}

func newTestHelper(t *testing.T) *testHelper {
	tmpDir, err := os.MkdirTemp("", "yaswag-test")
	if err != nil {
		t.Fatal(err)
	}
	return &testHelper{t: t, tmpDir: tmpDir}
}

func (h *testHelper) cleanup() {
	_ = os.RemoveAll(h.tmpDir)
}

func (h *testHelper) writeFile(name, content string) {
	path := filepath.Join(h.tmpDir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		h.t.Fatal(err)
	}
}

func (h *testHelper) parse() *Parser {
	p := New()
	if err := p.ParseDir(h.tmpDir); err != nil {
		h.t.Fatalf("ParseDir() error = %v", err)
	}
	return p
}

func TestParser_ParseDir(t *testing.T) {
	h := newTestHelper(t)
	defer h.cleanup()

	h.writeFile("api.go", parseDirTestContent)

	p := h.parse()
	spec := p.GetSpec()
	if spec == nil {
		t.Fatal("Expected spec to be parsed")
	}

	t.Run("version", func(t *testing.T) { verifyVersion(t, spec) })
	t.Run("info", func(t *testing.T) { verifyInfo(t, spec) })
	t.Run("contact", func(t *testing.T) { verifyContact(t, spec) })
	t.Run("license", func(t *testing.T) { verifyLicense(t, spec) })
	t.Run("servers", func(t *testing.T) { verifyServers(t, spec) })
	t.Run("tags", func(t *testing.T) { verifyTags(t, spec) })
	t.Run("operations", func(t *testing.T) { verifyOperations(t, spec) })
	t.Run("schemas", func(t *testing.T) { verifySchemas(t, p) })
}

const parseDirTestContent = `package main

// YaSwag Test API
//
// !api 3.0.3
// !info "Test API" v1.0.0 "A test API"
// !contact "Support" <support@test.com>
// !license MIT https://opensource.org/licenses/MIT
// !server https://api.test.com/v1 "Production"
// !tag users "User operations"
func main() {}

// GetUsers retrieves all users.
//
// !GET /users -> getUsers "Get all users" #users
// !query limit:integer "Number of results" default=10
// !ok User[] "Successful response"
// !error 500 - "Server error"
func GetUsers() {}

// CreateUser creates a new user.
//
// !POST /users -> createUser "Create a user" #users
// !body CreateUserRequest "User data" required
// !ok 201 User "User created"
// !error 400 Error "Bad request"
func CreateUser() {}

// User represents a user in the system.
// !model "A user in the system"
type User struct {
	// !field id:integer "User ID" required example=123
	ID int ` + "`json:\"id\"`" + `

	// !field name:string "User name" required example="John Doe"
	Name string ` + "`json:\"name\"`" + `

	// !field email:string "User email" example="john@example.com"
	Email string ` + "`json:\"email,omitempty\"`" + `
}
`

func verifyVersion(t *testing.T, spec *SpecData) {
	if spec.Version != "3.0.3" {
		t.Errorf("Version = %v, want %v", spec.Version, "3.0.3")
	}
}

func verifyInfo(t *testing.T, spec *SpecData) {
	assertEqual(t, "Info.Title", spec.Info.Title, "Test API")
	assertEqual(t, "Info.Version", spec.Info.Version, "1.0.0")
	assertEqual(t, "Info.Description", spec.Info.Description, "A test API")
}

func verifyContact(t *testing.T, spec *SpecData) {
	if spec.Info.Contact == nil {
		t.Fatal("Expected contact information")
	}
	assertEqual(t, "Contact.Name", spec.Info.Contact.Name, "Support")
	assertEqual(t, "Contact.Email", spec.Info.Contact.Email, "support@test.com")
}

func verifyLicense(t *testing.T, spec *SpecData) {
	if spec.Info.License == nil {
		t.Fatal("Expected license information")
	}
	assertEqual(t, "License.Name", spec.Info.License.Name, "MIT")
}

func verifyServers(t *testing.T, spec *SpecData) {
	assertLen(t, "servers", len(spec.Servers), 1)
	assertEqual(t, "Server.URL", spec.Servers[0].URL, "https://api.test.com/v1")
}

func verifyTags(t *testing.T, spec *SpecData) {
	assertLen(t, "tags", len(spec.Tags), 1)
	assertEqual(t, "Tag.Name", spec.Tags[0].Name, "users")
}

func verifyOperations(t *testing.T, spec *SpecData) {
	assertLen(t, "operations", len(spec.Operations), 2)
	getOp, postOp := findOperations(spec)
	verifyGetOperation(t, getOp)
	verifyPostOperation(t, postOp)
}

func findOperations(spec *SpecData) (*OperationData, *OperationData) {
	var getOp, postOp *OperationData
	for i := range spec.Operations {
		switch spec.Operations[i].Method {
		case "GET":
			getOp = &spec.Operations[i]
		case "POST":
			postOp = &spec.Operations[i]
		}
	}
	return getOp, postOp
}

func verifyGetOperation(t *testing.T, op *OperationData) {
	if op == nil {
		t.Fatal("Expected GET operation")
	}
	assertEqual(t, "GET Path", op.Path, "/users")
	assertEqual(t, "GET OperationID", op.OperationID, "getUsers")
	assertLen(t, "GET Parameters", len(op.Parameters), 1)
	assertLen(t, "GET Responses", len(op.Responses), 2)
}

func verifyPostOperation(t *testing.T, op *OperationData) {
	if op == nil {
		t.Fatal("Expected POST operation")
	}
	if op.RequestBody == nil {
		t.Fatal("Expected POST operation to have request body")
	}
}

func verifySchemas(t *testing.T, p *Parser) {
	schemas := p.GetGlobalSchemas()
	assertLen(t, "schemas", len(schemas), 1)

	userSchema := schemas["User"]
	if userSchema == nil {
		t.Fatal("Expected User schema")
	}
	assertEqual(t, "User description", userSchema.Description, "A user in the system")
	if userSchema.Schema == nil {
		t.Fatal("Expected User schema.Schema")
	}
	assertLen(t, "User properties", len(userSchema.Schema.Properties), 3)
}

func assertEqual(t *testing.T, name, got, want string) {
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertLen(t *testing.T, name string, got, want int) {
	if got != want {
		t.Errorf("%s count = %d, want %d", name, got, want)
	}
}

func TestParser_Generate(t *testing.T) {
	h := newTestHelper(t)
	defer h.cleanup()

	h.writeFile("api.go", generateTestContent)

	p := h.parse()
	doc := p.Generate()
	if doc == nil {
		t.Fatal("Expected document to be generated")
	}

	t.Run("openapi_version", func(t *testing.T) {
		assertEqual(t, "OpenAPI version", doc.OpenAPI, "3.0.3")
	})
	t.Run("info", func(t *testing.T) {
		assertEqual(t, "Info.Title", doc.Info.Title, "Test API")
	})
	t.Run("servers", func(t *testing.T) {
		assertLen(t, "Servers", len(doc.Servers), 1)
	})
	t.Run("paths", func(t *testing.T) {
		assertLen(t, "Paths", len(doc.Paths), 1)
		assertNotNil(t, "path /items", doc.Paths["/items"])
		assertNotNil(t, "GET /items", doc.Paths["/items"].Get)
	})
	t.Run("components", func(t *testing.T) {
		assertNotNil(t, "Components", doc.Components)
		assertLen(t, "Schemas", len(doc.Components.Schemas), 1)
		assertNotNil(t, "Item schema", doc.Components.Schemas["Item"])
	})
}

const generateTestContent = `package main

// !api 3.0.3
// !info "Test API" v1.0.0 "Description"
// !server https://api.test.com "Production"
func main() {}

// !GET /items -> getItems "Get items" #items
// !ok Item "Success"
func GetItems() {}

// !model "An item"
type Item struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`

func assertNotNil(t *testing.T, name string, val any) {
	if val == nil {
		t.Errorf("Expected %s to not be nil", name)
	}
}

func TestParser_SkipsVendorAndHiddenDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yaswag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create vendor directory with a Go file
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	vendorFile := filepath.Join(vendorDir, "dep.go")
	vendorContent := `package dep
// !api 3.0.0
// !info "Vendor API" v1.0.0 "Should be ignored"
func init() {}
`
	if err := os.WriteFile(vendorFile, []byte(vendorContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create hidden directory with a Go file
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	hiddenFile := filepath.Join(hiddenDir, "hidden.go")
	hiddenContent := `package hidden
// !api 3.0.0
// !info "Hidden API" v1.0.0 "Should be ignored"
func init() {}
`
	if err := os.WriteFile(hiddenFile, []byte(hiddenContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main file
	mainFile := filepath.Join(tmpDir, "main.go")
	mainContent := `package main
// !api 3.0.0
// !info "Main API" v1.0.0 "Should be included"
func main() {}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := New()
	if err := p.ParseDir(tmpDir); err != nil {
		t.Fatalf("ParseDir() error = %v", err)
	}

	spec := p.GetSpec()

	// Should have Main API info (not vendor or hidden)
	if spec.Info.Title != "Main API" {
		t.Errorf("Expected Main API title, got %v", spec.Info.Title)
	}
}

func TestParser_SkipsTestFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yaswag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test file
	testFile := filepath.Join(tmpDir, "api_test.go")
	testContent := `package main
// !api 3.0.0
// !info "Test File API" v1.0.0 "Should be ignored"
func TestSomething() {}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main file
	mainFile := filepath.Join(tmpDir, "api.go")
	mainContent := `package main
// !api 3.0.0
// !info "Main API" v1.0.0 "Should be included"
func main() {}
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := New()
	if err := p.ParseDir(tmpDir); err != nil {
		t.Fatalf("ParseDir() error = %v", err)
	}

	spec := p.GetSpec()

	// Should have Main API info (not from _test.go file)
	if spec.Info.Title != "Main API" {
		t.Errorf("Expected Main API title, got %v", spec.Info.Title)
	}
}

func TestParser_ArraySchemaRef(t *testing.T) {
	h := newTestHelper(t)
	defer h.cleanup()

	h.writeFile("api.go", arraySchemaTestContent)

	p := h.parse()
	doc := p.Generate()
	if doc == nil {
		t.Fatal("Expected document to be generated")
	}

	t.Run("users_array_suffix", func(t *testing.T) {
		verifyArrayResponse(t, doc, "/users", "User")
	})
	t.Run("items_array_prefix", func(t *testing.T) {
		verifyArrayResponse(t, doc, "/items", "Item")
	})
}

const arraySchemaTestContent = `package main

// !api 3.0.3
// !info "Test API" v1.0.0 "Test"
func main() {}

// !GET /users -> getUsers "Get users"
// !ok User[] "Success with array suffix"
func GetUsers() {}

// !GET /items -> getItems "Get items"
// !ok []Item "Success with array prefix"
func GetItems() {}
`

func verifyArrayResponse(t *testing.T, doc *openapi.Document, path, schemaName string) {
	pathItem := doc.Paths[path]
	if pathItem == nil || pathItem.Get == nil {
		t.Fatalf("Expected %s GET operation", path)
	}
	resp := pathItem.Get.Responses["200"]
	if resp == nil {
		t.Fatalf("Expected 200 response on %s", path)
	}
	schema := resp.Content["application/json"].Schema
	if schema.Type[0] != "array" {
		t.Errorf("Expected array type for %s response, got %v", path, schema.Type)
	}
	expectedRef := "#/components/schemas/" + schemaName
	if schema.Items == nil || schema.Items.Ref != expectedRef {
		t.Errorf("Expected items ref to %s schema", schemaName)
	}
}

func TestParser_HTTPMethods(t *testing.T) {
	h := newTestHelper(t)
	defer h.cleanup()

	h.writeFile("api.go", httpMethodsTestContent)

	p := h.parse()
	doc := p.Generate()
	if doc == nil {
		t.Fatal("Expected document to be generated")
	}

	t.Run("resource_path", func(t *testing.T) {
		verifyResourcePath(t, doc)
	})
	t.Run("resource_id_path", func(t *testing.T) {
		verifyResourceIDPath(t, doc)
	})
}

const httpMethodsTestContent = `package main

// !api 3.0.3
// !info "Test API" v1.0.0 "Test"
func main() {}

// !GET /resource -> getResource "Get"
// !ok - "Success"
func Get() {}

// !POST /resource -> createResource "Post"
// !ok - "Success"
func Post() {}

// !PUT /resource/{id} -> updateResource "Put"
// !ok - "Success"
func Put() {}

// !DELETE /resource/{id} -> deleteResource "Delete"
// !ok - "Success"
func Delete() {}

// !PATCH /resource/{id} -> patchResource "Patch"
// !ok - "Success"
func Patch() {}

// !OPTIONS /resource -> optionsResource "Options"
// !ok - "Success"
func Options() {}

// !HEAD /resource -> headResource "Head"
// !ok - "Success"
func Head() {}
`

func verifyResourcePath(t *testing.T, doc *openapi.Document) {
	resourcePath := doc.Paths["/resource"]
	if resourcePath == nil {
		t.Fatal("Expected /resource path")
	}
	assertNotNil(t, "GET on /resource", resourcePath.Get)
	assertNotNil(t, "POST on /resource", resourcePath.Post)
	assertNotNil(t, "OPTIONS on /resource", resourcePath.Options)
	assertNotNil(t, "HEAD on /resource", resourcePath.Head)
}

func verifyResourceIDPath(t *testing.T, doc *openapi.Document) {
	resourceIdPath := doc.Paths["/resource/{id}"]
	if resourceIdPath == nil {
		t.Fatal("Expected /resource/{id} path")
	}
	assertNotNil(t, "PUT on /resource/{id}", resourceIdPath.Put)
	assertNotNil(t, "DELETE on /resource/{id}", resourceIdPath.Delete)
	assertNotNil(t, "PATCH on /resource/{id}", resourceIdPath.Patch)
}

// TestParser_ParseDirFromCurrentDirectory tests parsing from the current directory using "."
// This reproduces the bug reported in issue #1
// currentDirTestSetup contains test fixture data
type currentDirTestSetup struct {
	t      *testing.T
	tmpDir string
	origWd string
}

// setupCurrentDirTest creates the test fixture
func setupCurrentDirTest(t *testing.T) *currentDirTestSetup {
	tmpDir, err := os.MkdirTemp("", "yaswag-test")
	if err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	return &currentDirTestSetup{t: t, tmpDir: tmpDir, origWd: origWd}
}

// cleanup restores the working directory and removes temp files
func (s *currentDirTestSetup) cleanup() {
	_ = os.Chdir(s.origWd)
	_ = os.RemoveAll(s.tmpDir)
}

// createSubdirWithFile creates a subdirectory and writes a file to it
func (s *currentDirTestSetup) createSubdirWithFile(subdir, filename, content string) {
	dir := filepath.Join(s.tmpDir, subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		s.t.Fatal(err)
	}
}

// changeToTmpDir changes to the temp directory
func (s *currentDirTestSetup) changeToTmpDir() {
	if err := os.Chdir(s.tmpDir); err != nil {
		s.t.Fatal(err)
	}
}

func TestParser_ParseDirFromCurrentDirectory(t *testing.T) {
	setup := setupCurrentDirTest(t)
	defer setup.cleanup()

	setup.createSubdirWithFile("docs", "doc.go", currentDirDocContent)
	setup.createSubdirWithFile("handlers", "handlers.go", currentDirHandlersContent)
	setup.changeToTmpDir()

	p := New()
	if err := p.ParseDir("."); err != nil {
		t.Fatalf("ParseDir(\".\") error = %v", err)
	}

	spec := p.GetSpec()
	verifyCurrentDirSpec(t, spec)
}

const currentDirDocContent = `package docs

// API Documentation
//
// !api 3.0.3
// !info "Test API" v1.0.0 "A test API for issue #1"
// !server https://api.example.com "Production"
`

const currentDirHandlersContent = `package handlers

// GetUsers retrieves all users
//
// !GET /users -> getUsers "Get all users" #users
// !ok 200 - "Success"
func GetUsers() {}
`

// verifyCurrentDirSpec validates the parsed spec from current directory
func verifyCurrentDirSpec(t *testing.T, spec *SpecData) {
	t.Helper()
	if spec.Info == nil || spec.Info.Title == "" {
		t.Fatal("Expected to find YaSwag annotations when parsing from current directory")
	}
	assertEqual(t, "Info.Title", spec.Info.Title, "Test API")
	assertEqual(t, "Info.Version", spec.Info.Version, "1.0.0")
	assertLen(t, "Operations", len(spec.Operations), 1)
	if len(spec.Operations) > 0 {
		assertEqual(t, "Operation.Path", spec.Operations[0].Path, "/users")
		assertEqual(t, "Operation.Method", spec.Operations[0].Method, "GET")
	}
}
