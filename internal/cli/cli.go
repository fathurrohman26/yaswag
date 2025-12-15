package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fathurrohman26/yaswag/internal/parser"
	"github.com/fathurrohman26/yaswag/pkg/mcp"
	"github.com/fathurrohman26/yaswag/pkg/openapi"
	"github.com/fathurrohman26/yaswag/pkg/output"
	"github.com/fathurrohman26/yaswag/pkg/swaggerui"
	"github.com/fathurrohman26/yaswag/pkg/validator"
)

type CLI struct {
	info struct {
		version string
		commit  string
		date    string
	}
}

type Option func(*CLI)

func New(opts ...Option) *CLI {
	cli := &CLI{}
	for _, opt := range opts {
		opt(cli)
	}

	return cli
}

func WithVersionInfo(version, commit, date string) Option {
	return func(c *CLI) {
		c.info.version = version
		c.info.commit = commit
		c.info.date = date
	}
}

func (c *CLI) Run() error {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println(c.Help())
		return nil
	}

	cmd := args[0]

	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(c.Version())
		return nil
	case "help", "-h", "--help":
		fmt.Println(c.Help())
		return nil
	case "generate":
		return c.runGenerate(args[1:])
	case "validate":
		return c.runValidate(args[1:])
	case "format":
		return c.runFormat(args[1:])
	case "serve":
		return c.runServe(args[1:])
	case "editor":
		return c.runEditor(args[1:])
	case "mcp":
		return c.runMCP(args[1:])
	}

	return fmt.Errorf("unknown command: %s", cmd)
}

func (c *CLI) runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	source := fs.String("source", ".", "Source directory to scan for annotations")
	format := fs.String("format", "yaml", "Output format (json or yaml)")
	outputPath := fs.String("output", "", "Output file path (empty for stdout)")
	pretty := fs.Int("pretty", 2, "Indentation spaces for pretty printing")
	showHelp := fs.Bool("help", false, "Show help for generate command")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		fmt.Println(c.GenerateHelp())
		return nil
	}

	openAPIDoc, err := c.parseAndGenerate(*source)
	if err != nil {
		return err
	}

	data, err := c.formatOutput(openAPIDoc, *format, *pretty)
	if err != nil {
		return err
	}

	return c.writeOutput(*outputPath, data, "OpenAPI specification")
}

func (c *CLI) parseAndGenerate(source string) (*openapi.Document, error) {
	p := parser.New()
	if err := p.ParseDir(source); err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	spec := p.GetSpec()
	if spec.Info == nil || spec.Info.Title == "" {
		return nil, fmt.Errorf("no YaSwag annotations found in %s", source)
	}

	doc := p.Generate()
	if doc == nil {
		return nil, fmt.Errorf("failed to generate OpenAPI document")
	}
	return doc, nil
}

func (c *CLI) formatOutput(doc *openapi.Document, format string, pretty int) ([]byte, error) {
	outputFormat, err := output.ParseFormat(format)
	if err != nil {
		return nil, err
	}

	formatter := output.NewFormatter(output.Options{
		Format: outputFormat,
		Indent: pretty,
		Pretty: pretty > 0,
	})

	data, err := formatter.Format(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to format output: %w", err)
	}
	return data, nil
}

func (c *CLI) runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	input := fs.String("input", "", "Input file path, URL, or - for stdin")
	showHelp := fs.Bool("help", false, "Show help for validate command")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		fmt.Println(c.ValidateHelp())
		return nil
	}

	v := validator.New()
	result, err := c.validateInput(v, *input)
	if err != nil {
		return err
	}

	fmt.Print(validator.FormatResult(result))
	if !result.Valid {
		os.Exit(1)
	}
	return nil
}

func (c *CLI) validateInput(v *validator.Validator, input string) (*validator.ValidationResult, error) {
	if isURL(input) {
		return v.ValidateInput(input)
	}

	stdinRes, err := readFromStdinOrFile(input, true)
	if err != nil {
		return nil, err
	}

	if stdinRes.fromStdin {
		return v.Validate(stdinRes.data)
	}
	return v.ValidateInput(input)
}

// stdinResult holds the result of reading from stdin
type stdinResult struct {
	data      []byte
	fromStdin bool
}

// readFromStdinOrFile reads data from stdin or a file based on the input flag.
// Returns data, whether it came from stdin, and any error.
func readFromStdinOrFile(input string, requireInput bool) (*stdinResult, error) {
	if input == "-" || input == "" {
		return readFromStdin(input, requireInput)
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}
	return &stdinResult{data: data, fromStdin: false}, nil
}

func readFromStdin(input string, requireInput bool) (*stdinResult, error) {
	stat, statErr := os.Stdin.Stat()
	if statErr != nil {
		if input == "" && requireInput {
			return nil, fmt.Errorf("--input is required (use - for stdin)")
		}
		return nil, fmt.Errorf("failed to check stdin: %w", statErr)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		if len(data) == 0 && requireInput {
			return nil, fmt.Errorf("no data received from stdin")
		}
		return &stdinResult{data: data, fromStdin: true}, nil
	}
	if input == "-" {
		return nil, fmt.Errorf("no data available on stdin")
	}
	if requireInput {
		return nil, fmt.Errorf("--input is required (use - for stdin)")
	}
	return &stdinResult{fromStdin: false}, nil
}

func (c *CLI) runFormat(args []string) error {
	fs := flag.NewFlagSet("format", flag.ExitOnError)
	input := fs.String("input", "", "Input file path or - for stdin")
	outputPath := fs.String("output", "", "Output file path (empty for stdout)")
	format := fs.String("format", "", "Output format (json or yaml, auto-detected from extension if not specified)")
	pretty := fs.Int("pretty", 4, "Indentation spaces for pretty printing")
	showHelp := fs.Bool("help", false, "Show help for format command")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		fmt.Println(c.FormatHelp())
		return nil
	}

	result, err := readFromStdinOrFile(*input, true)
	if err != nil {
		return err
	}

	outputFormat := c.determineOutputFormat(*format, *outputPath, *input, result.fromStdin)

	v := validator.New()
	valResult, err := v.Validate(result.data)
	if err != nil {
		return err
	}
	c.printValidationWarnings(valResult)

	formatted, err := formatSpec(result.data, outputFormat, *pretty)
	if err != nil {
		return fmt.Errorf("failed to format: %w", err)
	}

	return c.writeOutput(*outputPath, formatted, "Formatted specification")
}

func (c *CLI) determineOutputFormat(format, outputPath, input string, fromStdin bool) output.Format {
	if format != "" {
		if f, err := output.ParseFormat(format); err == nil {
			return f
		}
	}
	if outputPath != "" {
		return output.DetectFormat(outputPath)
	}
	if fromStdin {
		return output.FormatYAML
	}
	return output.DetectFormat(input)
}

func (c *CLI) printValidationWarnings(result *validator.ValidationResult) {
	if result.Valid {
		return
	}
	fmt.Fprintln(os.Stderr, "Warning: Input has validation errors")
	for _, e := range result.Errors {
		fmt.Fprintf(os.Stderr, "  - %s\n", e.Error())
	}
}

func (c *CLI) writeOutput(outputPath string, data []byte, msgPrefix string) error {
	if outputPath != "" {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("%s written to %s\n", msgPrefix, outputPath)
		return nil
	}
	fmt.Print(string(data))
	return nil
}

func (c *CLI) runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	input := fs.String("input", "", "Input file path, URL, or - for stdin")
	port := fs.Int("port", 8080, "Port to serve on")
	showHelp := fs.Bool("help", false, "Show help for serve command")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		fmt.Println(c.ServeHelp())
		return nil
	}

	server := swaggerui.NewServer(*port)
	if err := c.setServerSpec(server, *input, true); err != nil {
		return err
	}
	return server.Serve()
}

type specSetter interface {
	SetSpecFromData(data []byte)
	SetSpecFromURL(url string)
	SetSpecFromFile(path string) error
}

func (c *CLI) setServerSpec(server specSetter, input string, requireInput bool) error {
	if isURL(input) {
		server.SetSpecFromURL(input)
		return nil
	}

	result, err := readFromStdinOrFile(input, requireInput)
	if err != nil {
		return err
	}

	if result.fromStdin && len(result.data) > 0 {
		server.SetSpecFromData(result.data)
	} else if !result.fromStdin && input != "" {
		return server.SetSpecFromFile(input)
	}
	return nil
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func (c *CLI) runEditor(args []string) error {
	fs := flag.NewFlagSet("editor", flag.ExitOnError)
	input := fs.String("input", "", "Input file path, URL, or - for stdin (optional)")
	port := fs.Int("port", 8080, "Port to serve on")
	showHelp := fs.Bool("help", false, "Show help for editor command")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		fmt.Println(c.EditorHelp())
		return nil
	}

	server := swaggerui.NewEditorServer(*port)
	// Editor doesn't require input - can launch in create mode
	if err := c.setServerSpec(server, *input, false); err != nil {
		return err
	}
	return server.Serve()
}

func (c *CLI) runMCP(args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	showHelp := fs.Bool("help", false, "Show help for mcp command")
	skipValidation := fs.Bool("skip-validation", false, "Skip spec validation before starting")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showHelp {
		fmt.Println(c.MCPHelp())
		return nil
	}

	// Remaining args are spec file paths
	specPaths := fs.Args()

	if len(specPaths) == 0 {
		return fmt.Errorf("no specification provided: yaswag mcp <spec-file> [spec-file...]")
	}

	// Validate spec files before starting
	if !*skipValidation {
		result, err := mcp.ValidateSpecFile(specPaths[0])
		if err != nil {
			return fmt.Errorf("validation error: %w", err)
		}
		c.printMCPValidationResult(result)
		if !result.Valid {
			return fmt.Errorf("spec validation failed")
		}
	}

	server := mcp.NewServer(specPaths)
	return server.Run()
}

func (c *CLI) printMCPValidationResult(result *validator.ValidationResult) {
	if result.Valid && len(result.Warnings) == 0 {
		fmt.Fprintln(os.Stderr, "Spec validation: OK")
		return
	}

	if !result.Valid {
		fmt.Fprintln(os.Stderr, "Spec validation: FAILED")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  ERROR: %s\n", e.Message)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Spec validation: OK (with warnings)")
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "  WARNING: %s\n", w.Message)
	}
}

func (c *CLI) Version() string {
	return fmt.Sprintf("yaswag version %s (commit: %s, built: %s)", c.info.version, c.info.commit, c.info.date)
}

func (c *CLI) Help() string {
	help := strings.Builder{}
	help.WriteString("YaSwag - Yet Another Swagger Tool for Go\n")
	help.WriteString("Generate OpenAPI specifications from Go annotations.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag [command] [options]\n\n")
	help.WriteString("Commands:\n")
	help.WriteString("  generate    Generate OpenAPI specification from Go annotations\n")
	help.WriteString("  validate    Validate an existing OpenAPI specification\n")
	help.WriteString("  format      Format an OpenAPI specification file\n")
	help.WriteString("  serve       Serve OpenAPI specification with Swagger UI\n")
	help.WriteString("  editor      Launch Swagger Editor for creating/editing specifications\n")
	help.WriteString("  mcp         Start MCP server for AI assistant integration\n")
	help.WriteString("  version     Show version information\n")
	help.WriteString("  help        Show this help message\n\n")
	help.WriteString("Use 'yaswag [command] --help' for more information about a command.\n")
	return help.String()
}

func (c *CLI) GenerateHelp() string {
	help := strings.Builder{}
	help.WriteString("Generate OpenAPI specification from Go annotations.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag generate [options]\n\n")
	help.WriteString("Options:\n")
	help.WriteString("  --source <path>   Source directory to scan for annotations (default: .)\n")
	help.WriteString("  --format <type>   Output format: json or yaml (default: yaml)\n")
	help.WriteString("  --output <path>   Output file path (empty for stdout)\n")
	help.WriteString("  --pretty <n>      Indentation spaces (default: 2)\n")
	help.WriteString("  --help            Show this help message\n\n")
	help.WriteString("Examples:\n")
	help.WriteString("  yaswag generate --source ./api --format yaml --output ./swagger.yaml\n")
	help.WriteString("  yaswag generate --source . --format json\n")
	return help.String()
}

func (c *CLI) ValidateHelp() string {
	help := strings.Builder{}
	help.WriteString("Validate an existing OpenAPI specification.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag validate [options]\n")
	help.WriteString("  <command> | yaswag validate\n\n")
	help.WriteString("Options:\n")
	help.WriteString("  --input <path>    Input file path, URL, or - for stdin\n")
	help.WriteString("  --help            Show this help message\n\n")
	help.WriteString("Examples:\n")
	help.WriteString("  yaswag validate --input ./swagger.yaml\n")
	help.WriteString("  yaswag validate --input https://petstore3.swagger.io/api/v3/openapi.json\n")
	help.WriteString("  yaswag generate --source ./api | yaswag validate\n")
	help.WriteString("  cat swagger.yaml | yaswag validate\n")
	return help.String()
}

func (c *CLI) FormatHelp() string {
	help := strings.Builder{}
	help.WriteString("Format an OpenAPI specification file.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag format [options]\n")
	help.WriteString("  <command> | yaswag format [options]\n\n")
	help.WriteString("Options:\n")
	help.WriteString("  --input <path>    Input file path or - for stdin\n")
	help.WriteString("  --output <path>   Output file path (empty for stdout)\n")
	help.WriteString("  --format <type>   Output format: json or yaml (auto-detected if not specified)\n")
	help.WriteString("  --pretty <n>      Indentation spaces (default: 4)\n")
	help.WriteString("  --help            Show this help message\n\n")
	help.WriteString("Examples:\n")
	help.WriteString("  yaswag format --input ./swagger.json --pretty 2\n")
	help.WriteString("  yaswag format --input ./swagger.yaml --output ./swagger-formatted.yaml\n")
	help.WriteString("  yaswag generate --source ./api | yaswag format --format json\n")
	help.WriteString("  cat swagger.yaml | yaswag format --pretty 2\n")
	return help.String()
}

func (c *CLI) ServeHelp() string {
	help := strings.Builder{}
	help.WriteString("Serve OpenAPI specification with Swagger UI.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag serve [options]\n")
	help.WriteString("  <command> | yaswag serve\n\n")
	help.WriteString("Options:\n")
	help.WriteString("  --input <path>    Input file path, URL, or - for stdin\n")
	help.WriteString("  --port <n>        Port to serve on (default: 8080)\n")
	help.WriteString("  --help            Show this help message\n\n")
	help.WriteString("Examples:\n")
	help.WriteString("  yaswag serve --input ./swagger.yaml\n")
	help.WriteString("  yaswag serve --input ./swagger.yaml --port 9090\n")
	help.WriteString("  yaswag serve --input https://example.com/api/swagger.yaml\n")
	help.WriteString("  yaswag generate --source ./api | yaswag serve\n")
	help.WriteString("  yaswag generate --source ./api | yaswag serve --port 9090\n")
	help.WriteString("  cat swagger.yaml | yaswag serve\n")
	return help.String()
}

func (c *CLI) EditorHelp() string {
	help := strings.Builder{}
	help.WriteString("Launch Swagger Editor for creating and editing OpenAPI specifications.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag editor [options]\n")
	help.WriteString("  <command> | yaswag editor\n\n")
	help.WriteString("Options:\n")
	help.WriteString("  --input <path>    Input file path, URL, or - for stdin (optional)\n")
	help.WriteString("  --port <n>        Port to serve on (default: 8080)\n")
	help.WriteString("  --help            Show this help message\n\n")
	help.WriteString("Examples:\n")
	help.WriteString("  yaswag editor\n")
	help.WriteString("  yaswag editor --port 9090\n")
	help.WriteString("  yaswag editor --input ./swagger.yaml\n")
	help.WriteString("  yaswag editor --input https://petstore3.swagger.io/api/v3/openapi.json\n")
	help.WriteString("  yaswag generate --source ./api | yaswag editor\n")
	help.WriteString("  cat swagger.yaml | yaswag editor\n")
	return help.String()
}

func (c *CLI) MCPHelp() string {
	help := strings.Builder{}
	help.WriteString("Start MCP (Model Context Protocol) server for AI assistant integration.\n\n")
	help.WriteString("The MCP server enables AI assistants like Claude to interact with OpenAPI\n")
	help.WriteString("specifications through semantic search, schema exploration, and validation.\n\n")
	help.WriteString("Usage:\n")
	help.WriteString("  yaswag mcp [options] <spec-file> [spec-file...]\n\n")
	help.WriteString("Options:\n")
	help.WriteString("  --skip-validation Skip spec validation before starting the server\n")
	help.WriteString("  --help            Show this help message\n\n")
	help.WriteString("Note: MCP mode requires spec files as arguments. Stdin piping is not\n")
	help.WriteString("supported because stdin is reserved for JSON-RPC communication.\n\n")
	help.WriteString("Available Tools:\n")
	help.WriteString("  search_endpoints  Search for endpoints using natural language\n")
	help.WriteString("  list_endpoints    List all endpoints with optional filtering\n")
	help.WriteString("  get_endpoint      Get detailed endpoint information\n")
	help.WriteString("  search_schemas    Search for schema definitions\n")
	help.WriteString("  get_schema        Get detailed schema definition\n")
	help.WriteString("  validate_spec     Validate the OpenAPI specification\n")
	help.WriteString("  get_spec_info     Get general specification information\n")
	help.WriteString("  generate_example  Generate example request/response data\n")
	help.WriteString("  find_related      Find related endpoints\n")
	help.WriteString("  list_tags         List all tags with endpoint counts\n")
	help.WriteString("  analyze_security  Analyze security requirements\n\n")
	help.WriteString("Claude Code Configuration:\n")
	help.WriteString("  Add to your .claude/settings.json:\n\n")
	help.WriteString("  {\n")
	help.WriteString("    \"mcpServers\": {\n")
	help.WriteString("      \"yaswag\": {\n")
	help.WriteString("        \"command\": \"yaswag\",\n")
	help.WriteString("        \"args\": [\"mcp\", \"./openapi.json\"]\n")
	help.WriteString("      }\n")
	help.WriteString("    }\n")
	help.WriteString("  }\n\n")
	help.WriteString("Examples:\n")
	help.WriteString("  yaswag mcp ./openapi.json\n")
	help.WriteString("  yaswag mcp ./api/swagger.yaml ./api/openapi.json\n")
	help.WriteString("  yaswag mcp --skip-validation ./openapi.json\n")
	return help.String()
}

// formatSpec formats an OpenAPI spec to the specified format with indentation.
func formatSpec(data []byte, format output.Format, indent int) ([]byte, error) {
	// Use libopenapi to parse and render
	// For now, we use a simpler approach: parse as generic YAML/JSON and re-encode

	// Parse as generic interface
	var spec any

	// Try YAML first (it also parses JSON)
	if err := yamlUnmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	// Re-encode in the target format
	switch format {
	case output.FormatJSON:
		return jsonMarshalIndent(spec, indent)
	case output.FormatYAML:
		return yamlMarshalIndent(spec, indent)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
