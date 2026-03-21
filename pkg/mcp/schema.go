package mcp

// InputSchema describes tool input parameters as a minimal JSON Schema object.
// Only the "object" type is supported — all tool inputs are key/value maps.
type InputSchema struct {
	Type       string                    `json:"type"`                 // always "object"
	Properties map[string]SchemaProperty `json:"properties,omitempty"` // named parameters
	Required   []string                  `json:"required,omitempty"`   // required parameter names
}

// SchemaProperty describes a single parameter within an [InputSchema].
type SchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// propertyBuildState holds per-property build options, including whether the
// property is required. This type is internal to the builder and is never
// marshalled to JSON.
type propertyBuildState struct {
	SchemaProperty
	required bool
}

// PropertyOption configures a property during schema building.
// Options are applied by the [SchemaBuilder] before producing the final [InputSchema].
type PropertyOption func(*propertyBuildState)

// Required marks the property as required. When used with any [SchemaBuilder]
// method, the property name is added to the [InputSchema].Required slice.
func Required() PropertyOption {
	return func(p *propertyBuildState) {
		p.required = true
	}
}

// WithEnum constrains the property to a fixed set of allowed string values.
func WithEnum(values ...string) PropertyOption {
	return func(p *propertyBuildState) {
		p.Enum = append(p.Enum, values...)
	}
}

// WithDefault sets the default value for the property.
func WithDefault(v any) PropertyOption {
	return func(p *propertyBuildState) {
		p.Default = v
	}
}

// SchemaBuilder is a fluent builder for [InputSchema]. Create one with [Schema]
// and chain property-adding methods. Call [SchemaBuilder.Build] to produce the
// final InputSchema.
//
// Example:
//
//	schema := mcp.Schema().
//	    String("query", "The search query", mcp.Required()).
//	    Integer("limit", "Max results", mcp.WithDefault(10)).
//	    Build()
type SchemaBuilder struct {
	entries []schemaEntry
}

// schemaEntry is an internal build record for a single property.
type schemaEntry struct {
	name  string
	state propertyBuildState
}

// Schema returns a new SchemaBuilder with no properties.
func Schema() *SchemaBuilder {
	return &SchemaBuilder{}
}

// addProp creates a property entry with the given type and description, applies
// all PropertyOptions, and appends it to the builder.
func (b *SchemaBuilder) addProp(name, typ, desc string, opts []PropertyOption) *SchemaBuilder {
	state := propertyBuildState{
		SchemaProperty: SchemaProperty{
			Type:        typ,
			Description: desc,
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&state)
		}
	}
	b.entries = append(b.entries, schemaEntry{name: name, state: state})
	return b
}

// String adds a string property to the schema.
// Pass [Required]() in opts to mark it as required.
func (b *SchemaBuilder) String(name, desc string, opts ...PropertyOption) *SchemaBuilder {
	return b.addProp(name, "string", desc, opts)
}

// Number adds a number (float) property to the schema.
// Pass [Required]() in opts to mark it as required.
func (b *SchemaBuilder) Number(name, desc string, opts ...PropertyOption) *SchemaBuilder {
	return b.addProp(name, "number", desc, opts)
}

// Integer adds an integer property to the schema.
// Pass [Required]() in opts to mark it as required.
func (b *SchemaBuilder) Integer(name, desc string, opts ...PropertyOption) *SchemaBuilder {
	return b.addProp(name, "integer", desc, opts)
}

// Bool adds a boolean property to the schema.
// Pass [Required]() in opts to mark it as required.
func (b *SchemaBuilder) Bool(name, desc string, opts ...PropertyOption) *SchemaBuilder {
	return b.addProp(name, "boolean", desc, opts)
}

// Build constructs and returns the final [InputSchema]. Properties appear in
// registration order. The Required slice lists property names passed with the
// [Required] option, in registration order.
func (b *SchemaBuilder) Build() InputSchema {
	if len(b.entries) == 0 {
		return InputSchema{Type: "object"}
	}

	schema := InputSchema{
		Type:       "object",
		Properties: make(map[string]SchemaProperty, len(b.entries)),
	}
	for _, e := range b.entries {
		schema.Properties[e.name] = e.state.SchemaProperty
		if e.state.required {
			schema.Required = append(schema.Required, e.name)
		}
	}
	return schema
}
