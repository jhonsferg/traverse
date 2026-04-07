package openapi_test

import (
	"testing"

	"github.com/jhonsferg/traverse"
	"github.com/jhonsferg/traverse/ext/openapi"
)

// customerMeta returns a Metadata with a single Customer entity.
func customerMeta() *traverse.Metadata {
	return &traverse.Metadata{
		Namespace: "Northwind",
		EntityTypes: []traverse.EntityType{
			{
				Name: "Customer",
				Key:  []traverse.PropertyRef{{Name: "ID"}},
				Properties: []traverse.Property{
					{Name: "ID", Type: "Edm.Int32", Nullable: false},
					{Name: "Name", Type: "Edm.String", Nullable: false},
					{Name: "Email", Type: "Edm.String", Nullable: true},
				},
			},
		},
		EntitySets: []traverse.EntitySetInfo{
			{Name: "Customers", EntityTypeName: "Northwind.Customer"},
		},
	}
}

func TestExport_BasicEntitySet(t *testing.T) {
	meta := customerMeta()
	doc, err := openapi.Export(meta)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}
	if doc.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi 3.1.0, got %s", doc.OpenAPI)
	}

	if _, ok := doc.Paths["/Customers"]; !ok {
		t.Error("expected path /Customers to exist")
	}
	if doc.Paths["/Customers"].Get == nil {
		t.Error("expected GET /Customers")
	}
	if doc.Paths["/Customers"].Post == nil {
		t.Error("expected POST /Customers")
	}
}

func TestExport_KeyParameter(t *testing.T) {
	meta := customerMeta()
	doc, err := openapi.Export(meta)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	itemPath := "/Customers({ID})"
	item, ok := doc.Paths[itemPath]
	if !ok {
		t.Fatalf("expected path %s to exist, got paths: %v", itemPath, pathKeys(doc.Paths))
	}
	if item.Get == nil {
		t.Error("expected GET on item path")
	}
	if item.Patch == nil {
		t.Error("expected PATCH on item path")
	}
	if item.Delete == nil {
		t.Error("expected DELETE on item path")
	}

	if len(item.Get.Parameters) == 0 {
		t.Fatal("expected at least one path parameter on GET")
	}
	param := item.Get.Parameters[0]
	if param.Name != "ID" {
		t.Errorf("expected param name ID, got %s", param.Name)
	}
	if param.In != "path" {
		t.Errorf("expected param in=path, got %s", param.In)
	}
	if !param.Required {
		t.Error("expected path param to be required")
	}
}

func TestExport_SchemaMapping(t *testing.T) {
	meta := &traverse.Metadata{
		Namespace: "Test",
		EntityTypes: []traverse.EntityType{
			{
				Name: "Order",
				Key:  []traverse.PropertyRef{{Name: "OrderID"}},
				Properties: []traverse.Property{
					{Name: "OrderID", Type: "Edm.Guid", Nullable: false},
					{Name: "CreatedAt", Type: "Edm.DateTimeOffset", Nullable: true},
					{Name: "Amount", Type: "Edm.Decimal", Nullable: true},
					{Name: "Active", Type: "Edm.Boolean", Nullable: true},
					{Name: "Quantity", Type: "Edm.Int32", Nullable: true},
					{Name: "Data", Type: "Edm.Binary", Nullable: true},
				},
			},
		},
		EntitySets: []traverse.EntitySetInfo{
			{Name: "Orders", EntityTypeName: "Test.Order"},
		},
	}

	doc, err := openapi.Export(meta)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	schema, ok := doc.Components.Schemas["Order"]
	if !ok {
		t.Fatal("expected schema for Order")
	}

	cases := []struct {
		prop       string
		wantType   string
		wantFormat string
	}{
		{"OrderID", "string", "uuid"},
		{"CreatedAt", "string", "date-time"},
		{"Amount", "number", ""},
		{"Active", "boolean", ""},
		{"Quantity", "integer", ""},
		{"Data", "string", "byte"},
	}

	for _, tc := range cases {
		prop, found := schema.Properties[tc.prop]
		if !found {
			t.Errorf("property %s not found in schema", tc.prop)
			continue
		}
		if prop.Type != tc.wantType {
			t.Errorf("property %s: want type %s, got %s", tc.prop, tc.wantType, prop.Type)
		}
		if prop.Format != tc.wantFormat {
			t.Errorf("property %s: want format %q, got %q", tc.prop, tc.wantFormat, prop.Format)
		}
	}
}

func TestExport_Options(t *testing.T) {
	meta := customerMeta()
	doc, err := openapi.Export(meta,
		openapi.WithTitle("My API"),
		openapi.WithVersion("2.0.0"),
		openapi.WithServerURL("https://api.example.com/odata"),
	)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	if doc.Info.Title != "My API" {
		t.Errorf("expected title 'My API', got %q", doc.Info.Title)
	}
	if doc.Info.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", doc.Info.Version)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://api.example.com/odata" {
		t.Errorf("expected one server URL, got %v", doc.Servers)
	}
}

func TestExport_WithTagsFromNamespace(t *testing.T) {
	meta := customerMeta()
	doc, err := openapi.Export(meta, openapi.WithTagsFromNamespace())
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	item, ok := doc.Paths["/Customers"]
	if !ok {
		t.Fatal("expected /Customers path")
	}
	if len(item.Get.Tags) == 0 || item.Get.Tags[0] != "Northwind" {
		t.Errorf("expected tag Northwind, got %v", item.Get.Tags)
	}
}

func TestExport_NilMetadata(t *testing.T) {
	_, err := openapi.Export(nil)
	if err == nil {
		t.Error("expected error for nil metadata")
	}
}

func TestExport_DefaultInfo(t *testing.T) {
	meta := customerMeta()
	doc, err := openapi.Export(meta)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}
	if doc.Info.Title == "" {
		t.Error("expected default title to be set")
	}
	if doc.Info.Version == "" {
		t.Error("expected default version to be set")
	}
}

func TestExport_ComponentSchemas(t *testing.T) {
	meta := customerMeta()
	doc, err := openapi.Export(meta)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	if doc.Components == nil {
		t.Fatal("expected Components to be set")
	}
	schema, ok := doc.Components.Schemas["Customer"]
	if !ok {
		t.Fatal("expected schema for Customer")
	}
	if schema.Type != "object" {
		t.Errorf("expected object type, got %s", schema.Type)
	}
	if _, ok := schema.Properties["ID"]; !ok {
		t.Error("expected ID property in Customer schema")
	}
	if _, ok := schema.Properties["Name"]; !ok {
		t.Error("expected Name property in Customer schema")
	}
}

// pathKeys returns the keys of a paths map for error messages.
func pathKeys(paths map[string]*openapi.PathItem) []string {
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	return keys
}
