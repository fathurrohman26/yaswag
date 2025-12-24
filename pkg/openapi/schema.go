package openapi

import "encoding/json"

// Schema represents a JSON Schema object that describes the structure of data.
// https://spec.openapis.org/oas/v3.1.0#schema-object
type Schema struct {
	Ref          string                 `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type         SchemaType             `json:"type,omitempty" yaml:"type,omitempty"`
	Format       string                 `json:"format,omitempty" yaml:"format,omitempty"`
	Title        string                 `json:"title,omitempty" yaml:"title,omitempty"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Default      any                    `json:"default,omitempty" yaml:"default,omitempty"`
	Nullable     bool                   `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Deprecated   bool                   `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	ReadOnly     bool                   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly    bool                   `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	Example      any                    `json:"example,omitempty" yaml:"example,omitempty"`
	Examples     []any                  `json:"examples,omitempty" yaml:"examples,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`

	// String validation
	MinLength *int64 `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength *int64 `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	// Number validation
	Minimum          *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`

	// Array validation
	Items       *Schema `json:"items,omitempty" yaml:"items,omitempty"`
	MinItems    *int64  `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems    *int64  `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	UniqueItems bool    `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

	// Object validation
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Required             []string           `json:"required,omitempty" yaml:"required,omitempty"`
	MinProperties        *int64             `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProperties        *int64             `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`

	// Composition
	AllOf []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOf []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Not   *Schema   `json:"not,omitempty" yaml:"not,omitempty"`

	// Enumeration
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Discriminator
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`

	// XML
	XML *XML `json:"xml,omitempty" yaml:"xml,omitempty"`
}

// SchemaType represents the type field which can be a single type or array of types.
type SchemaType []string

// Type constants for schema types.
const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeInteger = "integer"
	TypeBoolean = "boolean"
	TypeArray   = "array"
	TypeObject  = "object"
	TypeNull    = "null"
)

// NewSchemaType creates a new SchemaType from a single type string.
func NewSchemaType(t string) SchemaType {
	return SchemaType{t}
}

// MarshalJSON implements json.Marshaler.
// For OpenAPI 3.0 compatibility, a single type is marshaled as a string.
func (s SchemaType) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("null"), nil
	}
	if len(s) == 1 {
		return []byte(`"` + s[0] + `"`), nil
	}
	// Multiple types (OpenAPI 3.1+)
	result := `[`
	for i, t := range s {
		if i > 0 {
			result += `,`
		}
		result += `"` + t + `"`
	}
	result += `]`
	return []byte(result), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// Handles both string (OpenAPI 3.0) and array (OpenAPI 3.1+) formats.
func (s *SchemaType) UnmarshalJSON(data []byte) error {
	// Try string first (OpenAPI 3.0 format)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = SchemaType{str}
		return nil
	}

	// Try array (OpenAPI 3.1+ format)
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*s = arr
	return nil
}

// MarshalYAML implements yaml.Marshaler.
// For OpenAPI 3.0 compatibility, a single type is marshaled as a string.
func (s SchemaType) MarshalYAML() (interface{}, error) {
	if len(s) == 0 {
		return nil, nil
	}
	if len(s) == 1 {
		return s[0], nil
	}
	// Multiple types (OpenAPI 3.1+)
	return []string(s), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
// Handles both string (OpenAPI 3.0) and array (OpenAPI 3.1+) formats.
func (s *SchemaType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string first (OpenAPI 3.0 format)
	var str string
	if err := unmarshal(&str); err == nil {
		*s = SchemaType{str}
		return nil
	}

	// Try array (OpenAPI 3.1+ format)
	var arr []string
	if err := unmarshal(&arr); err != nil {
		return err
	}
	*s = arr
	return nil
}

// Discriminator is used when request bodies or response payloads may be one of a number of different schemas.
// https://spec.openapis.org/oas/v3.1.0#discriminator-object
type Discriminator struct {
	PropertyName string            `json:"propertyName" yaml:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

// XML provides additional information to describe XML representation of this property.
// https://spec.openapis.org/oas/v3.1.0#xml-object
type XML struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty" yaml:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
}

// RefTo creates a reference schema to a component schema.
func RefTo(name string) *Schema {
	return &Schema{
		Ref: "#/components/schemas/" + name,
	}
}

// RefToResponse creates a reference to a component response.
func RefToResponse(name string) *Response {
	return &Response{
		Ref: "#/components/responses/" + name,
	}
}

// RefToParameter creates a reference to a component parameter.
func RefToParameter(name string) *Parameter {
	return &Parameter{
		Ref: "#/components/parameters/" + name,
	}
}

// RefToRequestBody creates a reference to a component request body.
func RefToRequestBody(name string) *RequestBody {
	return &RequestBody{
		Ref: "#/components/requestBodies/" + name,
	}
}

// StringSchema creates a string schema.
func StringSchema() *Schema {
	return &Schema{Type: NewSchemaType(TypeString)}
}

// IntegerSchema creates an integer schema.
func IntegerSchema() *Schema {
	return &Schema{Type: NewSchemaType(TypeInteger)}
}

// NumberSchema creates a number schema.
func NumberSchema() *Schema {
	return &Schema{Type: NewSchemaType(TypeNumber)}
}

// BooleanSchema creates a boolean schema.
func BooleanSchema() *Schema {
	return &Schema{Type: NewSchemaType(TypeBoolean)}
}

// ArraySchema creates an array schema with the given items schema.
func ArraySchema(items *Schema) *Schema {
	return &Schema{
		Type:  NewSchemaType(TypeArray),
		Items: items,
	}
}

// ObjectSchema creates an object schema.
func ObjectSchema() *Schema {
	return &Schema{
		Type:       NewSchemaType(TypeObject),
		Properties: make(map[string]*Schema),
	}
}
