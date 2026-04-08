package traverse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// ---- B2: Deep Update -------------------------------------------------------

func TestDeepUpdate_PatchWithKey(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body: `{"OrderID":1,"Status":"Confirmed","Items":[{"ID":10,"Qty":5}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	type Item struct {
		ID  int `json:"ID"`
		Qty int `json:"Qty"`
	}
	type Order struct {
		Status string `json:"Status"`
		Items  []Item `json:"Items"`
	}

	resp, err := client.From("Orders").Key(1).UpdateDeep(context.Background(), Order{
		Status: "Confirmed",
		Items:  []Item{{ID: 10, Qty: 5}},
	})
	if err != nil {
		t.Fatalf("UpdateDeep() failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	req := lastRecorded(t, server)
	if req.Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", req.Method)
	}
	if req.Path != "/Orders(1)" {
		t.Errorf("expected path /Orders(1), got %s", req.Path)
	}
}

func TestDeepUpdate_PreferHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Orders").Key(1).UpdateDeepWithPrefer(
		context.Background(),
		map[string]any{"Status": "Done"},
		"return=representation",
	)
	if err != nil {
		t.Fatalf("UpdateDeepWithPrefer() failed: %v", err)
	}

	req := lastRecorded(t, server)
	prefer := req.Headers.Get("Prefer")
	if prefer != "return=representation" {
		t.Errorf("expected Prefer: return=representation, got %q", prefer)
	}
}

func TestDeepUpdate_NoKeyUsesEntitySetURL(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Products").UpdateDeep(context.Background(), map[string]any{"Status": "Active"})
	if err != nil {
		t.Fatalf("UpdateDeep() without key failed: %v", err)
	}

	req := lastRecorded(t, server)
	if req.Path != "/Products" {
		t.Errorf("expected path /Products, got %s", req.Path)
	}
}

func TestDeepUpdate_StringKey(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204, Body: ``})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Categories").Key("Electronics").UpdateDeep(
		context.Background(),
		map[string]any{"Description": "All electronics"},
	)
	if err != nil {
		t.Fatalf("UpdateDeep() with string key failed: %v", err)
	}

	req := lastRecorded(t, server)
	if !strings.Contains(req.Path, "Categories") {
		t.Errorf("expected path to contain Categories, got %s", req.Path)
	}
}

func TestDeepUpdateOptions_PreferHeader(t *testing.T) {
	tests := []struct {
		name   string
		opts   DeepUpdateOptions
		expect string
	}{
		{
			name:   "return representation only",
			opts:   DeepUpdateOptions{ReturnRepresentation: true},
			expect: "return=representation",
		},
		{
			name:   "continue on error only",
			opts:   DeepUpdateOptions{ContinueOnError: true},
			expect: "odata.continue-on-error",
		},
		{
			name:   "both flags",
			opts:   DeepUpdateOptions{ReturnRepresentation: true, ContinueOnError: true},
			expect: "return=representation; odata.continue-on-error",
		},
		{
			name:   "no flags",
			opts:   DeepUpdateOptions{},
			expect: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.opts.PreferHeader()
			if got != tc.expect {
				t.Errorf("expected %q, got %q", tc.expect, got)
			}
		})
	}
}

func TestDeepUpdate_ContentTypeHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Orders").Key(42).UpdateDeep(context.Background(), map[string]any{"Note": "test"})
	if err != nil {
		t.Fatalf("UpdateDeep() failed: %v", err)
	}

	req := lastRecorded(t, server)
	ct := req.Headers.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type to contain application/json, got %q", ct)
	}
}

func TestDeepUpdate_BodyIsJSON(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	payload := map[string]any{
		"Status": "Confirmed",
		"Items":  []map[string]any{{"ID": 10, "Qty": 5}},
	}
	_, err = client.From("Orders").Key(1).UpdateDeep(context.Background(), payload)
	if err != nil {
		t.Fatalf("UpdateDeep() failed: %v", err)
	}

	req := lastRecorded(t, server)
	var decoded map[string]any
	if err := json.Unmarshal(req.Body, &decoded); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if decoded["Status"] != "Confirmed" {
		t.Errorf("expected Status=Confirmed in body, got %v", decoded["Status"])
	}
}

// ---- E3: SAP sap:* metadata attributes -------------------------------------

func TestSAPAnnotations_PropertyLevel(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"
           xmlns:sap="http://www.sap.com/Protocols/SAPData">
  <edmx:DataServices m:DataServiceVersion="2.0"
                     xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="TEST" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Material">
        <Key><PropertyRef Name="MaterialID"/></Key>
        <Property Name="MaterialID" Type="Edm.String"
          sap:label="Material" sap:key="true"/>
        <Property Name="Price" Type="Edm.Decimal"
          sap:label="Price" sap:filterable="true" sap:sortable="true"
          sap:unit="Currency" sap:required-in-filter="false"/>
        <Property Name="Description" Type="Edm.String"
          sap:label="Description" sap:searchable="true" sap:text="DescriptionText"
          sap:value-list="standard" sap:display-format="UpperCase"
          sap:semantics="email" sap:field-control="Mandatory"
          sap:updatable-path="IsUpdatable"/>
        <Property Name="Currency" Type="Edm.String" sap:label="Currency"/>
      </EntityType>
      <EntityContainer Name="TEST_SRV" m:IsDefaultEntityContainer="true">
        <EntitySet Name="MaterialSet" EntityType="TEST.Material"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta, err := ParseEDMX(strings.NewReader(edmx))
	if err != nil {
		t.Fatalf("ParseEDMX() failed: %v", err)
	}

	var matType *EntityType
	for i, et := range meta.EntityTypes {
		if et.Name == "Material" {
			matType = &meta.EntityTypes[i]
			break
		}
	}
	if matType == nil {
		t.Fatal("EntityType Material not found")
	}

	propMap := make(map[string]Property)
	for _, p := range matType.Properties {
		propMap[p.Name] = p
	}

	t.Run("sap:key", func(t *testing.T) {
		if !propMap["MaterialID"].SAP.IsKey {
			t.Error("expected MaterialID sap:key=true")
		}
	})

	t.Run("sap:label", func(t *testing.T) {
		if propMap["Price"].SAP.Label != "Price" {
			t.Errorf("expected Label=Price, got %q", propMap["Price"].SAP.Label)
		}
	})

	t.Run("sap:filterable", func(t *testing.T) {
		if !propMap["Price"].SAP.Filterable {
			t.Error("expected Price sap:filterable=true")
		}
	})

	t.Run("sap:sortable", func(t *testing.T) {
		if !propMap["Price"].SAP.Sortable {
			t.Error("expected Price sap:sortable=true")
		}
	})

	t.Run("sap:unit", func(t *testing.T) {
		if propMap["Price"].SAP.Unit != "Currency" {
			t.Errorf("expected Unit=Currency, got %q", propMap["Price"].SAP.Unit)
		}
	})

	t.Run("sap:required-in-filter", func(t *testing.T) {
		if propMap["Price"].SAP.Required {
			t.Error("expected Price sap:required-in-filter=false")
		}
	})

	t.Run("sap:searchable", func(t *testing.T) {
		if !propMap["Description"].SAP.Searchable {
			t.Error("expected Description sap:searchable=true")
		}
	})

	t.Run("sap:text", func(t *testing.T) {
		if propMap["Description"].SAP.Text != "DescriptionText" {
			t.Errorf("expected Text=DescriptionText, got %q", propMap["Description"].SAP.Text)
		}
	})

	t.Run("sap:value-list", func(t *testing.T) {
		if propMap["Description"].SAP.ValueList != "standard" {
			t.Errorf("expected ValueList=standard, got %q", propMap["Description"].SAP.ValueList)
		}
	})

	t.Run("sap:display-format", func(t *testing.T) {
		if propMap["Description"].SAP.DisplayFormat != "UpperCase" {
			t.Errorf("expected DisplayFormat=UpperCase, got %q", propMap["Description"].SAP.DisplayFormat)
		}
	})

	t.Run("sap:semantics", func(t *testing.T) {
		if propMap["Description"].SAP.Semantics != "email" {
			t.Errorf("expected Semantics=email, got %q", propMap["Description"].SAP.Semantics)
		}
	})

	t.Run("sap:field-control", func(t *testing.T) {
		if propMap["Description"].SAP.FieldControl != "Mandatory" {
			t.Errorf("expected FieldControl=Mandatory, got %q", propMap["Description"].SAP.FieldControl)
		}
	})

	t.Run("sap:updatable-path", func(t *testing.T) {
		if propMap["Description"].SAP.UpdatablePath != "IsUpdatable" {
			t.Errorf("expected UpdatablePath=IsUpdatable, got %q", propMap["Description"].SAP.UpdatablePath)
		}
	})
}

func TestSAPAnnotations_EntitySetLevel(t *testing.T) {
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"
           xmlns:sap="http://www.sap.com/Protocols/SAPData">
  <edmx:DataServices m:DataServiceVersion="2.0"
                     xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="TEST" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Material">
        <Key><PropertyRef Name="MaterialID"/></Key>
        <Property Name="MaterialID" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="TEST_SRV" m:IsDefaultEntityContainer="true">
        <EntitySet Name="MaterialSet" EntityType="TEST.Material"
          sap:label="Materials"
          sap:creatable="false"
          sap:updatable="true"
          sap:deletable="false"
          sap:pageable="true"
          sap:addressable="true"
          sap:requires-filter="true"
          sap:change-tracking="true"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta, err := ParseEDMX(strings.NewReader(edmx))
	if err != nil {
		t.Fatalf("ParseEDMX() failed: %v", err)
	}

	var esInfo *EntitySetInfo
	for i, es := range meta.EntitySets {
		if es.Name == "MaterialSet" {
			esInfo = &meta.EntitySets[i]
			break
		}
	}
	if esInfo == nil {
		t.Fatal("EntitySet MaterialSet not found")
	}

	sap := esInfo.SAP

	if sap.Label != "Materials" {
		t.Errorf("expected Label=Materials, got %q", sap.Label)
	}
	if sap.Creatable {
		t.Error("expected Creatable=false")
	}
	if !sap.Updatable {
		t.Error("expected Updatable=true")
	}
	if sap.Deletable {
		t.Error("expected Deletable=false")
	}
	if !sap.Pageable {
		t.Error("expected Pageable=true")
	}
	if !sap.Addressable {
		t.Error("expected Addressable=true")
	}
	if !sap.RequiresFilter {
		t.Error("expected RequiresFilter=true")
	}
	if !sap.ChangeTracking {
		t.Error("expected ChangeTracking=true")
	}
}

func TestSAPAnnotations_EntitySetDefaults(t *testing.T) {
	// Entity set with no SAP attributes should use defaults
	edmx := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0"
                     xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="TEST" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32"/>
      </EntityType>
      <EntityContainer Name="TEST_SRV" m:IsDefaultEntityContainer="true">
        <EntitySet Name="Products" EntityType="TEST.Product"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta, err := ParseEDMX(strings.NewReader(edmx))
	if err != nil {
		t.Fatalf("ParseEDMX() failed: %v", err)
	}

	var esInfo *EntitySetInfo
	for i, es := range meta.EntitySets {
		if es.Name == "Products" {
			esInfo = &meta.EntitySets[i]
			break
		}
	}
	if esInfo == nil {
		t.Fatal("EntitySet Products not found")
	}

	// When no sap:* attributes present, SAP field should be zero value
	// (booleans default to false in Go, which is the zero value)
	// The caller should use derefBoolStr defaults, but core struct stores zero.
	// Verify the SAP label and other fields are empty/zero (not panicking).
	if esInfo.SAP.Label != "" {
		t.Errorf("expected empty Label for entity set with no sap:label, got %q", esInfo.SAP.Label)
	}
}
