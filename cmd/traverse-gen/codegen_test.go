package main

import (
	"go/format"
	"os"
	"strings"
	"testing"
)

// parseNorthwind is a helper that parses the testdata/northwind.edmx file.
func parseNorthwind(t *testing.T) []Schema {
	t.Helper()
	f, err := os.Open("testdata/northwind.edmx")
	if err != nil {
		t.Fatalf("open northwind.edmx: %v", err)
	}
	defer func() { _ = f.Close() }()
	schemas, err := parseEDMX(f, "")
	if err != nil {
		t.Fatalf("parseEDMX: %v", err)
	}
	return schemas
}

// TestParseEDMX_NorthwindEntityTypes verifies entity types are parsed from northwind.edmx.
func TestParseEDMX_NorthwindEntityTypes(t *testing.T) {
	schemas := parseNorthwind(t)

	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	s := schemas[0]

	if s.Namespace != "NorthWind" {
		t.Errorf("namespace = %q, want %q", s.Namespace, "NorthWind")
	}
	if len(s.EntityTypes) != 3 {
		t.Fatalf("expected 3 entity types, got %d", len(s.EntityTypes))
	}

	wantTypes := []string{"Customer", "Order", "OrderDetail"}
	for i, want := range wantTypes {
		if s.EntityTypes[i].Name != want {
			t.Errorf("EntityType[%d].Name = %q, want %q", i, s.EntityTypes[i].Name, want)
		}
	}
}

// TestParseEDMX_NorthwindKeys verifies composite and single keys are parsed correctly.
func TestParseEDMX_NorthwindKeys(t *testing.T) {
	schemas := parseNorthwind(t)
	s := schemas[0]

	customerKeys := s.EntityTypes[0].Keys
	if len(customerKeys) != 1 || customerKeys[0] != "CustomerID" {
		t.Errorf("Customer keys = %v, want [CustomerID]", customerKeys)
	}

	orderDetailKeys := s.EntityTypes[2].Keys
	if len(orderDetailKeys) != 2 {
		t.Fatalf("OrderDetail should have 2 keys, got %d", len(orderDetailKeys))
	}
	if orderDetailKeys[0] != "OrderID" || orderDetailKeys[1] != "ProductID" {
		t.Errorf("OrderDetail keys = %v, want [OrderID ProductID]", orderDetailKeys)
	}
}

// TestParseEDMX_NorthwindProperties verifies properties and nullable flags.
func TestParseEDMX_NorthwindProperties(t *testing.T) {
	schemas := parseNorthwind(t)
	customer := schemas[0].EntityTypes[0] // Customer

	propMap := make(map[string]SchemaProperty)
	for _, p := range customer.Properties {
		propMap[p.Name] = p
	}

	cases := []struct {
		name     string
		wantType string
		nullable bool
	}{
		{"CustomerID", "Edm.String", false},
		{"CompanyName", "Edm.String", false},
		{"ContactName", "Edm.String", true},
	}
	for _, tc := range cases {
		p, ok := propMap[tc.name]
		if !ok {
			t.Errorf("property %q not found", tc.name)
			continue
		}
		if p.Type != tc.wantType {
			t.Errorf("Customer.%s.Type = %q, want %q", tc.name, p.Type, tc.wantType)
		}
		if p.Nullable != tc.nullable {
			t.Errorf("Customer.%s.Nullable = %v, want %v", tc.name, p.Nullable, tc.nullable)
		}
	}
}

// TestParseEDMX_NorthwindNavProps verifies v4 navigation properties (Type/Partner attrs).
func TestParseEDMX_NorthwindNavProps(t *testing.T) {
	schemas := parseNorthwind(t)

	customer := schemas[0].EntityTypes[0]
	if len(customer.NavigationProperties) != 1 {
		t.Fatalf("Customer: expected 1 nav prop, got %d", len(customer.NavigationProperties))
	}
	nav := customer.NavigationProperties[0]
	if nav.Name != "Orders" {
		t.Errorf("nav.Name = %q, want Orders", nav.Name)
	}
	if !nav.IsCollection {
		t.Error("Customer.Orders should be a collection")
	}
	if nav.TargetType != "Order" {
		t.Errorf("nav.TargetType = %q, want Order", nav.TargetType)
	}
	if nav.Partner != "Customer" {
		t.Errorf("nav.Partner = %q, want Customer", nav.Partner)
	}

	// Order.Customer - single nav prop
	order := schemas[0].EntityTypes[1]
	navMap := make(map[string]SchemaNavProp)
	for _, n := range order.NavigationProperties {
		navMap[n.Name] = n
	}
	cust := navMap["Customer"]
	if cust.IsCollection {
		t.Error("Order.Customer should NOT be a collection")
	}
	if cust.TargetType != "Customer" {
		t.Errorf("Order.Customer.TargetType = %q, want Customer", cust.TargetType)
	}
}

// TestParseEDMX_NorthwindEntitySets verifies entity sets and nav bindings.
func TestParseEDMX_NorthwindEntitySets(t *testing.T) {
	schemas := parseNorthwind(t)
	s := schemas[0]

	if len(s.EntitySets) != 3 {
		t.Fatalf("expected 3 entity sets, got %d", len(s.EntitySets))
	}

	setMap := make(map[string]SchemaEntitySet)
	for _, es := range s.EntitySets {
		setMap[es.Name] = es
	}

	customers, ok := setMap["Customers"]
	if !ok {
		t.Fatal("EntitySet Customers not found")
	}
	if customers.EntityType != "Customer" {
		t.Errorf("Customers.EntityType = %q, want Customer", customers.EntityType)
	}
	if len(customers.Bindings) != 1 || customers.Bindings[0].Path != "Orders" {
		t.Errorf("Customers bindings = %v, want [{Orders Orders}]", customers.Bindings)
	}

	orders, ok := setMap["Orders"]
	if !ok {
		t.Fatal("EntitySet Orders not found")
	}
	if len(orders.Bindings) != 2 {
		t.Errorf("Orders should have 2 nav bindings, got %d", len(orders.Bindings))
	}
}

// TestParseEDMX_NamespaceFilter verifies that the namespace filter works.
func TestParseEDMX_NamespaceFilter(t *testing.T) {
	f, err := os.Open("testdata/northwind.edmx")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	schemas, err := parseEDMX(f, "NoSuchNamespace")
	if err != nil {
		t.Fatalf("parseEDMX: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas for unknown namespace, got %d", len(schemas))
	}
}

// TestMapODataType covers the type mapping function.
func TestMapODataType(t *testing.T) {
	cases := []struct {
		odataType string
		nullable  bool
		want      string
	}{
		{"Edm.String", false, "string"},
		{"Edm.String", true, "*string"},
		{"Edm.Int32", false, "int32"},
		{"Edm.Int32", true, "*int32"},
		{"Edm.Int16", false, "int16"},
		{"Edm.Int64", false, "int64"},
		{"Edm.Boolean", false, "bool"},
		{"Edm.Double", false, "float64"},
		{"Edm.Single", false, "float32"},
		{"Edm.Decimal", false, "traverse.Decimal"},
		{"Edm.Decimal", true, "*traverse.Decimal"},
		{"Edm.DateTimeOffset", false, "traverse.DateTimeOffset"},
		{"Edm.DateTimeOffset", true, "*traverse.DateTimeOffset"},
		{"Edm.DateTime", false, "traverse.DateTime"},
		{"Edm.Guid", false, "traverse.Guid"},
		{"Edm.Binary", false, "[]byte"},
		{"Edm.Binary", true, "[]byte"}, // []byte is already a ref type
		{"Collection(Edm.String)", false, "[]string"},
		{"Collection(NorthWind.Order)", false, "[]Order"},
	}

	for _, tc := range cases {
		got := mapODataType(tc.odataType, tc.nullable)
		if got != tc.want {
			t.Errorf("mapODataType(%q, %v) = %q, want %q", tc.odataType, tc.nullable, got, tc.want)
		}
	}
}

// TestBuildStructTag verifies generated struct tags.
func TestBuildStructTag(t *testing.T) {
	cases := []struct {
		propName string
		isKey    bool
		isNav    bool
		nullable bool
		want     string
	}{
		{"ID", true, false, false, "`json:\"ID\" odata:\"key\"`"},
		{"Name", false, false, false, "`json:\"Name\"`"},
		{"ContactName", false, false, true, "`json:\"ContactName,omitempty\"`"},
		{"Orders", false, true, true, "`json:\"Orders,omitempty\" odata:\"nav\"`"},
		{"Customer", false, true, true, "`json:\"Customer,omitempty\" odata:\"nav\"`"},
	}

	for _, tc := range cases {
		got := buildStructTag(tc.propName, tc.isKey, tc.isNav, tc.nullable)
		if got != tc.want {
			t.Errorf("buildStructTag(%q, key=%v, nav=%v, nil=%v) = %q, want %q",
				tc.propName, tc.isKey, tc.isNav, tc.nullable, got, tc.want)
		}
	}
}

// TestRenderTypes_Northwind verifies that types.go generation produces valid Go code
// with the expected entity structs.
func TestRenderTypes_Northwind(t *testing.T) {
	schemas := parseNorthwind(t)
	gen := NewCodeGenerator(schemas, "northwind")

	code, err := gen.RenderTypes()
	if err != nil {
		t.Fatalf("RenderTypes: %v", err)
	}

	// Must be valid Go
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated types.go is not valid Go: %v\n---\n%s", err, code)
	}

	// Should contain expected struct declarations
	wantContains := []string{
		"package northwind",
		"type Customer struct",
		"type Order struct",
		"type OrderDetail struct",
		`json:"CustomerID" odata:"key"`,
		`json:"CompanyName"`,
		`json:"ContactName,omitempty"`,
		`odata:"nav"`,
		"traverse.DateTimeOffset",
		"traverse.Decimal",
		"int32",
		"int16",
	}
	for _, want := range wantContains {
		if !strings.Contains(code, want) {
			t.Errorf("types.go missing expected content %q\n---\n%s", want, code)
		}
	}
}

// TestRenderClient_Northwind verifies client.go generation.
func TestRenderClient_Northwind(t *testing.T) {
	schemas := parseNorthwind(t)
	gen := NewCodeGenerator(schemas, "northwind")

	code, err := gen.RenderClient()
	if err != nil {
		t.Fatalf("RenderClient: %v", err)
	}

	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated client.go is not valid Go: %v\n---\n%s", err, code)
	}

	wantContains := []string{
		"package northwind",
		"type GeneratedClient struct",
		"*traverse.Client",
		"func NewGeneratedClient",
		"traverse.WithBaseURL(baseURL)",
		"traverse.New(",
		"func (c *GeneratedClient) Customers()",
		"func (c *GeneratedClient) Orders()",
		"func (c *GeneratedClient) OrderDetails()",
		`c.Client.From("Customers")`,
		"*CustomersQuery",
		"*OrdersQuery",
	}
	for _, want := range wantContains {
		if !strings.Contains(code, want) {
			t.Errorf("client.go missing expected content %q\n---\n%s", want, code)
		}
	}
}

// TestRenderQueries_Northwind verifies queries.go generation.
func TestRenderQueries_Northwind(t *testing.T) {
	schemas := parseNorthwind(t)
	gen := NewCodeGenerator(schemas, "northwind")

	code, err := gen.RenderQueries()
	if err != nil {
		t.Fatalf("RenderQueries: %v", err)
	}

	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated queries.go is not valid Go: %v\n---\n%s", err, code)
	}

	wantContains := []string{
		"package northwind",
		"type CustomersQuery struct",
		"type OrdersQuery struct",
		"type OrderDetailsQuery struct",
		"*traverse.QueryBuilder",
	}
	for _, want := range wantContains {
		if !strings.Contains(code, want) {
			t.Errorf("queries.go missing expected content %q\n---\n%s", want, code)
		}
	}
}

// TestPascalCase verifies PascalCase conversion.
func TestPascalCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"customer", "Customer"},
		{"order_detail", "OrderDetail"},
		{"OrderDetails", "OrderDetails"},
		{"", ""},
		{"a", "A"},
	}
	for _, tc := range cases {
		if got := PascalCase(tc.in); got != tc.want {
			t.Errorf("PascalCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNavGoType verifies nav property Go type generation.
func TestNavGoType(t *testing.T) {
	collection := SchemaNavProp{IsCollection: true, TargetType: "Order"}
	single := SchemaNavProp{IsCollection: false, TargetType: "Customer"}

	if got := navGoType(collection); got != "[]Order" {
		t.Errorf("navGoType(collection) = %q, want []Order", got)
	}
	if got := navGoType(single); got != "*Customer" {
		t.Errorf("navGoType(single) = %q, want *Customer", got)
	}
}

// TestNeedsImports verifies import detection.
func TestNeedsImports(t *testing.T) {
	schemas := parseNorthwind(t)

	// NorthWind has Edm.DateTimeOffset and Edm.Decimal -> needs traverse import
	if !needsTraverseImport(schemas) {
		t.Error("needsTraverseImport should be true for northwind schema")
	}

	// NorthWind has no Edm.Date / Edm.Time -> no time import needed
	if needsTimeImport(schemas) {
		t.Error("needsTimeImport should be false for northwind schema")
	}
}

// TestRenderTypes_EmptySchema verifies graceful handling of an empty schema.
func TestRenderTypes_EmptySchema(t *testing.T) {
	schemas := []Schema{{Namespace: "Empty"}}
	gen := NewCodeGenerator(schemas, "empty")

	code, err := gen.RenderTypes()
	if err != nil {
		t.Fatalf("RenderTypes on empty schema: %v", err)
	}
	if _, err := format.Source([]byte(code)); err != nil {
		t.Fatalf("generated types.go for empty schema is not valid Go: %v\n---\n%s", err, code)
	}
	if !strings.Contains(code, "package empty") {
		t.Errorf("missing package declaration in: %s", code)
	}
}
