package traverse

import (
	"bytes"
	"testing"
)

func parseEDMXBytes(t *testing.T, data []byte) *Metadata {
	t.Helper()
	meta, err := ParseEDMX(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseEDMX: %v", err)
	}
	return meta
}

func TestParseAbstractEntityType(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Base" Abstract="true">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
      </EntityType>
      <EntityType Name="Concrete">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta := parseEDMXBytes(t, []byte(xmlDoc))
	if len(meta.EntityTypes) < 2 {
		t.Fatalf("expected 2 entity types, got %d", len(meta.EntityTypes))
	}

	base := meta.EntityTypes[0]
	if base.Name != "Base" {
		t.Fatalf("expected first entity type to be Base, got %s", base.Name)
	}
	if !base.Abstract {
		t.Errorf("expected Base.Abstract=true, got false")
	}

	concrete := meta.EntityTypes[1]
	if concrete.Abstract {
		t.Errorf("expected Concrete.Abstract=false, got true")
	}
}

func TestParseBaseType(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Base" Abstract="true">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32"/>
      </EntityType>
      <EntityType Name="Child" BaseType="Test.Base">
        <Property Name="Extra" Type="Edm.String"/>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta := parseEDMXBytes(t, []byte(xmlDoc))
	if len(meta.EntityTypes) < 2 {
		t.Fatalf("expected 2 entity types, got %d", len(meta.EntityTypes))
	}

	child := meta.EntityTypes[1]
	if child.Name != "Child" {
		t.Fatalf("expected second entity type to be Child, got %s", child.Name)
	}
	if child.BaseType != "Test.Base" {
		t.Errorf("expected Child.BaseType=Test.Base, got %q", child.BaseType)
	}

	base := meta.EntityTypes[0]
	if base.BaseType != "" {
		t.Errorf("expected Base.BaseType to be empty, got %q", base.BaseType)
	}
}

func TestParseNullableProperty(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Order">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Description" Type="Edm.String"/>
        <Property Name="Amount" Type="Edm.Decimal" Nullable="true"/>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta := parseEDMXBytes(t, []byte(xmlDoc))
	if len(meta.EntityTypes) == 0 {
		t.Fatal("expected entity types")
	}
	et := meta.EntityTypes[0]

	byName := map[string]Property{}
	for _, p := range et.Properties {
		byName[p.Name] = p
	}

	if byName["ID"].Nullable {
		t.Errorf("expected ID.Nullable=false, got true")
	}
	if !byName["Description"].Nullable {
		t.Errorf("expected Description.Nullable=true (default), got false")
	}
	if !byName["Amount"].Nullable {
		t.Errorf("expected Amount.Nullable=true, got false")
	}
}

func TestParseEnumTypeMembers(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EnumType Name="Status" UnderlyingType="Edm.Int32">
        <Member Name="Draft" Value="0"/>
        <Member Name="Active" Value="1"/>
        <Member Name="Archived" Value="2"/>
      </EnumType>
      <EnumType Name="Flags" IsFlags="true" UnderlyingType="Edm.Int32">
        <Member Name="Read" Value="1"/>
        <Member Name="Write" Value="2"/>
      </EnumType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta := parseEDMXBytes(t, []byte(xmlDoc))
	if len(meta.EnumTypes) != 2 {
		t.Fatalf("expected 2 enum types, got %d", len(meta.EnumTypes))
	}

	status := meta.EnumTypes[0]
	if status.Name != "Status" {
		t.Fatalf("expected first enum Status, got %s", status.Name)
	}
	if status.UnderlyingType != "Edm.Int32" {
		t.Errorf("expected UnderlyingType=Edm.Int32, got %s", status.UnderlyingType)
	}
	if status.IsFlags {
		t.Errorf("expected Status.IsFlags=false, got true")
	}
	if len(status.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(status.Members))
	}
	if status.Members[0].Name != "Draft" || status.Members[0].Value != 0 {
		t.Errorf("unexpected member[0]: %+v", status.Members[0])
	}
	if status.Members[1].Name != "Active" || status.Members[1].Value != 1 {
		t.Errorf("unexpected member[1]: %+v", status.Members[1])
	}
	if status.Members[2].Name != "Archived" || status.Members[2].Value != 2 {
		t.Errorf("unexpected member[2]: %+v", status.Members[2])
	}

	flags := meta.EnumTypes[1]
	if !flags.IsFlags {
		t.Errorf("expected Flags.IsFlags=true, got false")
	}
	if len(flags.Members) != 2 {
		t.Fatalf("expected 2 flag members, got %d", len(flags.Members))
	}
}

func TestParseComplexTypeProperties(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <ComplexType Name="Address">
        <Property Name="Street" Type="Edm.String" Nullable="false"/>
        <Property Name="City" Type="Edm.String"/>
        <Property Name="PostalCode" Type="Edm.String" MaxLength="10"/>
      </ComplexType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta := parseEDMXBytes(t, []byte(xmlDoc))
	if len(meta.ComplexTypes) != 1 {
		t.Fatalf("expected 1 complex type, got %d", len(meta.ComplexTypes))
	}

	addr := meta.ComplexTypes[0]
	if addr.Name != "Address" {
		t.Fatalf("expected complex type Address, got %s", addr.Name)
	}
	if len(addr.Properties) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(addr.Properties))
	}

	byName := map[string]Property{}
	for _, p := range addr.Properties {
		byName[p.Name] = p
	}

	if byName["Street"].Nullable {
		t.Errorf("expected Street.Nullable=false, got true")
	}
	if !byName["City"].Nullable {
		t.Errorf("expected City.Nullable=true (default), got false")
	}
	if byName["PostalCode"].MaxLength == nil || *byName["PostalCode"].MaxLength != 10 {
		t.Errorf("expected PostalCode.MaxLength=10, got %v", byName["PostalCode"].MaxLength)
	}
}

func TestParseNavigationBinding(t *testing.T) {
	const xmlDoc = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Order">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <NavigationProperty Name="Items" Type="Collection(Test.OrderItem)"/>
      </EntityType>
      <EntityType Name="OrderItem">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
      </EntityType>
      <EntityContainer Name="Default">
        <EntitySet Name="Orders" EntityType="Test.Order">
          <NavigationPropertyBinding Path="Items" Target="OrderItems"/>
        </EntitySet>
        <EntitySet Name="OrderItems" EntityType="Test.OrderItem"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta := parseEDMXBytes(t, []byte(xmlDoc))

	var ordersSet *EntitySetInfo
	for i := range meta.EntitySets {
		if meta.EntitySets[i].Name == "Orders" {
			ordersSet = &meta.EntitySets[i]
			break
		}
	}
	if ordersSet == nil {
		t.Fatal("expected Orders entity set")
	}
	if len(ordersSet.NavigationBindings) != 1 {
		t.Fatalf("expected 1 navigation binding, got %d", len(ordersSet.NavigationBindings))
	}
	nb := ordersSet.NavigationBindings[0]
	if nb.Path != "Items" {
		t.Errorf("expected Path=Items, got %q", nb.Path)
	}
	if nb.Target != "OrderItems" {
		t.Errorf("expected Target=OrderItems, got %q", nb.Target)
	}

	// OrderItems set should have no bindings
	var itemsSet *EntitySetInfo
	for i := range meta.EntitySets {
		if meta.EntitySets[i].Name == "OrderItems" {
			itemsSet = &meta.EntitySets[i]
			break
		}
	}
	if itemsSet == nil {
		t.Fatal("expected OrderItems entity set")
	}
	if len(itemsSet.NavigationBindings) != 0 {
		t.Errorf("expected no navigation bindings on OrderItems, got %d", len(itemsSet.NavigationBindings))
	}
}
