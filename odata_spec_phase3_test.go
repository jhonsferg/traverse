package traverse_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonsferg/traverse"
)

// ---------------------------------------------------------------------------
// D1 - Measures Vocabulary
// ---------------------------------------------------------------------------

func TestParseMeasuresVocabulary_ISOCurrency(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Measures.V1.ISOCurrency": "CurrencyCode",
	}
	v := traverse.ParseMeasuresVocabulary(annotations)
	if v.ISOCurrency != "CurrencyCode" {
		t.Errorf("ISOCurrency: got %q, want %q", v.ISOCurrency, "CurrencyCode")
	}
}

func TestParseMeasuresVocabulary_Scale(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Measures.V1.Scale": "2",
	}
	v := traverse.ParseMeasuresVocabulary(annotations)
	if v.Scale == nil || *v.Scale != 2 {
		t.Errorf("Scale: got %v, want 2", v.Scale)
	}
}

func TestParseMeasuresVocabulary_Unit(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Measures.V1.Unit": "kg",
	}
	v := traverse.ParseMeasuresVocabulary(annotations)
	if v.Unit != "kg" {
		t.Errorf("Unit: got %q, want %q", v.Unit, "kg")
	}
}

func TestParseMeasuresVocabulary_SIPrefix(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Measures.V1.SIPrefix": "Kilo",
	}
	v := traverse.ParseMeasuresVocabulary(annotations)
	if v.SIPrefix != "Kilo" {
		t.Errorf("SIPrefix: got %q, want %q", v.SIPrefix, "Kilo")
	}
}

func TestParseMeasuresVocabulary_DurationGranularity(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Measures.V1.DurationGranularity": "days",
	}
	v := traverse.ParseMeasuresVocabulary(annotations)
	if v.DurationGranularity != "days" {
		t.Errorf("DurationGranularity: got %q, want %q", v.DurationGranularity, "days")
	}
}

func TestParseMeasuresVocabulary_AllFields(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Measures.V1.ISOCurrency":         "CurrencyCode",
		"Org.OData.Measures.V1.Scale":               "4",
		"Org.OData.Measures.V1.Unit":                "m",
		"Org.OData.Measures.V1.SIPrefix":            "Mega",
		"Org.OData.Measures.V1.DurationGranularity": "hours",
		"Unrelated.Term":                            "ignored",
	}
	v := traverse.ParseMeasuresVocabulary(annotations)
	if v.ISOCurrency != "CurrencyCode" {
		t.Errorf("ISOCurrency: got %q", v.ISOCurrency)
	}
	if v.Scale == nil || *v.Scale != 4 {
		t.Errorf("Scale: got %v", v.Scale)
	}
	if v.Unit != "m" {
		t.Errorf("Unit: got %q", v.Unit)
	}
	if v.SIPrefix != "Mega" {
		t.Errorf("SIPrefix: got %q", v.SIPrefix)
	}
	if v.DurationGranularity != "hours" {
		t.Errorf("DurationGranularity: got %q", v.DurationGranularity)
	}
}

func TestParseMeasuresVocabulary_Empty(t *testing.T) {
	v := traverse.ParseMeasuresVocabulary(map[string]string{})
	if v.ISOCurrency != "" || v.Unit != "" || v.Scale != nil {
		t.Error("expected zero value for empty annotations")
	}
}

// ---------------------------------------------------------------------------
// D2 - Authorization Vocabulary
// ---------------------------------------------------------------------------

func TestParseAuthorizationVocabulary_Authorizations(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Authorization.V1.Authorizations": "OAuth2Implicit,ApiKey",
	}
	v := traverse.ParseAuthorizationVocabulary(annotations)
	if len(v.Authorizations) != 2 || v.Authorizations[0] != "OAuth2Implicit" || v.Authorizations[1] != "ApiKey" {
		t.Errorf("Authorizations: got %v", v.Authorizations)
	}
}

func TestParseAuthorizationVocabulary_RequiredScopes(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Authorization.V1.RequiredScopes": "read:data,write:data",
	}
	v := traverse.ParseAuthorizationVocabulary(annotations)
	if len(v.RequiredScopes) != 2 || v.RequiredScopes[0] != "read:data" {
		t.Errorf("RequiredScopes: got %v", v.RequiredScopes)
	}
}

func TestParseAuthorizationVocabulary_SecurityScheme(t *testing.T) {
	annotations := map[string]string{ //nolint:gosec // test data - not real credentials
		"Org.OData.Authorization.V1.SecuritySchemeType": "OAuth2",
		"Org.OData.Authorization.V1.AuthorizationURL":   "https://auth.example.com/oauth2/authorize",
		"Org.OData.Authorization.V1.TokenURL":           "https://auth.example.com/oauth2/token",
		"Org.OData.Authorization.V1.Scheme":             "bearer",
		"Org.OData.Authorization.V1.BearerFormat":       "JWT",
	}
	v := traverse.ParseAuthorizationVocabulary(annotations)
	if v.SecuritySchemeType != "OAuth2" {
		t.Errorf("SecuritySchemeType: got %q", v.SecuritySchemeType)
	}
	if v.AuthorizationURL != "https://auth.example.com/oauth2/authorize" {
		t.Errorf("AuthorizationURL: got %q", v.AuthorizationURL)
	}
	if v.TokenURL != "https://auth.example.com/oauth2/token" {
		t.Errorf("TokenURL: got %q", v.TokenURL)
	}
	if v.Scheme != "bearer" {
		t.Errorf("Scheme: got %q", v.Scheme)
	}
	if v.BearerFormat != "JWT" {
		t.Errorf("BearerFormat: got %q", v.BearerFormat)
	}
}

func TestParseAuthorizationVocabulary_ApiKey(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Authorization.V1.SecuritySchemeType": "ApiKey",
		"Org.OData.Authorization.V1.KeyName":            "X-API-Key",
		"Org.OData.Authorization.V1.Location":           "header",
	}
	v := traverse.ParseAuthorizationVocabulary(annotations)
	if v.KeyName != "X-API-Key" {
		t.Errorf("KeyName: got %q", v.KeyName)
	}
	if v.KeyLocation != "header" {
		t.Errorf("KeyLocation: got %q", v.KeyLocation)
	}
}

func TestParseAuthorizationVocabulary_OpenIDConnect(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Authorization.V1.OpenIDConnectUrl": "https://auth.example.com/.well-known/openid-configuration",
	}
	v := traverse.ParseAuthorizationVocabulary(annotations)
	if v.OpenIDConnectURL != "https://auth.example.com/.well-known/openid-configuration" {
		t.Errorf("OpenIDConnectURL: got %q", v.OpenIDConnectURL)
	}
}

func TestParseAuthorizationVocabulary_Empty(t *testing.T) {
	v := traverse.ParseAuthorizationVocabulary(map[string]string{})
	if len(v.Authorizations) != 0 || v.Scheme != "" {
		t.Error("expected zero value for empty annotations")
	}
}

// ---------------------------------------------------------------------------
// D3 - Analytics Vocabulary
// ---------------------------------------------------------------------------

func TestParseAnalyticsVocabulary_AggregationMethod_OASIS(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Aggregation.V1.default": "sum",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if v.AggregationMethod != "sum" {
		t.Errorf("AggregationMethod: got %q, want %q", v.AggregationMethod, "sum")
	}
}

func TestParseAnalyticsVocabulary_Dimension_SAP(t *testing.T) {
	annotations := map[string]string{
		"com.sap.vocabularies.Analytics.v1.Dimension": "true",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if !v.IsDimension {
		t.Error("expected IsDimension=true")
	}
}

func TestParseAnalyticsVocabulary_Measure_SAP(t *testing.T) {
	annotations := map[string]string{
		"com.sap.vocabularies.Analytics.v1.Measure": "true",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if !v.IsMeasure {
		t.Error("expected IsMeasure=true")
	}
}

func TestParseAnalyticsVocabulary_Dimensionality_OASIS(t *testing.T) {
	tests := []struct {
		val         string
		isDimension bool
		isMeasure   bool
	}{
		{"Dimension", true, false},
		{"Measure", false, true},
	}
	for _, tt := range tests {
		annotations := map[string]string{
			"Org.OData.Aggregation.V1.Dimensionality": tt.val,
		}
		v := traverse.ParseAnalyticsVocabulary(annotations)
		if v.IsDimension != tt.isDimension {
			t.Errorf("Dimensionality=%q: IsDimension got %v, want %v", tt.val, v.IsDimension, tt.isDimension)
		}
		if v.IsMeasure != tt.isMeasure {
			t.Errorf("Dimensionality=%q: IsMeasure got %v, want %v", tt.val, v.IsMeasure, tt.isMeasure)
		}
	}
}

func TestParseAnalyticsVocabulary_RollupLevels(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Aggregation.V1.RollupLevels": "3",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if v.RollupLevels != 3 {
		t.Errorf("RollupLevels: got %d, want 3", v.RollupLevels)
	}
}

func TestParseAnalyticsVocabulary_GroupableProperties(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Aggregation.V1.GroupableProperties": "Category,Region",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if len(v.GroupableProperties) != 2 || v.GroupableProperties[0] != "Category" {
		t.Errorf("GroupableProperties: got %v", v.GroupableProperties)
	}
}

func TestParseAnalyticsVocabulary_AggregatableProperties(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Aggregation.V1.AggregatableProperties": "Amount,Quantity",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if len(v.AggregatableProperties) != 2 || v.AggregatableProperties[1] != "Quantity" {
		t.Errorf("AggregatableProperties: got %v", v.AggregatableProperties)
	}
}

func TestParseAnalyticsVocabulary_SAPAggregationMethod(t *testing.T) {
	annotations := map[string]string{
		"com.sap.vocabularies.Analytics.v1.AggregationMethod": "average",
	}
	v := traverse.ParseAnalyticsVocabulary(annotations)
	if v.AggregationMethod != "average" {
		t.Errorf("AggregationMethod (SAP): got %q, want %q", v.AggregationMethod, "average")
	}
}

// ---------------------------------------------------------------------------
// E1 - Atom/XML response body parsing
// ---------------------------------------------------------------------------

const atomFeedSimple = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <title type="text">Orders</title>
  <entry>
    <id>Orders(1)</id>
    <content type="application/xml">
      <m:properties>
        <d:OrderID m:type="Edm.Int32">1</d:OrderID>
        <d:CustomerName>Alice</d:CustomerName>
        <d:Amount m:type="Edm.Decimal">100.50</d:Amount>
      </m:properties>
    </content>
  </entry>
  <entry>
    <id>Orders(2)</id>
    <content type="application/xml">
      <m:properties>
        <d:OrderID m:type="Edm.Int32">2</d:OrderID>
        <d:CustomerName>Bob</d:CustomerName>
        <d:Amount m:type="Edm.Decimal">250.00</d:Amount>
      </m:properties>
    </content>
  </entry>
</feed>`

const atomFeedWithCount = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <m:count>42</m:count>
  <entry>
    <id>Products(10)</id>
    <content type="application/xml">
      <m:properties>
        <d:ProductID m:type="Edm.Int32">10</d:ProductID>
        <d:Name>Widget</d:Name>
      </m:properties>
    </content>
  </entry>
</feed>`

const atomFeedWithNextLink = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <link rel="next" href="https://example.com/odata/Orders?$skip=10&amp;$top=10"/>
  <entry>
    <id>Orders(1)</id>
    <content type="application/xml">
      <m:properties>
        <d:OrderID m:type="Edm.Int32">1</d:OrderID>
      </m:properties>
    </content>
  </entry>
</feed>`

const atomFeedWithNullField = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <id>Customers(1)</id>
    <content type="application/xml">
      <m:properties>
        <d:CustomerID m:type="Edm.Int32">1</d:CustomerID>
        <d:MiddleName m:null="true"/>
        <d:Name>Carol</d:Name>
      </m:properties>
    </content>
  </entry>
</feed>`

func TestParseAtomFeed_TwoEntries(t *testing.T) {
	page := &traverse.Page{}
	err := traverse.ParseAtomFeed(bytes.NewReader([]byte(atomFeedSimple)), page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Value) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(page.Value))
	}
	if page.Value[0]["CustomerName"] != "Alice" {
		t.Errorf("first entry CustomerName: got %v, want Alice", page.Value[0]["CustomerName"])
	}
	if page.Value[1]["CustomerName"] != "Bob" {
		t.Errorf("second entry CustomerName: got %v, want Bob", page.Value[1]["CustomerName"])
	}
}

func TestParseAtomFeed_Count(t *testing.T) {
	page := &traverse.Page{}
	err := traverse.ParseAtomFeed(bytes.NewReader([]byte(atomFeedWithCount)), page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Count == nil || *page.Count != 42 {
		t.Errorf("Count: got %v, want 42", page.Count)
	}
}

func TestParseAtomFeed_NextLink(t *testing.T) {
	page := &traverse.Page{}
	err := traverse.ParseAtomFeed(bytes.NewReader([]byte(atomFeedWithNextLink)), page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.NextLink == "" {
		t.Error("expected NextLink to be set")
	}
	if page.NextLink != "https://example.com/odata/Orders?$skip=10&$top=10" {
		t.Errorf("NextLink: got %q", page.NextLink)
	}
}

func TestParseAtomFeed_NullField(t *testing.T) {
	page := &traverse.Page{}
	err := traverse.ParseAtomFeed(bytes.NewReader([]byte(atomFeedWithNullField)), page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Value) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(page.Value))
	}
	entry := page.Value[0]
	middleName, hasMiddleName := entry["MiddleName"]
	if !hasMiddleName {
		t.Error("expected MiddleName key to be present (even if nil)")
	}
	if middleName != nil {
		t.Errorf("MiddleName: expected nil, got %v", middleName)
	}
	if entry["Name"] != "Carol" {
		t.Errorf("Name: got %v, want Carol", entry["Name"])
	}
}

func TestParseAtomFeed_EmptyFeed(t *testing.T) {
	const emptyFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
</feed>`
	page := &traverse.Page{}
	err := traverse.ParseAtomFeed(bytes.NewReader([]byte(emptyFeed)), page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Value) != 0 {
		t.Errorf("expected 0 entries, got %d", len(page.Value))
	}
}

// ---------------------------------------------------------------------------
// E1 - Atom/XML integration via HTTP (Content-Type routing)
// ---------------------------------------------------------------------------

func TestAtomContentTypeRouting_Stream(t *testing.T) {
	const atomBody = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <content type="application/xml">
      <m:properties>
        <d:ID m:type="Edm.Int32">99</d:ID>
        <d:Label>TestItem</d:Label>
      </m:properties>
    </content>
  </entry>
</feed>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		io.WriteString(w, atomBody) //nolint:errcheck
	}))
	defer srv.Close()

	client, err := traverse.New(
		traverse.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("traverse.New: %v", err)
	}

	page, err := client.From("Items").Page(t.Context())
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Value) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(page.Value))
	}
	if page.Value[0]["Label"] != "TestItem" {
		t.Errorf("Label: got %v, want TestItem", page.Value[0]["Label"])
	}
}

// ---------------------------------------------------------------------------
// E1 - isAtomContentType
// ---------------------------------------------------------------------------

func TestIsAtomContentType(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/atom+xml; charset=utf-8", true},
		{"application/xml", true},
		{"text/xml", true},
		{"application/json", false},
		{"application/json;odata.metadata=minimal", false},
		{"", false},
	}
	for _, tt := range tests {
		got := traverse.IsAtomContentType(tt.ct)
		if got != tt.want {
			t.Errorf("isAtomContentType(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}
