package mcp

import (
	"encoding/json"
	"testing"
)

func TestSchemaBuilder_Empty(t *testing.T) {
	schema := Schema().Build()
	if schema.Type != "object" {
		t.Errorf("Type = %q, want %q", schema.Type, "object")
	}
	if len(schema.Properties) != 0 {
		t.Errorf("expected no properties, got %d", len(schema.Properties))
	}
	if len(schema.Required) != 0 {
		t.Errorf("expected no required fields, got %v", schema.Required)
	}
}

func TestSchemaBuilder_AllTypes(t *testing.T) {
	schema := Schema().
		String("name", "A name").
		Number("score", "A score").
		Integer("count", "A count").
		Bool("active", "Active flag").
		Build()

	wantTypes := map[string]string{
		"name":   "string",
		"score":  "number",
		"count":  "integer",
		"active": "boolean",
	}
	for prop, wantType := range wantTypes {
		t.Run(prop, func(t *testing.T) {
			p, ok := schema.Properties[prop]
			if !ok {
				t.Fatalf("property %q not found", prop)
			}
			if p.Type != wantType {
				t.Errorf("Type = %q, want %q", p.Type, wantType)
			}
		})
	}
}

func TestSchemaBuilder_Required(t *testing.T) {
	schema := Schema().
		String("required_field", "Must be present", Required()).
		String("optional_field", "May be omitted").
		Integer("also_required", "Must be present", Required()).
		Build()

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	if !requiredSet["required_field"] {
		t.Error("required_field must be in Required slice")
	}
	if requiredSet["optional_field"] {
		t.Error("optional_field must NOT be in Required slice")
	}
	if !requiredSet["also_required"] {
		t.Error("also_required must be in Required slice")
	}
}

func TestSchemaBuilder_WithEnum(t *testing.T) {
	schema := Schema().
		String("color", "A color", WithEnum("red", "green", "blue")).
		Build()

	p, ok := schema.Properties["color"]
	if !ok {
		t.Fatal("property 'color' not found")
	}
	want := []string{"red", "green", "blue"}
	if len(p.Enum) != len(want) {
		t.Fatalf("Enum length = %d, want %d", len(p.Enum), len(want))
	}
	for i, v := range want {
		if p.Enum[i] != v {
			t.Errorf("Enum[%d] = %q, want %q", i, p.Enum[i], v)
		}
	}
}

func TestSchemaBuilder_WithDefault(t *testing.T) {
	tests := []struct {
		name   string
		def    any
		propFn func(b *SchemaBuilder) *SchemaBuilder
	}{
		{
			name: "integer default",
			def:  10,
			propFn: func(b *SchemaBuilder) *SchemaBuilder {
				return b.Integer("limit", "Max", WithDefault(10))
			},
		},
		{
			name: "string default",
			def:  "asc",
			propFn: func(b *SchemaBuilder) *SchemaBuilder {
				return b.String("order", "Sort order", WithDefault("asc"))
			},
		},
		{
			name: "bool default",
			def:  true,
			propFn: func(b *SchemaBuilder) *SchemaBuilder {
				return b.Bool("verbose", "Verbose output", WithDefault(true))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Schema()
			tt.propFn(b)
			schema := b.Build()

			// Find the property (first one)
			if len(schema.Properties) == 0 {
				t.Fatal("no properties in schema")
			}
			for _, p := range schema.Properties {
				if p.Default != tt.def {
					t.Errorf("Default = %v, want %v", p.Default, tt.def)
				}
			}
		})
	}
}

func TestSchemaBuilder_RequiredAndEnum_Combined(t *testing.T) {
	schema := Schema().
		String("status", "Status filter",
			Required(),
			WithEnum("active", "inactive"),
			WithDefault("active"),
		).
		Build()

	if len(schema.Required) != 1 || schema.Required[0] != "status" {
		t.Errorf("Required = %v, want [status]", schema.Required)
	}
	p, ok := schema.Properties["status"]
	if !ok {
		t.Fatal("property 'status' not found")
	}
	if len(p.Enum) != 2 {
		t.Errorf("Enum len = %d, want 2", len(p.Enum))
	}
	if p.Default != "active" {
		t.Errorf("Default = %v, want %q", p.Default, "active")
	}
}

func TestInputSchema_JSON(t *testing.T) {
	schema := Schema().
		String("query", "Search query", Required()).
		Integer("limit", "Max results", WithDefault(10)).
		Build()

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if m["type"] != "object" {
		t.Errorf(`"type" = %v, want "object"`, m["type"])
	}

	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties must be an object")
	}
	if _, ok := props["query"]; !ok {
		t.Error("properties must contain 'query'")
	}
	if _, ok := props["limit"]; !ok {
		t.Error("properties must contain 'limit'")
	}

	req, ok := m["required"].([]any)
	if !ok {
		t.Fatal("required must be an array")
	}
	if len(req) != 1 || req[0] != "query" {
		t.Errorf("required = %v, want [query]", req)
	}
}

func TestSchemaBuilder_Chaining(t *testing.T) {
	// Verify method chaining returns the same builder.
	b := Schema()
	b2 := b.String("a", "field a")
	if b != b2 {
		t.Error("String() must return the same *SchemaBuilder for chaining")
	}
	b3 := b.Number("b", "field b")
	if b != b3 {
		t.Error("Number() must return the same *SchemaBuilder for chaining")
	}
}

func TestSchemaProperty_JSON_OmitsEmpty(t *testing.T) {
	schema := Schema().String("q", "query").Build()
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	props := m["properties"].(map[string]any)
	qProp := props["q"].(map[string]any)

	// enum and default should be omitted when not set
	if _, ok := qProp["enum"]; ok {
		t.Error(`"enum" must be omitted when not set`)
	}
	if _, ok := qProp["default"]; ok {
		t.Error(`"default" must be omitted when not set`)
	}
}
