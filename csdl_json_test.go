package traverse

import (
	"strings"
	"testing"
)

// minimalCSDL is a CSDL JSON document with a single entity type and entity set.
const minimalCSDL = `{
  "$Version": "4.01",
  "$EntityContainer": "TestService.Container",
  "TestService": {
    "$Kind": "Schema",
    "Customer": {
      "$Kind": "EntityType",
      "$Key": ["ID"],
      "ID": { "$Type": "Edm.Int32" },
      "Name": { "$Type": "Edm.String" }
    },
    "Container": {
      "$Kind": "EntityContainer",
      "Customers": {
        "$Type": "TestService.Customer",
        "$Collection": true
      }
    }
  }
}`

func TestParseCSDLJSON_MinimalDocument(t *testing.T) {
	md, err := ParseCSDLJSON([]byte(minimalCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if md.Version != "4.01" {
		t.Errorf("Version: got %q, want %q", md.Version, "4.01")
	}

	if md.Namespace != "TestService" {
		t.Errorf("Namespace: got %q, want %q", md.Namespace, "TestService")
	}

	if len(md.EntityTypes) != 1 {
		t.Fatalf("EntityTypes count: got %d, want 1", len(md.EntityTypes))
	}

	et := md.EntityTypes[0]
	if et.Name != "Customer" {
		t.Errorf("EntityType name: got %q, want %q", et.Name, "Customer")
	}

	if len(et.Key) != 1 || et.Key[0].Name != "ID" {
		t.Errorf("EntityType key: got %v, want [{ID}]", et.Key)
	}

	if len(et.Properties) != 2 {
		t.Errorf("EntityType properties count: got %d, want 2", len(et.Properties))
	}

	if len(md.EntitySets) != 1 {
		t.Fatalf("EntitySets count: got %d, want 1", len(md.EntitySets))
	}

	es := md.EntitySets[0]
	if es.Name != "Customers" {
		t.Errorf("EntitySet name: got %q, want %q", es.Name, "Customers")
	}

	if es.EntityTypeName != "Customer" {
		t.Errorf("EntitySet type: got %q, want %q", es.EntityTypeName, "Customer")
	}
}

// complexTypeCSDL contains a complex type with nested properties.
const complexTypeCSDL = `{
  "$Version": "4.01",
  "MyService": {
    "$Kind": "Schema",
    "Address": {
      "$Kind": "ComplexType",
      "Street": { "$Type": "Edm.String" },
      "City": { "$Type": "Edm.String" },
      "Postcode": { "$Type": "Edm.String", "$Nullable": false }
    }
  }
}`

func TestParseCSDLJSON_ComplexTypes(t *testing.T) {
	md, err := ParseCSDLJSON([]byte(complexTypeCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(md.ComplexTypes) != 1 {
		t.Fatalf("ComplexTypes count: got %d, want 1", len(md.ComplexTypes))
	}

	ct := md.ComplexTypes[0]
	if ct.Name != "Address" {
		t.Errorf("ComplexType name: got %q, want %q", ct.Name, "Address")
	}

	if len(ct.Properties) != 3 {
		t.Errorf("ComplexType properties count: got %d, want 3", len(ct.Properties))
	}

	// Find the Postcode property and verify nullable=false.
	var postcode *Property
	for i := range ct.Properties {
		if ct.Properties[i].Name == "Postcode" {
			postcode = &ct.Properties[i]
			break
		}
	}

	if postcode == nil {
		t.Fatal("Postcode property not found in ComplexType")
	}

	if postcode.Nullable {
		t.Errorf("Postcode.Nullable: got true, want false")
	}
}

// enumTypeCSDL contains an enum type with three members.
const enumTypeCSDL = `{
  "$Version": "4.01",
  "MyService": {
    "$Kind": "Schema",
    "Colour": {
      "$Kind": "EnumType",
      "$UnderlyingType": "Edm.Int32",
      "Red": 0,
      "Green": 1,
      "Blue": 2
    }
  }
}`

func TestParseCSDLJSON_EnumType(t *testing.T) {
	md, err := ParseCSDLJSON([]byte(enumTypeCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(md.EnumTypes) != 1 {
		t.Fatalf("EnumTypes count: got %d, want 1", len(md.EnumTypes))
	}

	en := md.EnumTypes[0]
	if en.Name != "Colour" {
		t.Errorf("EnumType name: got %q, want %q", en.Name, "Colour")
	}

	if en.UnderlyingType != "Edm.Int32" {
		t.Errorf("EnumType UnderlyingType: got %q, want %q", en.UnderlyingType, "Edm.Int32")
	}

	if len(en.Members) != 3 {
		t.Errorf("EnumType members count: got %d, want 3", len(en.Members))
	}

	memberValues := make(map[string]int, len(en.Members))
	for _, m := range en.Members {
		memberValues[m.Name] = m.Value
	}

	for _, wantName := range []string{"Red", "Green", "Blue"} {
		if _, ok := memberValues[wantName]; !ok {
			t.Errorf("member %q not found", wantName)
		}
	}

	if memberValues["Red"] != 0 || memberValues["Green"] != 1 || memberValues["Blue"] != 2 {
		t.Errorf("unexpected member values: %v", memberValues)
	}
}

// multiSchemaCSDL contains two schema namespaces in one document.
const multiSchemaCSDL = `{
  "$Version": "4.01",
  "SchemaA": {
    "$Kind": "Schema",
    "TypeA": {
      "$Kind": "EntityType",
      "$Key": ["ID"],
      "ID": { "$Type": "Edm.Guid" }
    }
  },
  "SchemaB": {
    "$Kind": "Schema",
    "TypeB": {
      "$Kind": "EntityType",
      "$Key": ["Code"],
      "Code": { "$Type": "Edm.String" }
    }
  }
}`

func TestParseCSDLJSON_MultipleSchemas(t *testing.T) {
	md, err := ParseCSDLJSON([]byte(multiSchemaCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(md.EntityTypes) != 2 {
		t.Fatalf("EntityTypes count: got %d, want 2", len(md.EntityTypes))
	}

	names := make(map[string]bool, 2)
	for _, et := range md.EntityTypes {
		names[et.Name] = true
	}

	for _, want := range []string{"TypeA", "TypeB"} {
		if !names[want] {
			t.Errorf("EntityType %q not found", want)
		}
	}
}

// nullableCSDL verifies that $Nullable handling is correct for both values.
const nullableCSDL = `{
  "$Version": "4.01",
  "MyService": {
    "$Kind": "Schema",
    "Product": {
      "$Kind": "EntityType",
      "$Key": ["ID"],
      "ID": { "$Type": "Edm.Int32", "$Nullable": false },
      "Description": { "$Type": "Edm.String", "$Nullable": true },
      "Price": { "$Type": "Edm.Decimal" }
    }
  }
}`

func TestParseCSDLJSON_NullableProperties(t *testing.T) {
	md, err := ParseCSDLJSON([]byte(nullableCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(md.EntityTypes) != 1 {
		t.Fatalf("EntityTypes count: got %d, want 1", len(md.EntityTypes))
	}

	props := make(map[string]Property)
	for _, p := range md.EntityTypes[0].Properties {
		props[p.Name] = p
	}

	if id, ok := props["ID"]; !ok {
		t.Error("property ID not found")
	} else if id.Nullable {
		t.Errorf("ID.Nullable: got true, want false")
	}

	if desc, ok := props["Description"]; !ok {
		t.Error("property Description not found")
	} else if !desc.Nullable {
		t.Errorf("Description.Nullable: got false, want true")
	}

	// Absent $Nullable defaults to true.
	if price, ok := props["Price"]; !ok {
		t.Error("property Price not found")
	} else if !price.Nullable {
		t.Errorf("Price.Nullable: got false, want true (default)")
	}
}

// actionsCSDL contains a bound action with parameters.
const actionsCSDL = `{
  "$Version": "4.01",
  "MyService": {
    "$Kind": "Schema",
    "Approve": {
      "$Kind": "Action",
      "$IsBound": true,
      "$Parameter": [
        { "$Name": "bindingParameter", "$Type": "MyService.Order" },
        { "$Name": "Comment", "$Type": "Edm.String", "$Nullable": true }
      ],
      "$ReturnType": { "$Type": "MyService.Order" }
    }
  }
}`

func TestParseCSDLJSON_Actions(t *testing.T) {
	md, err := ParseCSDLJSON([]byte(actionsCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(md.Actions) != 1 {
		t.Fatalf("Actions count: got %d, want 1", len(md.Actions))
	}

	a := md.Actions[0]
	if a.Name != "Approve" {
		t.Errorf("Action name: got %q, want %q", a.Name, "Approve")
	}

	if len(a.Parameters) != 2 {
		t.Errorf("Action parameters count: got %d, want 2", len(a.Parameters))
	}

	if a.ReturnType != "MyService.Order" {
		t.Errorf("Action ReturnType: got %q, want %q", a.ReturnType, "MyService.Order")
	}
}

func TestParseCSDLJSONReader_Basic(t *testing.T) {
	md, err := ParseCSDLJSONReader(strings.NewReader(minimalCSDL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if md.Version != "4.01" {
		t.Errorf("Version: got %q, want %q", md.Version, "4.01")
	}

	if len(md.EntityTypes) != 1 {
		t.Errorf("EntityTypes count: got %d, want 1", len(md.EntityTypes))
	}
}

func TestParseCSDLJSON_InvalidJSON(t *testing.T) {
	_, err := ParseCSDLJSON([]byte(`not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseCSDLJSON_MissingVersion(t *testing.T) {
	// A document without $Version should still parse successfully with an empty version.
	const noVersion = `{
  "MyService": {
    "$Kind": "Schema",
    "Item": {
      "$Kind": "EntityType",
      "$Key": ["ID"],
      "ID": { "$Type": "Edm.Int32" }
    }
  }
}`

	md, err := ParseCSDLJSON([]byte(noVersion))
	if err != nil {
		t.Fatalf("unexpected error for missing $Version: %v", err)
	}

	if md.Version != "" {
		t.Errorf("Version: got %q, want empty string", md.Version)
	}

	if len(md.EntityTypes) != 1 {
		t.Errorf("EntityTypes count: got %d, want 1", len(md.EntityTypes))
	}
}
