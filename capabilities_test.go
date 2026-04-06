package traverse

import (
	"testing"
)

func TestParseCapabilities_FilterRestrictions(t *testing.T) {
	edmxXML := []byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="TestService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
      </EntityType>
      <EntityContainer Name="TestContainer">
        <EntitySet Name="Products" EntityType="TestService.Product">
          <Annotation Term="Capabilities.FilterRestrictions">
            <Record>
              <PropertyValue Property="Filterable" Bool="false"/>
            </Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	registry, err := ParseCapabilities(edmxXML)
	if err != nil {
		t.Fatalf("ParseCapabilities failed: %v", err)
	}

	cap := registry.Get("Products")
	if cap.Filterable {
		t.Error("Expected Filterable to be false")
	}
}

func TestParseCapabilities_NoAnnotations(t *testing.T) {
	edmxXML := []byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="TestService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
      </EntityType>
      <EntityContainer Name="TestContainer">
        <EntitySet Name="Products" EntityType="TestService.Product"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	registry, err := ParseCapabilities(edmxXML)
	if err != nil {
		t.Fatalf("ParseCapabilities failed: %v", err)
	}

	cap := registry.Get("Products")
	if !cap.Filterable || !cap.Sortable || !cap.Insertable || !cap.Updatable || !cap.Deletable {
		t.Error("Expected all operations to be allowed by default")
	}
}

func TestCapabilityError_Format(t *testing.T) {
	err := &CapabilityError{
		EntitySet: "Products",
		Operation: "filter",
		Message:   "service does not support filtering",
	}

	expected := "traverse: capability error on Products: filter operation not supported: service does not support filtering"
	if err.Error() != expected {
		t.Errorf("Error format mismatch\nGot:      %s\nExpected: %s", err.Error(), expected)
	}
}

func TestCapabilitiesRegistry_Get(t *testing.T) {
	registry := NewCapabilitiesRegistry()

	registry.sets["Products"] = EntityCapabilities{
		Filterable: false,
		Sortable:   true,
	}

	cap := registry.Get("Products")
	if cap.Filterable {
		t.Error("Expected Filterable to be false")
	}

	cap = registry.Get("NonExistent")
	if !cap.Filterable {
		t.Error("Expected default capabilities")
	}
}

func TestWithCapabilitiesValidation_NilRegistry(t *testing.T) {
	cfg := &clientConfig{
		beforeQuery: make([]func(*QueryBuilder) error, 0),
	}

	opt := WithCapabilitiesValidation(nil)
	err := opt(cfg)
	if err != nil {
		t.Fatalf("WithCapabilitiesValidation(nil) failed: %v", err)
	}

	if len(cfg.beforeQuery) != 0 {
		t.Error("Expected no hooks to be added for nil registry")
	}
}
