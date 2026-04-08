package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	relay "github.com/jhonsferg/relay"

	"github.com/jhonsferg/traverse/testutil"
)

// ---------------------------------------------------------------------------
// capabilities.go - WithCapabilitiesValidation and helpers
// ---------------------------------------------------------------------------

func buildTestRegistry(t *testing.T, filterRestrictions, sortRestrictions string) *CapabilitiesRegistry {
	t.Helper()
	template := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Svc" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="Container">
        <EntitySet Name="Products" EntityType="Svc.Product">
          %s
          %s
        </EntitySet>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`
	reg, err := ParseCapabilities([]byte(fmt.Sprintf(template, filterRestrictions, sortRestrictions)))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	return reg
}

func TestWithCapabilitiesValidation_NilRegistryK4(t *testing.T) {
	opt := WithCapabilitiesValidation(nil)
	cfg := &clientConfig{}
	if err := opt(cfg); err != nil {
		t.Fatalf("nil registry should not error: %v", err)
	}
}

func TestWithCapabilitiesValidation_FilterNotAllowed(t *testing.T) {
	reg := buildTestRegistry(t,
		`<Annotation Term="Capabilities.FilterRestrictions"><Record><PropertyValue Property="Filterable" Bool="false"/></Record></Annotation>`,
		"")

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[]}`})

	c, err := New(WithBaseURL(server.URL()), WithCapabilitiesValidation(reg))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.From("Products").Filter("Name eq 'Test'").Collect(context.Background())
	if err == nil {
		t.Fatal("expected error when filtering is not allowed")
	}
}

func TestWithCapabilitiesValidation_FilterAllowed(t *testing.T) {
	reg := buildTestRegistry(t, "", "")

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[]}`})

	c, err := New(WithBaseURL(server.URL()), WithCapabilitiesValidation(reg))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.From("Products").Filter("Name eq 'Test'").Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithCapabilitiesValidation_SortNotAllowed(t *testing.T) {
	reg := buildTestRegistry(t, "",
		`<Annotation Term="Capabilities.SortRestrictions"><Record><PropertyValue Property="Sortable" Bool="false"/></Record></Annotation>`)

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[]}`})

	c, err := New(WithBaseURL(server.URL()), WithCapabilitiesValidation(reg))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.From("Products").OrderBy("Name").Collect(context.Background())
	if err == nil {
		t.Fatal("expected error when sorting is not allowed")
	}
}

func TestWithCapabilitiesValidation_ExpandRestriction(t *testing.T) {
	edmxXML := []byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Svc" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="Container">
        <EntitySet Name="Orders" EntityType="Svc.Order">
          <Annotation Term="Capabilities.ExpandRestrictions">
            <Record>
              <PropertyValue Property="ExpandableProperties">
                <Collection>
                  <Record><PropertyValue Property="Name" String="Items"/></Record>
                </Collection>
              </PropertyValue>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	reg, err := ParseCapabilities(edmxXML)
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[]}`})

	c, err := New(WithBaseURL(server.URL()), WithCapabilitiesValidation(reg))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	// "Customer" is not in the expandable list (only "Items" is)
	_, err = c.From("Orders").Expand("Customer").Collect(context.Background())
	if err == nil {
		t.Fatal("expected CapabilityError for expand not in allowed list")
	}
	var capErr *CapabilityError
	if !isCapabilityError(err, &capErr) {
		t.Fatalf("expected CapabilityError, got %T: %v", err, err)
	}
}

func isCapabilityError(err error, out **CapabilityError) bool {
	if ce, ok := err.(*CapabilityError); ok {
		if out != nil {
			*out = ce
		}
		return true
	}
	return false
}

func TestCapabilityError_Error(t *testing.T) {
	ce := &CapabilityError{EntitySet: "Products", Operation: "filter", Property: "Price", Message: "not filterable"}
	s := ce.Error()
	if s == "" {
		t.Fatal("Error() should return non-empty string")
	}
}

func TestCapabilityError_NoProperty(t *testing.T) {
	ce := &CapabilityError{EntitySet: "Products", Operation: "sort", Message: "not sortable"}
	s := ce.Error()
	if s == "" {
		t.Fatal("Error() should return non-empty string")
	}
}

// ---------------------------------------------------------------------------
// geo_types.go - Geometry MarshalJSON / UnmarshalJSON
// ---------------------------------------------------------------------------

func TestGeometryPolygon_MarshalUnmarshalJSON(t *testing.T) {
	poly := GeometryPolygon{
		ExteriorRing: []GeometryPoint{
			{X: 0, Y: 0},
			{X: 1, Y: 0},
			{X: 1, Y: 1},
			{X: 0, Y: 1},
			{X: 0, Y: 0},
		},
	}

	data, err := json.Marshal(poly)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var decoded GeometryPolygon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if len(decoded.ExteriorRing) != 5 {
		t.Errorf("expected 5 exterior ring points, got %d", len(decoded.ExteriorRing))
	}
}

func TestGeometryPolygon_WithInteriorRing(t *testing.T) {
	poly := GeometryPolygon{
		ExteriorRing: []GeometryPoint{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}, {X: 0, Y: 0}},
		InteriorRings: [][]GeometryPoint{
			{{X: 2, Y: 2}, {X: 4, Y: 2}, {X: 4, Y: 4}, {X: 2, Y: 4}, {X: 2, Y: 2}},
		},
	}

	data, err := json.Marshal(poly)
	if err != nil {
		t.Fatalf("MarshalJSON with interior ring: %v", err)
	}

	var decoded GeometryPolygon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON with interior ring: %v", err)
	}
	if len(decoded.InteriorRings) != 1 {
		t.Errorf("expected 1 interior ring, got %d", len(decoded.InteriorRings))
	}
}

func TestGeometryPoint_MarshalUnmarshalJSON(t *testing.T) {
	p := GeometryPoint{X: 13.405, Y: 52.52}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var decoded GeometryPoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if decoded.X != p.X || decoded.Y != p.Y {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, p)
	}
}

// ---------------------------------------------------------------------------
// geo_filter.go - GeoDistanceGeom, GeoIntersectsGeom QueryBuilder methods
// ---------------------------------------------------------------------------

func TestQueryBuilder_GeoDistanceGeom(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.From("Places").GeoDistanceGeom("Location", GeometryPoint{X: 10, Y: 20}, "lt", 5000).Collect(context.Background())
	if err != nil {
		t.Fatalf("GeoDistanceGeom: %v", err)
	}

	req := server.RecordedRequests()[0]
	filter := req.Query.Get("$filter")
	if filter == "" {
		t.Error("expected $filter in request")
	}
}

func TestQueryBuilder_GeoDistanceGeom_InvalidOp(t *testing.T) {
	c, err := New(WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	qb := c.From("Places").GeoDistanceGeom("Location", GeometryPoint{X: 10, Y: 20}, "like", 5000)
	if qb.lastError == nil {
		t.Error("expected error for invalid geo operator")
	}
}

func TestQueryBuilder_GeoIntersectsGeom(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	poly := GeometryPolygon{
		ExteriorRing: []GeometryPoint{
			{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}, {X: 0, Y: 0},
		},
	}

	_, err = c.From("Areas").GeoIntersectsGeom("Boundary", poly).Collect(context.Background())
	if err != nil {
		t.Fatalf("GeoIntersectsGeom: %v", err)
	}
}

// ---------------------------------------------------------------------------
// csdl_json.go - parseCSDLNavProperty, parseCSDLEntitySet, parseCSDLFunction
// ---------------------------------------------------------------------------

func TestParseCSDLJSON_NavigationProperty(t *testing.T) {
	csdlJSON := []byte(`{
		"$Version": "4.0",
		"$EntityContainer": "Svc.Container",
		"Svc": {
			"Order": {
				"$Kind": "EntityType",
				"$Key": ["ID"],
				"ID": {"$Type": "Edm.Int32"},
				"Items": {
					"$Kind": "NavigationProperty",
					"$Type": "Collection(Svc.Item)",
					"$Partner": "Order"
				}
			},
			"Item": {
				"$Kind": "EntityType",
				"$Key": ["ID"],
				"ID": {"$Type": "Edm.Int32"}
			},
			"Container": {
				"$Kind": "EntityContainer",
				"Orders": {"$Collection": true, "$Type": "Svc.Order"},
				"Items": {"$Collection": true, "$Type": "Svc.Item"}
			}
		}
	}`)

	md, err := ParseCSDLJSON(csdlJSON)
	if err != nil {
		t.Fatalf("ParseCSDLJSON: %v", err)
	}
	if len(md.EntityTypes) == 0 {
		t.Fatal("expected entity types")
	}
	var order *EntityType
	for i := range md.EntityTypes {
		if md.EntityTypes[i].Name == "Order" {
			order = &md.EntityTypes[i]
			break
		}
	}
	if order == nil {
		t.Fatal("expected Order entity type")
	}
	if len(order.NavigationProperties) == 0 {
		t.Fatal("expected navigation properties on Order")
	}
	nav := order.NavigationProperties[0]
	if nav.Name != "Items" {
		t.Errorf("expected nav property name 'Items', got '%s'", nav.Name)
	}
}

func TestParseCSDLJSON_Function(t *testing.T) {
	csdlJSON := []byte(`{
		"$Version": "4.0",
		"$EntityContainer": "Svc.Container",
		"Svc": {
			"GetTopProducts": [
				{
					"$Kind": "Function",
					"$IsComposable": true,
					"$Parameter": [
						{"$Name": "Count", "$Type": "Edm.Int32"}
					],
					"$ReturnType": {"$Type": "Collection(Svc.Product)"}
				}
			],
			"Product": {
				"$Kind": "EntityType",
				"$Key": ["ID"],
				"ID": {"$Type": "Edm.Int32"}
			},
			"Container": {
				"$Kind": "EntityContainer",
				"Products": {"$Collection": true, "$Type": "Svc.Product"}
			}
		}
	}`)

	md, err := ParseCSDLJSON(csdlJSON)
	if err != nil {
		t.Fatalf("ParseCSDLJSON: %v", err)
	}
	if len(md.Functions) == 0 {
		t.Fatal("expected functions")
	}
	fn := md.Functions[0]
	if fn.Name != "GetTopProducts" {
		t.Errorf("expected function 'GetTopProducts', got '%s'", fn.Name)
	}
	if !fn.IsComposable {
		t.Error("expected IsComposable=true")
	}
	if len(fn.Parameters) == 0 {
		t.Error("expected parameters")
	}
}

// ---------------------------------------------------------------------------
// cache.go - NoOpCache Set and Clear
// ---------------------------------------------------------------------------

func TestNoOpCache_SetAndClear(t *testing.T) {
	c := &NoOpCache{}
	// Set is a no-op - should not panic
	c.Set("key", &Metadata{})
	// Clear is a no-op - should not panic
	c.Clear()
	// Get always returns nil
	got, ok := c.Get("key")
	if ok || got != nil {
		t.Error("NoOpCache.Get should always return nil, false")
	}
}

// ---------------------------------------------------------------------------
// client.go - RelayClient accessor
// ---------------------------------------------------------------------------

func TestClient_RelayClient(t *testing.T) {
	c, err := New(WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	rc := c.RelayClient()
	if rc == nil {
		t.Fatal("RelayClient() should return non-nil")
	}
}

// ---------------------------------------------------------------------------
// options_relay.go - WithHTTPOption
// ---------------------------------------------------------------------------

func TestWithHTTPOption(t *testing.T) {
	// WithHTTPOption accepts a relay.Option - use an existing relay option
	opt := WithHTTPOption(relay.WithTimeout(0))

	c, err := New(WithBaseURL("http://example.com"), opt)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()
}

// Suppress unused import warning if http package isn't used elsewhere.
var _ = http.MethodGet
var _ = fmt.Sprintf
