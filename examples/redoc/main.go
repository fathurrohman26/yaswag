package main

import (
	"log"
	"net/http"

	"github.com/fathurrohman26/yaswag/pkg/openapi"
	"github.com/fathurrohman26/yaswag/pkg/yahttp"
)

func main() {
	// Create OpenAPI spec
	spec := createSpec()

	// Create HTTP plugin
	plugin := yahttp.New(spec, &yahttp.Options{
		SpecPath: "/openapi.json",
	})

	// Create router
	mux := http.NewServeMux()

	// Mount API routes
	mux.HandleFunc("/pets", petsHandler)
	mux.HandleFunc("/pets/", petByIDHandler)

	// Mount documentation endpoints
	mux.Handle("/openapi.json", plugin.SpecHandler())
	mux.Handle("/openapi.yaml", plugin.SpecHandler())
	mux.Handle("/docs", plugin.SwaggerUIHandler())
	mux.Handle("/redoc", plugin.RedocHandler())

	// Or use custom ReDoc options
	mux.Handle("/redoc-custom", plugin.RedocHandlerWithOptions(&yahttp.RedocOptions{
		Title:   "Pet Store API - ReDoc",
		SpecURL: "/openapi.json",
	}))

	log.Println("Server starting on http://localhost:8080")
	log.Println("ReDoc:      http://localhost:8080/redoc")
	log.Println("Swagger UI: http://localhost:8080/docs")
	log.Println("OpenAPI:    http://localhost:8080/openapi.json")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func createSpec() *openapi.Document {
	return &openapi.Document{
		OpenAPI: "3.0.3",
		Info: openapi.Info{
			Title:       "Pet Store API",
			Version:     "1.0.0",
			Description: "A sample Pet Store API to demonstrate ReDoc documentation",
			Contact: &openapi.Contact{
				Name:  "API Support",
				Email: "support@example.com",
				URL:   "https://example.com/support",
			},
			License: &openapi.License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []openapi.Server{
			{URL: "http://localhost:8080", Description: "Local development server"},
		},
		Tags: []openapi.Tag{
			{Name: "pets", Description: "Pet operations"},
		},
		Paths: map[string]*openapi.PathItem{
			"/pets": {
				Get: &openapi.Operation{
					Summary:     "List all pets",
					Description: "Returns a list of all pets in the store",
					OperationID: "listPets",
					Tags:        []string{"pets"},
					Parameters: []*openapi.Parameter{
						{
							Name:        "limit",
							In:          openapi.ParameterInQuery,
							Description: "Maximum number of pets to return",
							Required:    false,
							Schema:      openapi.IntegerSchema(),
						},
						{
							Name:        "offset",
							In:          openapi.ParameterInQuery,
							Description: "Number of pets to skip",
							Required:    false,
							Schema:      openapi.IntegerSchema(),
						},
					},
					Responses: map[string]*openapi.Response{
						"200": {
							Description: "A list of pets",
							Content: map[string]openapi.MediaType{
								"application/json": {
									Schema: openapi.ArraySchema(&openapi.Schema{
										Ref: "#/components/schemas/Pet",
									}),
								},
							},
						},
					},
				},
				Post: &openapi.Operation{
					Summary:     "Create a pet",
					Description: "Creates a new pet in the store",
					OperationID: "createPet",
					Tags:        []string{"pets"},
					RequestBody: &openapi.RequestBody{
						Description: "Pet to create",
						Required:    true,
						Content: map[string]openapi.MediaType{
							"application/json": {
								Schema: &openapi.Schema{
									Ref: "#/components/schemas/CreatePet",
								},
							},
						},
					},
					Responses: map[string]*openapi.Response{
						"201": {
							Description: "Pet created successfully",
							Content: map[string]openapi.MediaType{
								"application/json": {
									Schema: &openapi.Schema{
										Ref: "#/components/schemas/Pet",
									},
								},
							},
						},
						"400": {
							Description: "Invalid input",
						},
					},
				},
			},
			"/pets/{id}": {
				Get: &openapi.Operation{
					Summary:     "Get a pet by ID",
					Description: "Returns a single pet by its ID",
					OperationID: "getPet",
					Tags:        []string{"pets"},
					Parameters: []*openapi.Parameter{
						{
							Name:        "id",
							In:          openapi.ParameterInPath,
							Description: "Pet ID",
							Required:    true,
							Schema:      openapi.IntegerSchema(),
						},
					},
					Responses: map[string]*openapi.Response{
						"200": {
							Description: "Pet found",
							Content: map[string]openapi.MediaType{
								"application/json": {
									Schema: &openapi.Schema{
										Ref: "#/components/schemas/Pet",
									},
								},
							},
						},
						"404": {
							Description: "Pet not found",
						},
					},
				},
				Delete: &openapi.Operation{
					Summary:     "Delete a pet",
					Description: "Deletes a pet by its ID",
					OperationID: "deletePet",
					Tags:        []string{"pets"},
					Parameters: []*openapi.Parameter{
						{
							Name:        "id",
							In:          openapi.ParameterInPath,
							Description: "Pet ID",
							Required:    true,
							Schema:      openapi.IntegerSchema(),
						},
					},
					Responses: map[string]*openapi.Response{
						"204": {
							Description: "Pet deleted successfully",
						},
						"404": {
							Description: "Pet not found",
						},
					},
				},
			},
		},
		Components: &openapi.Components{
			Schemas: map[string]*openapi.Schema{
				"Pet": {
					Type:        openapi.NewSchemaType(openapi.TypeObject),
					Description: "A pet in the store",
					Required:    []string{"id", "name"},
					Properties: map[string]*openapi.Schema{
						"id": {
							Type:        openapi.NewSchemaType(openapi.TypeInteger),
							Description: "Unique identifier",
							Example:     1,
						},
						"name": {
							Type:        openapi.NewSchemaType(openapi.TypeString),
							Description: "Pet name",
							Example:     "Fluffy",
						},
						"species": {
							Type:        openapi.NewSchemaType(openapi.TypeString),
							Description: "Pet species",
							Enum:        []any{"dog", "cat", "bird", "fish"},
							Example:     "cat",
						},
						"age": {
							Type:        openapi.NewSchemaType(openapi.TypeInteger),
							Description: "Pet age in years",
							Example:     3,
						},
						"createdAt": {
							Type:        openapi.NewSchemaType(openapi.TypeString),
							Format:      "date-time",
							Description: "Creation timestamp",
						},
					},
				},
				"CreatePet": {
					Type:        openapi.NewSchemaType(openapi.TypeObject),
					Description: "Data for creating a new pet",
					Required:    []string{"name"},
					Properties: map[string]*openapi.Schema{
						"name": {
							Type:        openapi.NewSchemaType(openapi.TypeString),
							Description: "Pet name",
							Example:     "Fluffy",
						},
						"species": {
							Type:        openapi.NewSchemaType(openapi.TypeString),
							Description: "Pet species",
							Enum:        []any{"dog", "cat", "bird", "fish"},
						},
						"age": {
							Type:        openapi.NewSchemaType(openapi.TypeInteger),
							Description: "Pet age in years",
						},
					},
				},
			},
		},
	}
}

func petsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		_, _ = w.Write([]byte(`[{"id":1,"name":"Fluffy","species":"cat","age":3},{"id":2,"name":"Buddy","species":"dog","age":5}]`))
	case http.MethodPost:
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":3,"name":"New Pet","species":"bird","age":1}`))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func petByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		_, _ = w.Write([]byte(`{"id":1,"name":"Fluffy","species":"cat","age":3}`))
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
