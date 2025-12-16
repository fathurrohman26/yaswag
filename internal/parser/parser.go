package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/fathurrohman26/yaswag/pkg/openapi"
)

// Parser extracts OpenAPI documentation from Go source code using YaSwag's eccentric annotations.
type Parser struct {
	fset             *token.FileSet
	annotationParser *AnnotationParser

	// Parsed specification data
	spec *SpecData

	// Global schemas (from !model annotations)
	globalSchemas map[string]*SchemaData
}

// SpecData holds all parsed data for an OpenAPI specification.
type SpecData struct {
	Version      string
	Info         *openapi.Info
	Servers      []openapi.Server
	Tags         []openapi.Tag
	Operations   []OperationData
	Schemas      map[string]*SchemaData
	Securities   map[string]*openapi.SecurityScheme
	ExternalDocs *openapi.ExternalDocumentation
	Links        []LinkData // Additional links for description
}

// LinkData holds a link label and URL.
type LinkData struct {
	Label string
	URL   string
}

// OperationData holds parsed operation data.
type OperationData struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Parameters  []*openapi.Parameter
	RequestBody *openapi.RequestBody
	Responses   openapi.Responses
	Security    []openapi.SecurityRequirement
}

// SchemaData holds parsed schema data with examples.
type SchemaData struct {
	Name        string
	Description string
	Schema      *openapi.Schema
	Examples    map[string]any
}

// New creates a new Parser instance.
func New() *Parser {
	return &Parser{
		fset:             token.NewFileSet(),
		annotationParser: NewAnnotationParser(),
		spec: &SpecData{
			Version:    "3.0.3",
			Info:       &openapi.Info{},
			Schemas:    make(map[string]*SchemaData),
			Securities: make(map[string]*openapi.SecurityScheme),
		},
		globalSchemas: make(map[string]*SchemaData),
	}
}

// ParseDir parses all Go files in the given directory recursively.
func (p *Parser) ParseDir(dir string) error {
	// Clean the path to normalize it
	root := filepath.Clean(dir)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			// Don't skip the root directory itself
			if path != root {
				// Skip vendor, hidden directories, and testdata
				if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		// Skip non-Go files and test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return p.parseFile(path)
	})
}

func (p *Parser) parseFile(path string) error {
	f, err := parser.ParseFile(p.fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Parse all comment groups for API-level annotations
	for _, cg := range f.Comments {
		p.parseCommentGroup(cg)
	}

	// Parse function declarations for operation annotations
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			p.parseFuncDecl(fn)
		}
	}

	// Parse type declarations for schema annotations
	for _, decl := range f.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			p.parseTypeDecl(genDecl)
		}
	}

	return nil
}

func (p *Parser) parseCommentGroup(cg *ast.CommentGroup) {
	if cg == nil {
		return
	}
	text := cg.Text()

	annotations := p.annotationParser.Parse(text)
	for _, a := range annotations {
		p.handleAnnotation(a)
	}
}

func (p *Parser) handleAnnotation(a Annotation) {
	handlers := map[AnnotationType]func(Annotation){
		AnnotationAPI:          func(a Annotation) { p.spec.Version = GetAPI(a).Version },
		AnnotationInfo:         p.handleInfo,
		AnnotationContact:      p.handleContact,
		AnnotationLicense:      p.handleLicense,
		AnnotationServer:       p.handleServer,
		AnnotationTag:          p.handleTag,
		AnnotationTOS:          func(a Annotation) { p.spec.Info.TermsOfService = GetTOS(a).URL },
		AnnotationSecurity:     p.handleSecurity,
		AnnotationScope:        p.handleScope,
		AnnotationExternalDocs: p.handleExternalDocs,
		AnnotationLink:         p.handleLink,
	}
	if handler, ok := handlers[a.Type]; ok {
		handler(a)
	}
}

func (p *Parser) handleInfo(a Annotation) {
	info := GetInfo(a)
	p.spec.Info.Title = info.Title
	p.spec.Info.Version = info.Version
	p.spec.Info.Description = info.Description
}

func (p *Parser) handleContact(a Annotation) {
	contact := GetContact(a)
	p.spec.Info.Contact = &openapi.Contact{
		Name:  contact.Name,
		Email: contact.Email,
		URL:   contact.URL,
	}
}

func (p *Parser) handleLicense(a Annotation) {
	license := GetLicense(a)
	p.spec.Info.License = &openapi.License{
		Name: license.Name,
		URL:  license.URL,
	}
}

func (p *Parser) handleServer(a Annotation) {
	server := GetServer(a)
	p.spec.Servers = append(p.spec.Servers, openapi.Server{
		URL:         server.URL,
		Description: server.Description,
	})
}

func (p *Parser) handleTag(a Annotation) {
	tag := GetTag(a)
	p.spec.Tags = append(p.spec.Tags, openapi.Tag{
		Name:        tag.Name,
		Description: tag.Description,
	})
}

func (p *Parser) handleSecurity(a Annotation) {
	sec := GetSecurity(a)
	scheme := &openapi.SecurityScheme{Description: sec.Description}

	switch sec.Type {
	case "apiKey":
		p.configureAPIKeySecurity(scheme, sec)
	case "oauth2":
		p.configureOAuth2Security(scheme, sec)
	case "http":
		scheme.Type = "http"
		scheme.Scheme = sec.Location
	case "openIdConnect":
		scheme.Type = "openIdConnect"
		scheme.OpenIDConnectURL = sec.URL
	}
	p.spec.Securities[sec.Name] = scheme
}

func (p *Parser) configureAPIKeySecurity(scheme *openapi.SecurityScheme, sec ParsedSecurity) {
	scheme.Type = "apiKey"
	scheme.In = sec.Location
	scheme.Name = sec.Name
	if sec.URL != "" {
		scheme.Name = sec.URL
	}
}

func (p *Parser) configureOAuth2Security(scheme *openapi.SecurityScheme, sec ParsedSecurity) {
	scheme.Type = "oauth2"
	scheme.Flows = &openapi.OAuthFlows{}

	switch sec.Location {
	case "implicit":
		scheme.Flows.Implicit = &openapi.OAuthFlow{AuthorizationURL: sec.URL, Scopes: map[string]string{}}
	case "password":
		scheme.Flows.Password = &openapi.OAuthFlow{TokenURL: sec.URL, Scopes: map[string]string{}}
	case "clientCredentials":
		scheme.Flows.ClientCredentials = &openapi.OAuthFlow{TokenURL: sec.URL, Scopes: map[string]string{}}
	case "authorizationCode":
		scheme.Flows.AuthorizationCode = &openapi.OAuthFlow{AuthorizationURL: sec.URL, TokenURL: sec.URL, Scopes: map[string]string{}}
	default:
		if sec.URL != "" {
			scheme.Flows.Implicit = &openapi.OAuthFlow{AuthorizationURL: sec.URL, Scopes: map[string]string{}}
		}
	}
}

func (p *Parser) handleScope(a Annotation) {
	scope := GetScope(a)
	scheme, ok := p.spec.Securities[scope.Security]
	if !ok || scheme.Flows == nil {
		return
	}
	addScopeToFlow(scheme.Flows.Implicit, scope.Name, scope.Description)
	addScopeToFlow(scheme.Flows.Password, scope.Name, scope.Description)
	addScopeToFlow(scheme.Flows.ClientCredentials, scope.Name, scope.Description)
	addScopeToFlow(scheme.Flows.AuthorizationCode, scope.Name, scope.Description)
}

func addScopeToFlow(flow *openapi.OAuthFlow, name, description string) {
	if flow == nil {
		return
	}
	if flow.Scopes == nil {
		flow.Scopes = make(map[string]string)
	}
	flow.Scopes[name] = description
}

func (p *Parser) handleExternalDocs(a Annotation) {
	extDocs := GetExternalDocs(a)
	p.spec.ExternalDocs = &openapi.ExternalDocumentation{
		URL:         extDocs.URL,
		Description: extDocs.Description,
	}
}

func (p *Parser) handleLink(a Annotation) {
	link := GetLink(a)
	p.spec.Links = append(p.spec.Links, LinkData(link))
}

func (p *Parser) parseFuncDecl(fn *ast.FuncDecl) {
	if fn.Doc == nil {
		return
	}

	text := fn.Doc.Text()
	if !strings.Contains(text, "!") {
		return
	}

	annotations := p.annotationParser.Parse(text)
	if len(annotations) == 0 {
		return
	}

	op := p.parseOperationAnnotations(annotations)
	if op != nil {
		p.spec.Operations = append(p.spec.Operations, *op)
	}
}

func (p *Parser) parseOperationAnnotations(annotations []Annotation) *OperationData {
	op := &OperationData{Responses: make(openapi.Responses)}

	for _, a := range annotations {
		p.applyOperationAnnotation(op, a)
	}

	if op.Method == "" || op.Path == "" {
		return nil
	}
	return op
}

func (p *Parser) applyOperationAnnotation(op *OperationData, a Annotation) {
	switch a.Type {
	case AnnotationRoute:
		p.applyRouteAnnotation(op, a)
	case AnnotationQuery, AnnotationPath, AnnotationHeader:
		p.applyParamAnnotation(op, a)
	case AnnotationBody:
		p.applyBodyAnnotation(op, a)
	case AnnotationOK, AnnotationError:
		p.applyResponseAnnotation(op, a)
	case AnnotationSecure:
		p.applySecureAnnotation(op, a)
	}
}

func (p *Parser) applyRouteAnnotation(op *OperationData, a Annotation) {
	route := GetRoute(a)
	op.Method = route.Method
	op.Path = route.Path
	op.OperationID = route.OperationID
	op.Summary = route.Summary
	op.Tags = route.Tags
}

func (p *Parser) applyParamAnnotation(op *OperationData, a Annotation) {
	param := GetParam(a)
	op.Parameters = append(op.Parameters, &openapi.Parameter{
		Name:        param.Name,
		In:          openapi.ParameterLocation(param.In),
		Description: param.Description,
		Required:    param.Required || param.In == "path",
		Schema:      p.typeToSchema(param.Type),
		Example:     parseDefaultValue(param.Default),
	})
}

func (p *Parser) applyBodyAnnotation(op *OperationData, a Annotation) {
	body := GetBody(a)
	op.RequestBody = &openapi.RequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content: map[string]openapi.MediaType{
			"application/json": {Schema: p.parseSchemaRef(body.Schema)},
		},
	}
}

func (p *Parser) applyResponseAnnotation(op *OperationData, a Annotation) {
	resp := GetResponse(a)
	response := &openapi.Response{Description: resp.Description}
	if resp.Schema != "" && resp.Schema != "-" && resp.Schema != "nil" && resp.Schema != "none" {
		response.Content = map[string]openapi.MediaType{
			"application/json": {Schema: p.parseSchemaRef(resp.Schema)},
		}
	}
	op.Responses[resp.Status] = response
}

func (p *Parser) applySecureAnnotation(op *OperationData, a Annotation) {
	secure := GetSecure(a)
	for _, name := range secure.Names {
		op.Security = append(op.Security, openapi.SecurityRequirement{name: []string{}})
	}
}

func (p *Parser) parseTypeDecl(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		var docText string
		if decl.Doc != nil {
			docText = decl.Doc.Text()
		}

		// Only process types with !model annotation
		if !strings.Contains(docText, "!model") {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		annotations := p.annotationParser.Parse(docText)
		for _, a := range annotations {
			if a.Type == AnnotationModel {
				model := GetModel(a)
				schemaData := &SchemaData{
					Name:        typeSpec.Name.Name,
					Description: model.Description,
					Schema:      p.structToSchema(structType, docText),
					Examples:    make(map[string]any),
				}
				schemaData.Schema.Description = model.Description

				// Parse field annotations from struct fields
				p.parseStructFieldAnnotations(structType, schemaData)

				// Store schema globally by struct type name
				p.globalSchemas[typeSpec.Name.Name] = schemaData
			}
		}
	}
}

func (p *Parser) parseStructFieldAnnotations(structType *ast.StructType, schemaData *SchemaData) {
	for _, field := range structType.Fields.List {
		jsonName := p.getFieldJSONName(field)
		if jsonName == "" {
			continue
		}
		p.applyFieldAnnotations(field, jsonName, schemaData)
	}
}

func (p *Parser) getFieldJSONName(field *ast.Field) string {
	if len(field.Names) == 0 {
		return ""
	}
	jsonName := getJSONTagName(field)
	if jsonName == "-" {
		return ""
	}
	if jsonName == "" {
		return field.Names[0].Name
	}
	return jsonName
}

func (p *Parser) applyFieldAnnotations(field *ast.Field, jsonName string, schemaData *SchemaData) {
	if field.Doc == nil {
		return
	}
	annotations := p.annotationParser.Parse(field.Doc.Text())
	for _, a := range annotations {
		if a.Type == AnnotationField {
			p.applyFieldInfo(jsonName, GetField(a), schemaData)
		}
	}
}

func (p *Parser) applyFieldInfo(jsonName string, fieldInfo ParsedField, schemaData *SchemaData) {
	propSchema, ok := schemaData.Schema.Properties[jsonName]
	if !ok {
		return
	}
	propSchema.Description = fieldInfo.Description
	if fieldInfo.Example != "" {
		propSchema.Example = parseValue(fieldInfo.Example)
	}
	if fieldInfo.Required && !slices.Contains(schemaData.Schema.Required, jsonName) {
		schemaData.Schema.Required = append(schemaData.Schema.Required, jsonName)
	}
}

func (p *Parser) structToSchema(structType *ast.StructType, docText string) *openapi.Schema {
	schema := &openapi.Schema{
		Type:       openapi.NewSchemaType(openapi.TypeObject),
		Properties: make(map[string]*openapi.Schema),
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		fieldName := field.Names[0].Name
		jsonName := getJSONTagName(field)
		if jsonName == "-" {
			continue
		}
		if jsonName == "" {
			jsonName = fieldName
		}

		fieldSchema := p.fieldToSchema(field)
		schema.Properties[jsonName] = fieldSchema

		// Add to required if not omitempty
		if !strings.Contains(getJSONTag(field), "omitempty") {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema
}

func (p *Parser) fieldToSchema(field *ast.Field) *openapi.Schema {
	desc := p.getFieldDescription(field)
	schema := p.astTypeToSchema(field.Type)
	if desc != "" && schema.Description == "" {
		schema.Description = desc
	}
	return schema
}

func (p *Parser) getFieldDescription(field *ast.Field) string {
	if field.Doc != nil {
		return cleanDescription(field.Doc.Text())
	}
	if field.Comment != nil {
		return cleanDescription(field.Comment.Text())
	}
	return ""
}

func (p *Parser) astTypeToSchema(expr ast.Expr) *openapi.Schema {
	switch t := expr.(type) {
	case *ast.Ident:
		return p.typeToSchema(t.Name)
	case *ast.StarExpr:
		return p.starExprToSchema(t)
	case *ast.ArrayType:
		return p.arrayTypeToSchema(t)
	case *ast.MapType:
		return p.mapTypeToSchema(t)
	case *ast.SelectorExpr:
		return p.selectorExprToSchema(t)
	default:
		return &openapi.Schema{}
	}
}

func (p *Parser) starExprToSchema(t *ast.StarExpr) *openapi.Schema {
	if ident, ok := t.X.(*ast.Ident); ok {
		schema := p.typeToSchema(ident.Name)
		schema.Nullable = true
		return schema
	}
	return &openapi.Schema{}
}

func (p *Parser) arrayTypeToSchema(t *ast.ArrayType) *openapi.Schema {
	schema := &openapi.Schema{Type: openapi.NewSchemaType(openapi.TypeArray)}
	schema.Items = p.extractArrayItemSchema(t.Elt)
	return schema
}

func (p *Parser) extractArrayItemSchema(elt ast.Expr) *openapi.Schema {
	if ident, ok := elt.(*ast.Ident); ok {
		return p.typeToSchema(ident.Name)
	}
	if star, ok := elt.(*ast.StarExpr); ok {
		if ident, ok := star.X.(*ast.Ident); ok {
			return p.typeToSchema(ident.Name)
		}
	}
	return nil
}

func (p *Parser) mapTypeToSchema(t *ast.MapType) *openapi.Schema {
	schema := &openapi.Schema{Type: openapi.NewSchemaType(openapi.TypeObject)}
	if valIdent, ok := t.Value.(*ast.Ident); ok {
		schema.AdditionalProperties = p.typeToSchema(valIdent.Name)
	}
	return schema
}

func (p *Parser) selectorExprToSchema(t *ast.SelectorExpr) *openapi.Schema {
	x, ok := t.X.(*ast.Ident)
	if !ok {
		return &openapi.Schema{}
	}
	if x.Name == "time" && t.Sel.Name == "Time" {
		return &openapi.Schema{Type: openapi.NewSchemaType(openapi.TypeString), Format: "date-time"}
	}
	if x.Name == "uuid" && t.Sel.Name == "UUID" {
		return &openapi.Schema{Type: openapi.NewSchemaType(openapi.TypeString), Format: "uuid"}
	}
	return &openapi.Schema{}
}

// schemaTypeInfo holds OpenAPI schema type and format.
type schemaTypeInfo struct {
	schemaType string
	format     string
}

// typeSchemaMapping maps Go types to OpenAPI schema types and formats.
var typeSchemaMapping = map[string]schemaTypeInfo{
	"string":      {openapi.TypeString, ""},
	"int":         {openapi.TypeInteger, "int32"},
	"int8":        {openapi.TypeInteger, "int32"},
	"int16":       {openapi.TypeInteger, "int32"},
	"int32":       {openapi.TypeInteger, "int32"},
	"integer":     {openapi.TypeInteger, "int32"},
	"int64":       {openapi.TypeInteger, "int64"},
	"uint":        {openapi.TypeInteger, "int32"},
	"uint8":       {openapi.TypeInteger, "int32"},
	"uint16":      {openapi.TypeInteger, "int32"},
	"uint32":      {openapi.TypeInteger, "int32"},
	"uint64":      {openapi.TypeInteger, "int64"},
	"float32":     {openapi.TypeNumber, "float"},
	"float":       {openapi.TypeNumber, "float"},
	"float64":     {openapi.TypeNumber, "double"},
	"double":      {openapi.TypeNumber, "double"},
	"number":      {openapi.TypeNumber, "double"},
	"bool":        {openapi.TypeBoolean, ""},
	"boolean":     {openapi.TypeBoolean, ""},
	"byte":        {openapi.TypeString, "byte"},
	"any":         {openapi.TypeObject, ""},
	"interface{}": {openapi.TypeObject, ""},
	"object":      {openapi.TypeObject, ""},
	"array":       {openapi.TypeArray, ""},
}

func (p *Parser) typeToSchema(typeName string) *openapi.Schema {
	if info, ok := typeSchemaMapping[typeName]; ok {
		return &openapi.Schema{Type: openapi.NewSchemaType(info.schemaType), Format: info.format}
	}
	return openapi.RefTo(typeName)
}

func (p *Parser) parseSchemaRef(ref string) *openapi.Schema {
	// Check if it's an array type like []User or User[]
	if strings.HasPrefix(ref, "[]") {
		itemType := strings.TrimPrefix(ref, "[]")
		return &openapi.Schema{
			Type:  openapi.NewSchemaType(openapi.TypeArray),
			Items: openapi.RefTo(itemType),
		}
	}
	if strings.HasSuffix(ref, "[]") {
		itemType := strings.TrimSuffix(ref, "[]")
		return &openapi.Schema{
			Type:  openapi.NewSchemaType(openapi.TypeArray),
			Items: openapi.RefTo(itemType),
		}
	}
	return openapi.RefTo(ref)
}

// GetSpec returns the parsed specification with global schemas merged.
func (p *Parser) GetSpec() *SpecData {
	// Merge global schemas into spec
	for name, schemaData := range p.globalSchemas {
		if _, exists := p.spec.Schemas[name]; !exists {
			p.spec.Schemas[name] = schemaData
		}
	}
	return p.spec
}

// GetGlobalSchemas returns all parsed global schemas.
func (p *Parser) GetGlobalSchemas() map[string]*SchemaData {
	return p.globalSchemas
}

// Generate generates an OpenAPI document from the parsed specification.
func (p *Parser) Generate() *openapi.Document {
	spec := p.GetSpec()
	return p.generateDocument(spec)
}

func (p *Parser) generateDocument(spec *SpecData) *openapi.Document {
	doc := &openapi.Document{
		OpenAPI:      spec.Version,
		Info:         p.buildInfo(spec),
		Servers:      spec.Servers,
		Tags:         spec.Tags,
		Paths:        make(openapi.Paths),
		ExternalDocs: spec.ExternalDocs,
	}

	p.addPaths(doc, spec.Operations)
	p.addComponents(doc, spec)
	return doc
}

func (p *Parser) buildInfo(spec *SpecData) openapi.Info {
	info := *spec.Info
	if len(spec.Links) > 0 {
		info.Description += "\n\nSome useful links:\n"
		for _, link := range spec.Links {
			info.Description += fmt.Sprintf("- [%s](%s)\n", link.Label, link.URL)
		}
	}
	return info
}

func (p *Parser) addPaths(doc *openapi.Document, operations []OperationData) {
	for _, op := range operations {
		pathItem := doc.Paths[op.Path]
		if pathItem == nil {
			pathItem = &openapi.PathItem{}
			doc.Paths[op.Path] = pathItem
		}
		setPathOperation(pathItem, op)
	}
}

func setPathOperation(pathItem *openapi.PathItem, op OperationData) {
	operation := &openapi.Operation{
		OperationID: op.OperationID,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        op.Tags,
		Deprecated:  op.Deprecated,
		Parameters:  op.Parameters,
		RequestBody: op.RequestBody,
		Responses:   op.Responses,
		Security:    op.Security,
	}

	switch op.Method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "OPTIONS":
		pathItem.Options = operation
	case "HEAD":
		pathItem.Head = operation
	case "TRACE":
		pathItem.Trace = operation
	}
}

func (p *Parser) addComponents(doc *openapi.Document, spec *SpecData) {
	hasSchemas := len(spec.Schemas) > 0 || len(p.globalSchemas) > 0
	hasSecurities := len(spec.Securities) > 0

	if !hasSchemas && !hasSecurities {
		return
	}

	doc.Components = &openapi.Components{}
	if hasSchemas {
		doc.Components.Schemas = p.buildSchemas(spec)
	}
	if hasSecurities {
		doc.Components.SecuritySchemes = spec.Securities
	}
}

func (p *Parser) buildSchemas(spec *SpecData) map[string]*openapi.Schema {
	schemas := make(map[string]*openapi.Schema)
	for name, schemaData := range spec.Schemas {
		schemas[name] = schemaData.Schema
	}
	for name, schemaData := range p.globalSchemas {
		if _, exists := schemas[name]; !exists {
			schemas[name] = schemaData.Schema
		}
	}
	return schemas
}

// Helper functions

func getJSONTagName(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	tag := field.Tag.Value
	tag = strings.Trim(tag, "`")

	jsonTagRegex := regexp.MustCompile(`json:"([^"]*)"`)
	match := jsonTagRegex.FindStringSubmatch(tag)
	if len(match) < 2 {
		return ""
	}

	parts := strings.Split(match[1], ",")
	return parts[0]
}

func getJSONTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	tag := field.Tag.Value
	tag = strings.Trim(tag, "`")

	jsonTagRegex := regexp.MustCompile(`json:"([^"]*)"`)
	match := jsonTagRegex.FindStringSubmatch(tag)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func cleanDescription(desc string) string {
	lines := strings.Split(desc, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip annotation lines
		if strings.HasPrefix(trimmed, "!") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}
	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

func parseDefaultValue(value string) any {
	if value == "" {
		return nil
	}
	return parseValue(value)
}
