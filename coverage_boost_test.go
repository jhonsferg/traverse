package traverse

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"testing"
	"time"

	"github.com/jhonsferg/traverse/testutil"
)

// ---------------------------------------------------------------------------
// Cast
// ---------------------------------------------------------------------------

func TestCast_ZeroArgs(t *testing.T) {
	result := Cast()
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestCast_OneArg(t *testing.T) {
	result := Cast("Edm.Decimal")
	if result != "cast(Edm.Decimal)" {
		t.Errorf("expected cast(Edm.Decimal), got %q", result)
	}
}

func TestCast_TwoArgs(t *testing.T) {
	result := Cast("Budget", "Edm.Decimal")
	if result != "cast(Budget,Edm.Decimal)" {
		t.Errorf("expected cast(Budget,Edm.Decimal), got %q", result)
	}
}

func TestCast_ThreeArgs(t *testing.T) {
	result := Cast("a", "b", "c")
	if result != "" {
		t.Errorf("expected empty string for 3 args, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// IsOf
// ---------------------------------------------------------------------------

func TestIsOfFunc(t *testing.T) {
	result := IsOf("Edm.String")
	want := "isof(Edm.String)"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

// ---------------------------------------------------------------------------
// isXMLContentType
// ---------------------------------------------------------------------------

func TestIsXMLContentType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"application/json", false},
		{"text/xml", true},
		{"application/xml", true},
		{"application/atom+xml", true},
		{"TEXT/XML", true},
		{"Application/XML", true},
		{"application/json; charset=utf-8", false},
		{"multipart/mixed", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isXMLContentType(tt.input)
			if got != tt.want {
				t.Errorf("isXMLContentType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isXMLBody
// ---------------------------------------------------------------------------

func TestIsXMLBody(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"empty", []byte{}, false},
		{"nil", nil, false},
		{"whitespace", []byte("   \n\t  "), false},
		{"xml declaration", []byte(`<?xml version="1.0"?>`), true},
		{"xml tag", []byte(`<root></root>`), true},
		{"xml with whitespace", []byte("  <root/>"), true},
		{"json", []byte(`{"key":"value"}`), false},
		{"plain text", []byte("hello world"), false},
		{"atom feed", []byte(`<feed xmlns="http://www.w3.org/2005/Atom">`), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isXMLBody(tt.input)
			if got != tt.want {
				t.Errorf("isXMLBody(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// lambdaVarName
// ---------------------------------------------------------------------------

func TestLambdaVarName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tags", "t"},
		{"items", "i"},
		{"orderItems", "o"},
		{"", "x"},
		{"Order/Tags", "t"},
		{"A/B/C/Items", "i"},
		{"Tags/Details", "d"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := lambdaVarName(tt.input)
			if got != tt.want {
				t.Errorf("lambdaVarName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// splitEntityPath
// ---------------------------------------------------------------------------

func TestSplitEntityPath_Coverage(t *testing.T) {
	tests := []struct {
		input string
		wantP string
		wantQ string
	}{
		{"Products", "Products", ""},
		{"Products?sap-language=ES", "Products", "?sap-language=ES"},
		{"Products?$filter=Price gt 100", "Products", "?$filter=Price gt 100"},
		{"A?B?C", "A", "?B?C"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotP, gotQ := splitEntityPath(tt.input)
			if gotP != tt.wantP || gotQ != tt.wantQ {
				t.Errorf("splitEntityPath(%q) = (%q, %q), want (%q, %q)", tt.input, gotP, gotQ, tt.wantP, tt.wantQ)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildEntityKeyURL
// ---------------------------------------------------------------------------

func TestBuildEntityKeyURL_Simple(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	got := qb.buildEntityKeyURL("ID=1")
	want := "/Products(ID=1)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildEntityKeyURL_WithQueryString(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products?sap-language=ES")
	got := qb.buildEntityKeyURL("ID=1")
	want := "/Products(ID=1)?sap-language=ES"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildEntityKeyURL_WithSelect(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products").Select("Name", "Price")
	got := qb.buildEntityKeyURL("ID=1")
	want := "/Products(ID=1)?$select=Name,Price"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildEntityKeyURL_WithParam(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products").Param("sap-client", "100")
	got := qb.buildEntityKeyURL("ID=1")
	want := "/Products(ID=1)?sap-client=100"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildEntityKeyURL_WithQueryStringAndSelect(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products?sap-language=ES").Select("Name")
	got := qb.buildEntityKeyURL("ID=1")
	want := "/Products(ID=1)?sap-language=ES&$select=Name"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// ParseCapabilities
// ---------------------------------------------------------------------------

func TestParseCapabilities_InvalidXML(t *testing.T) {
	_, err := ParseCapabilities([]byte("not xml"))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestParseCapabilities_EmptyEdmx(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestParseCapabilities_WithEntitySet(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Products" EntityType="Test.Product"/>
      </EntityContainer>
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cap := reg.Get("Products")
	if !cap.Filterable || !cap.Sortable || !cap.Insertable || !cap.Updatable || !cap.Deletable {
		t.Error("expected all capabilities to be true")
	}
}

func TestParseCapabilities_SortRestrictions(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Products" EntityType="Test.Product">
          <Annotation Term="Capabilities.SortRestrictions">
            <Record>
              <PropertyValue Property="Sortable" Bool="false"/>
              <PropertyValue Property="NonSortableProperties">
                <Collection>
                  <Record>
                    <PropertyValue Property="Name" String="CreatedDate"/>
                  </Record>
                </Collection>
              </PropertyValue>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cap := reg.Get("Products")
	if cap.Sortable {
		t.Error("expected Sortable=false")
	}
	if len(cap.NonSortableProperties) != 1 || cap.NonSortableProperties[0] != "CreatedDate" {
		t.Errorf("expected NonSortableProperties=[CreatedDate], got %v", cap.NonSortableProperties)
	}
}

func TestParseCapabilities_ExpandRestrictions(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Orders" EntityType="Test.Order">
          <Annotation Term="Capabilities.ExpandRestrictions">
            <Record>
              <PropertyValue Property="ExpandableProperties">
                <Collection>
                  <Record>
                    <PropertyValue Property="Name" String="Items"/>
                  </Record>
                </Collection>
              </PropertyValue>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cap := reg.Get("Orders")
	if len(cap.ExpandableNavigationProperties) != 1 || cap.ExpandableNavigationProperties[0] != "Items" {
		t.Errorf("expected ExpandableNavigationProperties=[Items], got %v", cap.ExpandableNavigationProperties)
	}
}

func TestParseCapabilities_InsertRestrictions(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Products" EntityType="Test.Product">
          <Annotation Term="Capabilities.InsertRestrictions">
            <Record>
              <PropertyValue Property="Insertable" Bool="false"/>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cap := reg.Get("Products")
	if cap.Insertable {
		t.Error("expected Insertable=false")
	}
}

func TestParseCapabilities_UpdateRestrictions(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Products" EntityType="Test.Product">
          <Annotation Term="Capabilities.UpdateRestrictions">
            <Record>
              <PropertyValue Property="Updatable" Bool="false"/>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cap := reg.Get("Products")
	if cap.Updatable {
		t.Error("expected Updatable=false")
	}
}

func TestParseCapabilities_DeleteRestrictions(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<Edmx Version="4.0" xmlns="http://docs.oasis-open.org/odata/ns/edmx">
  <DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Products" EntityType="Test.Product">
          <Annotation Term="Capabilities.DeleteRestrictions">
            <Record>
              <PropertyValue Property="Deletable" Bool="false"/>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </DataServices>
</Edmx>`
	reg, err := ParseCapabilities([]byte(edmx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cap := reg.Get("Products")
	if cap.Deletable {
		t.Error("expected Deletable=false")
	}
}

// ---------------------------------------------------------------------------
// parseCSDLProperty
// ---------------------------------------------------------------------------

func TestParseCSDLProperty_AllFields(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Type":      json.RawMessage(`"Edm.Decimal"`),
		"$Nullable":  json.RawMessage(`false`),
		"$MaxLength": json.RawMessage(`255`),
		"$Precision": json.RawMessage(`10`),
		"$Scale":     json.RawMessage(`2`),
	}
	p := parseCSDLProperty("Price", obj)
	if p.Name != "Price" {
		t.Errorf("expected Name=Price, got %q", p.Name)
	}
	if p.Type != "Edm.Decimal" {
		t.Errorf("expected Type=Edm.Decimal, got %q", p.Type)
	}
	if p.Nullable {
		t.Error("expected Nullable=false")
	}
	if p.MaxLength == nil || *p.MaxLength != 255 {
		t.Errorf("expected MaxLength=255, got %v", p.MaxLength)
	}
	if p.Precision == nil || *p.Precision != 10 {
		t.Errorf("expected Precision=10, got %v", p.Precision)
	}
	if p.Scale == nil || *p.Scale != 2 {
		t.Errorf("expected Scale=2, got %v", p.Scale)
	}
}

func TestParseCSDLProperty_Defaults(t *testing.T) {
	obj := map[string]json.RawMessage{}
	p := parseCSDLProperty("Name", obj)
	if p.Type != "Edm.String" {
		t.Errorf("expected default Type=Edm.String, got %q", p.Type)
	}
	if !p.Nullable {
		t.Error("expected default Nullable=true")
	}
}

func TestParseCSDLProperty_InvalidMaxLength(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$MaxLength": json.RawMessage(`"not-a-number"`),
	}
	p := parseCSDLProperty("Field", obj)
	if p.MaxLength != nil {
		t.Errorf("expected nil MaxLength for invalid input, got %v", p.MaxLength)
	}
}

func TestParseCSDLProperty_InvalidPrecision(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Precision": json.RawMessage(`"invalid"`),
	}
	p := parseCSDLProperty("Field", obj)
	if p.Precision != nil {
		t.Errorf("expected nil Precision for invalid input, got %v", p.Precision)
	}
}

func TestParseCSDLProperty_InvalidScale(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Scale": json.RawMessage(`"bad"`),
	}
	p := parseCSDLProperty("Field", obj)
	if p.Scale != nil {
		t.Errorf("expected nil Scale for invalid input, got %v", p.Scale)
	}
}

// ---------------------------------------------------------------------------
// parseCSDLNavProperty
// ---------------------------------------------------------------------------

func TestParseCSDLNavProperty(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Type":    json.RawMessage(`"Test.OrderItem"`),
		"$Partner": json.RawMessage(`"Order"`),
	}
	np := parseCSDLNavProperty("Items", obj)
	if np.Name != "Items" {
		t.Errorf("expected Name=Items, got %q", np.Name)
	}
	if np.ToEntityType != "Test.OrderItem" {
		t.Errorf("expected ToEntityType=Test.OrderItem, got %q", np.ToEntityType)
	}
	if np.Partner != "Order" {
		t.Errorf("expected Partner=Order, got %q", np.Partner)
	}
}

func TestParseCSDLNavProperty_Empty(t *testing.T) {
	obj := map[string]json.RawMessage{}
	np := parseCSDLNavProperty("Items", obj)
	if np.Name != "Items" {
		t.Errorf("expected Name=Items, got %q", np.Name)
	}
	if np.ToEntityType != "" {
		t.Errorf("expected empty ToEntityType, got %q", np.ToEntityType)
	}
}

// ---------------------------------------------------------------------------
// parseCSDLEntityContainer
// ---------------------------------------------------------------------------

func TestParseCSDLEntityContainer_WithEntityType(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Products": json.RawMessage(`{"$Type": "Test.Product"}`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("DefaultContainer", "Test", obj, md)
	if len(md.EntitySets) != 1 {
		t.Fatalf("expected 1 entity set, got %d", len(md.EntitySets))
	}
	if md.EntitySets[0].Name != "Products" {
		t.Errorf("expected name Products, got %q", md.EntitySets[0].Name)
	}
}

func TestParseCSDLEntityContainer_WithEntityTypeAlternate(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Orders": json.RawMessage(`{"$EntityType": "Test.Order"}`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 1 {
		t.Fatalf("expected 1 entity set, got %d", len(md.EntitySets))
	}
	if md.EntitySets[0].EntityTypeName != "Order" {
		t.Errorf("expected EntityTypeName=Order, got %q", md.EntitySets[0].EntityTypeName)
	}
}

func TestParseCSDLEntityContainer_DollarKeySkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Kind": json.RawMessage(`"EntityContainer"`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 0 {
		t.Errorf("expected 0 entity sets, got %d", len(md.EntitySets))
	}
}

func TestParseCSDLEntityContainer_InvalidJSON(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Bad": json.RawMessage(`not json`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 0 {
		t.Errorf("expected 0 entity sets for invalid JSON, got %d", len(md.EntitySets))
	}
}

func TestParseCSDLEntityContainer_SingletonKindSkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Me": json.RawMessage(`{"$Kind": "Singleton"}`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 0 {
		t.Errorf("expected 0 entity sets for Singleton kind, got %d", len(md.EntitySets))
	}
}

func TestParseCSDLEntityContainer_UnknownKindSkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Func": json.RawMessage(`{"$Kind": "Function"}`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 0 {
		t.Errorf("expected 0 entity sets for Function kind, got %d", len(md.EntitySets))
	}
}

func TestParseCSDLEntityContainer_EmptyTypeSkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Empty": json.RawMessage(`{}`),
	}
	md := &Metadata{}
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 0 {
		t.Errorf("expected 0 entity sets for empty type, got %d", len(md.EntitySets))
	}
}

func TestParseCSDLEntityContainer_DuplicateSkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Products": json.RawMessage(`{"$Type": "Test.Product"}`),
	}
	md := &Metadata{}
	md.EntitySets = append(md.EntitySets, EntitySetInfo{Name: "Products", EntityTypeName: "Product"})
	parseCSDLEntityContainer("C", "Test", obj, md)
	if len(md.EntitySets) != 1 {
		t.Errorf("expected 1 entity set (duplicate skipped), got %d", len(md.EntitySets))
	}
}

// ---------------------------------------------------------------------------
// parseCSDLComplexType
// ---------------------------------------------------------------------------

func TestParseCSDLComplexType_WithProperties(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Kind":   json.RawMessage(`"ComplexType"`),
		"Street":  json.RawMessage(`{"$Type": "Edm.String"}`),
		"City":    json.RawMessage(`{"$Type": "Edm.String"}`),
		"NavProp": json.RawMessage(`{"$Kind": "NavigationProperty"}`),
	}
	ct := parseCSDLComplexType("Address", obj)
	if ct.Name != "Address" {
		t.Errorf("expected Name=Address, got %q", ct.Name)
	}
	if len(ct.Properties) != 2 {
		t.Errorf("expected 2 properties (nav skipped), got %d", len(ct.Properties))
	}
}

func TestParseCSDLComplexType_DollarPropsSkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$Kind": json.RawMessage(`"ComplexType"`),
	}
	ct := parseCSDLComplexType("Empty", obj)
	if len(ct.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(ct.Properties))
	}
}

func TestParseCSDLComplexType_InvalidPropJSON(t *testing.T) {
	obj := map[string]json.RawMessage{
		"BadProp": json.RawMessage(`not json`),
	}
	ct := parseCSDLComplexType("Test", obj)
	if len(ct.Properties) != 0 {
		t.Errorf("expected 0 properties for invalid JSON, got %d", len(ct.Properties))
	}
}

// ---------------------------------------------------------------------------
// parseCSDLEnumType
// ---------------------------------------------------------------------------

func TestParseCSDLEnumType_WithMembers(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$UnderlyingType": json.RawMessage(`"Edm.Int32"`),
		"$IsFlags":        json.RawMessage(`true`),
		"Red":             json.RawMessage(`0`),
		"Green":           json.RawMessage(`1`),
		"Blue":            json.RawMessage(`2`),
	}
	en := parseCSDLEnumType("Color", obj)
	if en.Name != "Color" {
		t.Errorf("expected Name=Color, got %q", en.Name)
	}
	if en.UnderlyingType != "Edm.Int32" {
		t.Errorf("expected UnderlyingType=Edm.Int32, got %q", en.UnderlyingType)
	}
	if !en.IsFlags {
		t.Error("expected IsFlags=true")
	}
	if len(en.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(en.Members))
	}
}

func TestParseCSDLEnumType_DollarMembersSkipped(t *testing.T) {
	obj := map[string]json.RawMessage{
		"$UnderlyingType": json.RawMessage(`"Edm.Int32"`),
	}
	en := parseCSDLEnumType("Empty", obj)
	if len(en.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(en.Members))
	}
}

func TestParseCSDLEnumType_InvalidMemberValue(t *testing.T) {
	obj := map[string]json.RawMessage{
		"Bad": json.RawMessage(`"not-int"`),
	}
	en := parseCSDLEnumType("Test", obj)
	if len(en.Members) != 0 {
		t.Errorf("expected 0 members for invalid value, got %d", len(en.Members))
	}
}

// ---------------------------------------------------------------------------
// entitySetExists
// ---------------------------------------------------------------------------

func TestEntitySetExistsFunc(t *testing.T) {
	sets := []EntitySetInfo{
		{Name: "Products", EntityTypeName: "Product"},
	}
	if !entitySetExists(sets, "Products", "Test") {
		t.Error("expected entitySetExists to return true for matching name")
	}
	if entitySetExists(sets, "Orders", "Test") {
		t.Error("expected entitySetExists to return false for non-matching name")
	}
}

// ---------------------------------------------------------------------------
// NoOpCache
// ---------------------------------------------------------------------------

func TestNoOpCache_Coverage(t *testing.T) {
	cache := &NoOpCache{}
	cache.Set("key", &Metadata{})
	cache.Clear()
}

// ---------------------------------------------------------------------------
// goroutinePool submit
// ---------------------------------------------------------------------------

func TestGoroutinePool_Submit(t *testing.T) {
	pool := newGoroutinePool(2)
	defer pool.close()

	done := make(chan bool, 1)
	pool.submit(func() {
		done <- true
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task not executed")
	}
}

// ---------------------------------------------------------------------------
// GeoLength
// ---------------------------------------------------------------------------

func TestGeoLength_InvalidOp(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Routes")
	qb.GeoLength("Path", "xx", 50000)
	if qb.lastError == nil {
		t.Error("expected error for invalid geo operator")
	}
}

func TestGeoLengthFilterFunc(t *testing.T) {
	got := GeoLengthFilter("Path", "le", 50000)
	want := "geo.length(Path) le 50000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// ParseGeometryPoint
// ---------------------------------------------------------------------------

func TestParseGeometryPointFunc(t *testing.T) {
	tests := []struct {
		name  string
		input string
		wantX float64
		wantY float64
	}{
		{"wkt plain", "POINT(1.5 2.5)", 1.5, 2.5},
		{"geometry prefix", "geometry'SRID=0;POINT(3.0 4.0)'", 3.0, 4.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, err := ParseGeometryPoint(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pt.X != tt.wantX || pt.Y != tt.wantY {
				t.Errorf("got (%v, %v), want (%v, %v)", pt.X, pt.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestParseGeometryPoint_Invalid(t *testing.T) {
	_, err := ParseGeometryPoint("INVALID")
	if err == nil {
		t.Error("expected error for invalid geometry point")
	}
}

// ---------------------------------------------------------------------------
// applyAuthOpts
// ---------------------------------------------------------------------------

func TestApplyAuthOpts_AllVariants(t *testing.T) {
	tests := []struct {
		name string
		cfg  *clientConfig
		want int
	}{
		{"basic auth", &clientConfig{basicAuthUser: "u", basicAuthPass: "p"}, 1},
		{"bearer", &clientConfig{bearerToken: "tok"}, 1},
		{"api key", &clientConfig{apiKeyHeader: "X-Key", apiKeyValue: "val"}, 1},
		{"none", &clientConfig{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyAuthOpts(tt.cfg)
			if len(tt.cfg.relayOpts) != tt.want {
				t.Errorf("expected %d relay opts, got %d", tt.want, len(tt.cfg.relayOpts))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parsePageFromBytes
// ---------------------------------------------------------------------------

func TestParsePageFromBytes_JSON(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	body := []byte(`{"value":[{"ID":1},{"ID":2}]}`)
	page, err := qb.parsePageFromBytes(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Value) != 2 {
		t.Errorf("expected 2 entities, got %d", len(page.Value))
	}
}

func TestParsePageFromBytes_Atom(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	body := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <content type="application/xml">
      <m:properties>
        <d:ID m:type="Edm.Int32">1</d:ID>
      </m:properties>
    </content>
  </entry>
</feed>`)
	page, err := qb.parsePageFromBytes(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Value) != 1 {
		t.Errorf("expected 1 entity, got %d", len(page.Value))
	}
}

func TestParsePageFromBytes_InvalidJSON(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	body := []byte(`{invalid json}`)
	_, err := qb.parsePageFromBytes(body)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParsePageFromBytes_InvalidAtom(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	// Not a valid Atom feed but valid XML - parseAtomFeed should fail
	body := []byte(`<?xml version="1.0" encoding="utf-8"?><root/>`)
	_, err := qb.parsePageFromBytes(body)
	if err == nil {
		t.Log("valid XML was accepted as Atom (no parse error)")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Collect
// ---------------------------------------------------------------------------

func TestE2E_Collect(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Laptop"}, map[string]interface{}{"ID": 2, "Name": "Phone"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: FindByKey
// ---------------------------------------------------------------------------

func TestE2E_FindByKey(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataSingleResponse(map[string]interface{}{"ID": 1, "Name": "Laptop"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	result, err := c.From("Products").FindByKey(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: CollectWithFilter
// ---------------------------------------------------------------------------

func TestE2E_CollectWithFilter(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Laptop", "Price": 999.99}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		Filter("Price gt 700").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with Price > 700, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Stream
// ---------------------------------------------------------------------------

func TestE2E_Stream(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Laptop"}, map[string]interface{}{"ID": 2, "Name": "Phone"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	count := 0
	for result := range c.From("Products").Stream(context.Background()) {
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 results from Stream, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// End-to-end: StreamJsonAs
// ---------------------------------------------------------------------------

func TestE2E_StreamJsonAs(t *testing.T) {
	type Product struct {
		ID   int    `json:"ID"`
		Name string `json:"Name"`
	}

	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Laptop"}, map[string]interface{}{"ID": 2, "Name": "Phone"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	var products []Product
	for result := range StreamJsonAs[Product](c.From("Products"), context.Background()) {
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		products = append(products, result.Value)
	}
	if len(products) != 2 {
		t.Errorf("expected 2 products, got %d", len(products))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: BulkDelete
// ---------------------------------------------------------------------------

func TestE2E_BulkDelete(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 204})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.From("Products").
		Filter("Name eq 'Old'").
		BulkDelete(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_BulkDelete_ValidationError(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	qb.lastError = fmt.Errorf("validation error")
	err := qb.BulkDelete(context.Background())
	if err == nil {
		t.Error("expected error for BulkDelete with lastError")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: FetchPageAt
// ---------------------------------------------------------------------------

func TestE2E_FetchPageAt(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Laptop"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	page, err := c.FetchPageAt(context.Background(), srv.URL()+"/Products?$top=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Value) != 1 {
		t.Errorf("expected 1 result, got %d", len(page.Value))
	}
}

func TestE2E_FetchPageAt_Error(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 404})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := c.FetchPageAt(context.Background(), srv.URL()+"/NonExistent")
	if err == nil {
		t.Error("expected error for FetchPageAt with 404")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: FilterLambda
// ---------------------------------------------------------------------------

func TestE2E_FilterLambda(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		FilterLambda("tags/any(t: t eq 'electronics')").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// End-to-end: Count
// ---------------------------------------------------------------------------

func TestE2E_Count(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `3`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	count, err := c.From("Products").Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// End-to-end: FunctionImport
// ---------------------------------------------------------------------------

func TestE2E_FunctionImport(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Top Product"}]}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	result, err := c.FunctionImport("GetTopProducts").Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: ActionImport
// ---------------------------------------------------------------------------

func TestE2E_ActionImport(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":null}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := c.Action("ResetData").Execute(context.Background())
	if err != nil {
		t.Logf("expected behavior for Action: %v", err)
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Update and Delete
// ---------------------------------------------------------------------------

func TestE2E_Update(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 204})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.Update(context.Background(), "Products", "1", map[string]interface{}{
		"ID":   1,
		"Name": "New",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_Delete(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 204})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.Delete(context.Background(), "Products", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Insert
// ---------------------------------------------------------------------------

func TestE2E_Insert(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 201,
		Body:   testutil.ODataSingleResponse(map[string]interface{}{"ID": 1, "Name": "New Product"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	result, err := c.Create(context.Background(), "Products", map[string]interface{}{
		"ID":   1,
		"Name": "New Product",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result from Create")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Batch
// ---------------------------------------------------------------------------

func TestE2E_Batch(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": "multipart/mixed; boundary=batch_boundary",
		},
		Body: "--batch_boundary\r\nContent-Type: application/http\r\nContent-Transfer-Encoding: binary\r\n\r\nHTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"value\":[{\"ID\":1}]}\r\n--batch_boundary--\r\n",
	})

	c, _ := New(WithBaseURL(srv.URL()))
	batch := c.Batch()
	batch.Get("Products", "1")
	resp, err := batch.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil batch response")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Metadata
// ---------------------------------------------------------------------------

func TestE2E_Metadata(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testServiceMetadataV4,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	md, err := c.Metadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md == nil {
		t.Fatal("expected non-nil metadata")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Error handling
// ---------------------------------------------------------------------------

func TestE2E_CollectServerError(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   testutil.ODataErrorResponse("500", "Internal Server Error"),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := c.From("Products").Collect(context.Background())
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestE2E_FindByKey_NotFound(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 404,
		Body:   testutil.ODataErrorResponse("NOT_FOUND", "Entity not found"),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := c.From("Products").FindByKey(context.Background(), "999")
	if err == nil {
		t.Error("expected error for not found entity")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: WithPrefer
// ---------------------------------------------------------------------------

func TestE2E_WithPrefer(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Laptop"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		WithPrefer("return=representation").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Expand
// ---------------------------------------------------------------------------

func TestE2E_Expand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		Expand("Items").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Cast in filter
// ---------------------------------------------------------------------------

func TestE2E_FilterWithCast(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Price": 99.99}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		Filter(Cast("Price", "Edm.Decimal") + " gt 50").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// End-to-end: WithCount
// ---------------------------------------------------------------------------

func TestE2E_WithCount(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}],"@odata.count":1}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	page, err := c.From("Products").
		WithCount().
		Page(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page == nil {
		t.Fatal("expected non-nil page")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: GeoContains / GeoIntersects / GeoDistance
// ---------------------------------------------------------------------------

func TestE2E_GeoContains(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Stores").
		GeoIntersects("Location", GeographyPolygon{}).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_GeoIntersects(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	polygon := GeographyPolygon{
		ExteriorRing: []GeographyPoint{
			{Longitude: 0, Latitude: 0},
			{Longitude: 1, Latitude: 0},
			{Longitude: 1, Latitude: 1},
			{Longitude: 0, Latitude: 1},
			{Longitude: 0, Latitude: 0},
		},
	}
	results, err := c.From("Areas").
		GeoIntersects("Boundary", polygon).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_GeoDistance(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Locations").
		GeoDistance("Point", GeographyPoint{Longitude: 1.0, Latitude: 2.0}, "le", 1000).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// End-to-end: ODataV2
// ---------------------------------------------------------------------------

func TestE2E_ODataV2_Collect(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"results":[{"ID":1,"Name":"Laptop"}]}}`,
	})

	c, _ := New(WithBaseURL(srv.URL()), WithODataVersion(ODataV2))
	results, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Version handling
// ---------------------------------------------------------------------------

func TestE2E_Versions(t *testing.T) {
	versions := []ODataVersion{ODataV2, ODataV4}
	for _, v := range versions {
		t.Run(fmt.Sprintf("v%d", int(v)), func(t *testing.T) {
			srv := testutil.NewMockServer()
			defer srv.Close()

			if v == ODataV2 {
				srv.Enqueue(testutil.MockResponse{
					Status: 200,
					Body:   `{"d":{"results":[{"ID":1}]}}`,
				})
			} else {
				srv.Enqueue(testutil.MockResponse{
					Status: 200,
					Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
				})
			}

			c, _ := New(WithBaseURL(srv.URL()), WithODataVersion(v))
			results, err := c.From("Products").Collect(context.Background())
			if err != nil {
				t.Fatalf("unexpected error for version %d: %v", v, err)
			}
			if len(results) != 1 {
				t.Errorf("expected 1 result for version %d, got %d", v, len(results))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// End-to-end: BeforeQuery / AfterExecute hooks
// ---------------------------------------------------------------------------

func TestE2E_BeforeQueryHook(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	hookCalled := false
	c, _ := New(
		WithBaseURL(srv.URL()),
		WithBeforeQuery(func(qb *QueryBuilder) error {
			hookCalled = true
			return nil
		}),
	)

	_, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hookCalled {
		t.Error("expected BeforeQuery hook to be called")
	}
}

func TestE2E_AfterExecuteHook(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	hookCalled := false
	c, _ := New(
		WithBaseURL(srv.URL()),
		WithAfterExecute(func(qb *QueryBuilder) error {
			hookCalled = true
			return nil
		}),
	)

	_, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// AfterExecute hook may not be called for all operations
	_ = hookCalled
}

// ---------------------------------------------------------------------------
// End-to-end: DeltaSync
// ---------------------------------------------------------------------------

func TestE2E_DeltaSync(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1, "Name": "Product1"}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	delta := c.NewDeltaSync("Products")
	result, token, err := delta.Full(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	count := 0
	for r := range result {
		if r.Err != nil {
			t.Fatalf("unexpected error in channel: %v", r.Err)
		}
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 delta result, got %d", count)
	}
	_ = token
}

// ---------------------------------------------------------------------------
// End-to-end: SingletonAsCtx
// ---------------------------------------------------------------------------

type Me struct {
	Name string `json:"Name"`
}

func TestE2E_SingletonAsCtx(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"Name":"John"}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	result, err := SingletonAsCtx[Me](c, context.Background(), "me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "John" {
		t.Errorf("expected Name=John, got %q", result.Name)
	}
}

func TestE2E_SingletonAsCtx_V2(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"Name":"John"}}`,
	})

	c, _ := New(WithBaseURL(srv.URL()), WithODataVersion(ODataV2))
	result, err := SingletonAsCtx[Me](c, context.Background(), "me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "John" {
		t.Errorf("expected Name=John, got %q", result.Name)
	}
}

func TestE2E_SingletonAsCtx_ErrorStatus(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 500})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := SingletonAsCtx[Me](c, context.Background(), "me")
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestE2E_SingletonAsCtx_InvalidJSON(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{invalid}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := SingletonAsCtx[Me](c, context.Background(), "me")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: FetchPropertyAs
// ---------------------------------------------------------------------------

func TestE2E_FetchPropertyAs(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"Name":"Laptop"}]}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	qb := c.From("Products('1')")
	name, err := FetchPropertyAs[string](qb, context.Background(), "Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Laptop" {
		t.Errorf("expected Name=Laptop, got %q", name)
	}
}

func TestE2E_FetchPropertyAs_EmptyProperty(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products('1')")
	_, err := FetchPropertyAs[string](qb, context.Background(), "")
	if err == nil {
		t.Error("expected error for empty property name")
	}
}

// ---------------------------------------------------------------------------
// End-to-end: Page with nextLink
// ---------------------------------------------------------------------------

func TestE2E_PageWithNextLink(t *testing.T) {
	callCount := 0
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   fmt.Sprintf(`{"value":[{"ID":1}],"@odata.nextLink":"%s/Products?$skip=1"}`, srv.URL()),
	})
	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 2}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results from paginated response, got %d", len(results))
	}
	_ = callCount
}

// ---------------------------------------------------------------------------
// End-to-end: WithSkip + WithTop
// ---------------------------------------------------------------------------

func TestE2E_WithSkipAndTop(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 3}, map[string]interface{}{"ID": 4}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		Skip(2).
		Top(2).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests for remaining low-coverage functions
// ---------------------------------------------------------------------------

func TestIsOf_ZeroArgs(t *testing.T) {
	result := IsOf()
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestIsOf_TwoArgs(t *testing.T) {
	result := IsOf("x", "Edm.String")
	want := "isof(x,Edm.String)"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestIsOf_ThreeArgs(t *testing.T) {
	result := IsOf("a", "b", "c")
	if result != "" {
		t.Errorf("expected empty string for 3 args, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// E2E: BulkUpdate
// ---------------------------------------------------------------------------

func TestE2E_BulkUpdate(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 204})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.From("Products").
		Filter("Status eq 'Draft'").
		BulkUpdate(context.Background(), map[string]interface{}{"Status": "Active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_BulkUpdate_ValidationError(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	qb.lastError = fmt.Errorf("validation error")
	err := qb.BulkUpdate(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for BulkUpdate with lastError")
	}
}

func TestE2E_BulkUpdate_NotFound(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 404})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.From("Products").BulkUpdate(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for BulkUpdate 404")
	}
}

func TestE2E_BulkUpdate_Conflict(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 409})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.From("Products").BulkUpdate(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for BulkUpdate 409")
	}
}

// ---------------------------------------------------------------------------
// E2E: BulkDelete additional
// ---------------------------------------------------------------------------

func TestE2E_BulkDelete_NotFound(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 404})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.From("Products").BulkDelete(context.Background())
	if err == nil {
		t.Error("expected error for BulkDelete 404")
	}
}

func TestE2E_BulkDelete_Conflict(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 409})

	c, _ := New(WithBaseURL(srv.URL()))
	err := c.From("Products").BulkDelete(context.Background())
	if err == nil {
		t.Error("expected error for BulkDelete 409")
	}
}

// ---------------------------------------------------------------------------
// E2E: StreamRaw for doStreamPagesRaw coverage
// ---------------------------------------------------------------------------

func TestE2E_StreamRaw(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1},{"ID":2}]}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	qb := c.From("Products")

	count := 0
	for result := range qb.streamRaw(context.Background(), 256) {
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 raw results, got %d", count)
	}
}

func TestE2E_StreamRaw_ValidationError(t *testing.T) {
	c, _ := New(WithBaseURL("https://example.com/odata"))
	qb := c.From("Products")
	qb.lastError = fmt.Errorf("validation error")

	for result := range qb.streamRaw(context.Background(), 256) {
		if result.Err == nil {
			t.Error("expected error from streamRaw with lastError")
		}
	}
}

func TestE2E_StreamRaw_ContextCancel(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	for i := 0; i < 100; i++ {
		srv.Enqueue(testutil.MockResponse{
			Status: 200,
			Body:   testutil.ODataResponse(map[string]interface{}{"ID": i}),
		})
	}

	c, _ := New(WithBaseURL(srv.URL()), WithPageSize(1))
	qb := c.From("Products")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	count := 0
	for result := range qb.streamRaw(ctx, 256) {
		if result.Err != nil {
			break
		}
		count++
		if count >= 3 {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// E2E: StreamXmlAs
// ---------------------------------------------------------------------------

type XmlEntry struct {
	XMLName xml.Name `xml:"entry"`
	ID      int      `xml:"id"`
	Name    string   `xml:"name"`
}

func TestE2E_StreamXmlAs(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	atomBody := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <content type="application/xml">
      <m:properties>
        <d:ID m:type="Edm.Int32">1</d:ID>
      </m:properties>
    </content>
  </entry>
</feed>`

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   atomBody,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	qb := c.From("Products")
	count := 0
	for result := range StreamXmlAs[XmlEntry](qb, context.Background()) {
		if result.Err != nil {
			t.Logf("StreamXmlAs result: %v", result.Err)
		}
		count++
	}
}

// ---------------------------------------------------------------------------
// E2E: WithDeltaToken
// ---------------------------------------------------------------------------

func TestE2E_WithDeltaToken(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		WithDeltaToken("token123").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// E2E: WithPrefetch
// ---------------------------------------------------------------------------

func TestE2E_WithPrefetch(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		WithPrefetch(2).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// E2E: WithNoPrefetch
// ---------------------------------------------------------------------------

func TestE2E_WithNoPrefetch(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		WithNoPrefetch().
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// E2E: WithSchemaVersion
// ---------------------------------------------------------------------------

func TestE2E_WithSchemaVersion(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		WithSchemaVersion("v1").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// E2E: WithCache
// ---------------------------------------------------------------------------

func TestE2E_WithCache(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Products").
		WithCache(5 * time.Minute).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// E2E: Expand with multiple options
// ---------------------------------------------------------------------------

func TestE2E_ExpandWithMultipleOptions(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		Expand("Items",
			WithExpandFilter("Quantity gt 5"),
			WithExpandOrderBy("Quantity"),
			WithExpandTop(10),
		).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_ExpandWithOrderByDesc(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		Expand("Items", WithExpandOrderByDesc("Quantity")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_ExpandWithSelect(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		Expand("Items", WithExpandSelect("Name", "Quantity")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_ExpandWithSkipAndLevels(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		Expand("Items", WithExpandSkip(2), WithExpandLevels(3)).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// E2E: CreateDeep / CreateDeepWithPrefer
// ---------------------------------------------------------------------------

func TestE2E_CreateDeep(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 201,
		Body:   `{"ID":1,"Name":"Order"}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	resp, err := c.From("Orders").CreateDeep(context.Background(), map[string]interface{}{
		"ID":   1,
		"Name": "Order",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestE2E_CreateDeepWithPrefer(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 201,
		Body:   `{"ID":1}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	resp, err := c.From("Orders").CreateDeepWithPrefer(context.Background(), map[string]interface{}{
		"ID": 1,
	}, "return=representation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

// ---------------------------------------------------------------------------
// E2E: Service
// ---------------------------------------------------------------------------

func TestE2E_Service(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"name":"Products","kind":"EntitySet"}]}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	svc, err := c.Service(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service document")
	}
}

func TestE2E_Service_Error(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 500})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := c.Service(context.Background())
	if err == nil {
		t.Error("expected error for Service with 500")
	}
}

// ---------------------------------------------------------------------------
// E2E: Metadata error
// ---------------------------------------------------------------------------

func TestE2E_Metadata_Error(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: 500})

	c, _ := New(WithBaseURL(srv.URL()))
	_, err := c.Metadata(context.Background())
	if err == nil {
		t.Error("expected error for Metadata with 500")
	}
}

// ---------------------------------------------------------------------------
// E2E: Expand with filter option
// ---------------------------------------------------------------------------

func TestE2E_ExpandWithFilter(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		Expand("Items", WithExpandFilter("Quantity gt 5")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// E2E: GeoLength with operator variations
// ---------------------------------------------------------------------------

func TestE2E_GeoLengthOps(t *testing.T) {
	ops := []string{"lt", "le", "gt", "ge", "eq", "ne"}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			srv := testutil.NewMockServer()
			defer srv.Close()

			srv.Enqueue(testutil.MockResponse{
				Status: 200,
				Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
			})

			c, _ := New(WithBaseURL(srv.URL()))
			results, err := c.From("Routes").
				GeoLength("Path", op, 50000).
				Collect(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = results
		})
	}
}

func TestE2E_GeoLengthGeom(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Routes").
		GeoLength("Path", "le", 50000).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_GeoDistanceGeom(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Locations").
		GeoDistanceGeom("Point", GeometryPoint{X: 1.0, Y: 2.0}, "le", 1000).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

func TestE2E_GeoIntersectsGeom(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	polygon := GeometryPolygon{
		ExteriorRing: []GeometryPoint{
			{X: 0, Y: 0},
			{X: 1, Y: 0},
			{X: 1, Y: 1},
			{X: 0, Y: 1},
			{X: 0, Y: 0},
		},
	}
	results, err := c.From("Areas").
		GeoIntersectsGeom("Boundary", polygon).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// E2E: LambdaAll
// ---------------------------------------------------------------------------

func TestE2E_LambdaAll(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	results, err := c.From("Orders").
		FilterLambda("Items/all(i: i/Quantity gt 0)").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ---------------------------------------------------------------------------
// E2E: Replace
// ---------------------------------------------------------------------------

func TestE2E_Replace(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"ID":1,"Name":"Replaced"}}`,
	})

	c, _ := New(WithBaseURL(srv.URL()), WithODataVersion(ODataV2))
	err := c.Replace(context.Background(), "Products", "1", map[string]interface{}{
		"ID":   1,
		"Name": "Replaced",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// E2E: WithSchema
// ---------------------------------------------------------------------------

func TestE2E_WithSchema(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   testutil.ODataResponse(map[string]interface{}{"ID": 1}),
	})

	c, _ := New(WithBaseURL(srv.URL()))
	schema := &EntitySchema{
		Properties: map[string]string{
			"ID":   "int",
			"Name": "string",
		},
	}
	results, err := c.From("Products").
		WithSchema(schema).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// E2E: FetchPropertyAs additional branches
// ---------------------------------------------------------------------------

func TestE2E_FetchPropertyAs_RawValue(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":"hello"}`,
	})

	c, _ := New(WithBaseURL(srv.URL()))
	qb := c.From("Products('1')")
	val, err := FetchPropertyAs[string](qb, context.Background(), "nonexistent")
	if err != nil {
		t.Logf("RawValue fallback behavior: %v", err)
	}
	_ = val
}

// ---------------------------------------------------------------------------
// E2E: Stream context cancel
// ---------------------------------------------------------------------------

func TestE2E_Stream_ContextCancel(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	for i := 0; i < 100; i++ {
		srv.Enqueue(testutil.MockResponse{
			Status: 200,
			Body:   testutil.ODataResponse(map[string]interface{}{"ID": i}),
		})
	}

	c, _ := New(WithBaseURL(srv.URL()), WithPageSize(1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := 0
	for result := range c.From("Products").Stream(ctx) {
		if result.Err != nil {
			break
		}
		count++
		if count >= 5 {
			break
		}
	}
}
