package traverse

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeXMLResponse_V2Atom(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"
       xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
       xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <content type="application/xml">
    <m:properties>
      <d:Name>Widget</d:Name>
      <d:Price>9.99</d:Price>
      <d:ID>123</d:ID>
    </m:properties>
  </content>
</entry>`

	result, err := decodeXMLResponse([]byte(xmlBody), ODataV2)
	if err != nil {
		t.Fatalf("decodeXMLResponse error: %v", err)
	}
	if result["Name"] != "Widget" {
		t.Errorf("Name = %q, want Widget", result["Name"])
	}
	if result["Price"] != "9.99" {
		t.Errorf("Price = %q, want 9.99", result["Price"])
	}
	if result["ID"] != "123" {
		t.Errorf("ID = %q, want 123", result["ID"])
	}
}

func TestDecodeXMLResponse_V4Properties(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"
       xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
       xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <m:properties>
    <d:Name>Widget</d:Name>
    <d:Price>9.99</d:Price>
  </m:properties>
</entry>`

	result, err := decodeXMLResponse([]byte(xmlBody), ODataV4)
	if err != nil {
		t.Fatalf("decodeXMLResponse error: %v", err)
	}
	if result["Name"] != "Widget" {
		t.Errorf("Name = %q, want Widget", result["Name"])
	}
}

func TestDecodeXMLResponse_Invalid(t *testing.T) {
	_, err := decodeXMLResponse([]byte("not xml at all <>"), ODataV2)
	if err == nil {
		t.Error("expected error for invalid XML content")
	}
}

func TestDecodeXMLResponse_ActuallyInvalid(t *testing.T) {
	_, err := decodeXMLResponse([]byte("<unclosed"), ODataV2)
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}

func TestDecodeXMLResponse_EmptyEntry(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"></entry>`
	result, err := decodeXMLResponse([]byte(xmlBody), ODataV2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestCreateWithRawXML(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"
       xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
       xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <content type="application/xml">
    <m:properties>
      <d:ID>42</d:ID>
      <d:Name>New Item</d:Name>
    </m:properties>
  </content>
</entry>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Accept") != "application/atom+xml" {
			t.Errorf("Accept = %s, want application/atom+xml", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(xmlResponse))
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	body, err := client.createWithRawXML(context.Background(), "Items", map[string]string{"Name": "New Item"})
	if err != nil {
		t.Fatalf("createWithRawXML error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}

	var entry struct {
		XMLName xml.Name `xml:"entry"`
	}
	if err := xml.Unmarshal(body, &entry); err != nil {
		t.Errorf("failed to parse response as XML: %v", err)
	}
}

func TestCreateWithRawXML_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"400","message":{"value":"Bad request"}}}`))
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	_, err = client.createWithRawXML(context.Background(), "Items", map[string]string{})
	if err == nil {
		t.Error("expected error for 400 status")
	}
}

func TestXmlBytesToStruct(t *testing.T) {
	type Product struct {
		XMLName xml.Name `xml:"Product"`
		ID      int      `xml:"ID"`
		Name    string   `xml:"Name"`
		Price   float64  `xml:"Price"`
	}

	xmlData := []byte(`<Product><ID>42</ID><Name>Widget</Name><Price>9.99</Price></Product>`)
	product, err := XmlBytesToStruct[Product](xmlData)
	if err != nil {
		t.Fatalf("XmlBytesToStruct error: %v", err)
	}
	if product.ID != 42 {
		t.Errorf("ID = %d, want 42", product.ID)
	}
	if product.Name != "Widget" {
		t.Errorf("Name = %q, want Widget", product.Name)
	}
	if product.Price != 9.99 {
		t.Errorf("Price = %f, want 9.99", product.Price)
	}
}

func TestXmlBytesToStruct_Error(t *testing.T) {
	type Product struct {
		ID int `xml:"ID"`
	}
	_, err := XmlBytesToStruct[Product]([]byte("not xml"))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestRawMessageToXmlStruct(t *testing.T) {
	type Product struct {
		ID   int    `xml:"ID"`
		Name string `xml:"Name"`
	}

	raw := json.RawMessage(`<Product><ID>1</ID><Name>Test</Name></Product>`)
	product, err := rawMessageToXmlStruct[Product](raw)
	if err != nil {
		t.Fatalf("rawMessageToXmlStruct error: %v", err)
	}
	if product.ID != 1 {
		t.Errorf("ID = %d, want 1", product.ID)
	}
	if product.Name != "Test" {
		t.Errorf("Name = %q, want Test", product.Name)
	}
}

func TestRawMessageToXmlStruct_Error(t *testing.T) {
	type Product struct {
		ID int `xml:"ID"`
	}
	_, err := rawMessageToXmlStruct[Product](json.RawMessage(`not xml`))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestCreateXmlAs_Integration(t *testing.T) {
	jsonResponse := `{"ID":42,"Name":"Widget","Price":9.99}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, jsonResponse)
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Product struct {
		ID    int     `json:"ID" xml:"ID"`
		Name  string  `json:"Name" xml:"Name"`
		Price float64 `json:"Price" xml:"Price"`
	}

	product, err := CreateXmlAs[Product](client, context.Background(), "Products", map[string]interface{}{"Name": "Widget"})
	if err != nil {
		t.Fatalf("CreateXmlAs error: %v", err)
	}
	if product.ID != 42 {
		t.Errorf("ID = %d, want 42", product.ID)
	}
}

func TestCollectXmlAs_Integration(t *testing.T) {
	jsonResponse := `{"value":[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, jsonResponse)
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Item struct {
		ID   int    `json:"ID" xml:"ID"`
		Name string `json:"Name" xml:"Name"`
	}

	items, err := CollectXmlAs[Item](client.From("Items"), context.Background())
	if err != nil {
		t.Fatalf("CollectXmlAs error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Name != "A" {
		t.Errorf("items[0].Name = %q, want A", items[0].Name)
	}
}

func TestFirstXmlAs_Integration(t *testing.T) {
	jsonResponse := `{"value":[{"ID":7,"Name":"First"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, jsonResponse)
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Item struct {
		ID   int    `json:"ID" xml:"ID"`
		Name string `json:"Name" xml:"Name"`
	}

	item, err := FirstXmlAs[Item](client.From("Items"), context.Background())
	if err != nil {
		t.Fatalf("FirstXmlAs error: %v", err)
	}
	if item.ID != 7 {
		t.Errorf("ID = %d, want 7", item.ID)
	}
}

func TestFindByKeyXmlAs_Integration(t *testing.T) {
	jsonResponse := `{"ID":5,"Name":"ByKey"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, jsonResponse)
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Item struct {
		ID   int    `json:"ID" xml:"ID"`
		Name string `json:"Name" xml:"Name"`
	}

	item, err := FindByKeyXmlAs[Item](client.From("Items"), context.Background(), 5)
	if err != nil {
		t.Fatalf("FindByKeyXmlAs error: %v", err)
	}
	if item.Name != "ByKey" {
		t.Errorf("Name = %q, want ByKey", item.Name)
	}
}

func TestStreamXmlAs_Integration(t *testing.T) {
	jsonResponse := `{"value":[{"ID":1},{"ID":2},{"ID":3}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Item struct {
		ID int `json:"ID" xml:"ID"`
	}

	results, err := CollectXmlAs[Item](client.From("Items"), context.Background())
	if err != nil {
		t.Fatalf("CollectXmlAs error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}
}

func TestCreateAtomXmlAs_Integration(t *testing.T) {
	xmlResponse := `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"
       xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
       xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <content type="application/xml">
    <m:properties>
      <d:NotificationID>N001</d:NotificationID>
      <d:Description>Test notification</d:Description>
    </m:properties>
  </content>
</entry>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, xmlResponse)
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type SAPNotification struct {
		XMLName xml.Name `xml:"entry"`
		Content struct {
			Properties struct {
				NotificationID string `xml:"NotificationID"`
				Description    string `xml:"Description"`
			} `xml:"http://schemas.microsoft.com/ado/2007/08/dataservices/metadata properties"`
		} `xml:"http://www.w3.org/2005/Atom content"`
	}

	notif, err := CreateAtomXmlAs[SAPNotification](client, context.Background(), "Notifications", map[string]string{})
	if err != nil {
		t.Fatalf("CreateAtomXmlAs error: %v", err)
	}
	if notif.Content.Properties.NotificationID != "N001" {
		t.Errorf("NotificationID = %q, want N001", notif.Content.Properties.NotificationID)
	}
}

func TestCreateAtomXmlAs_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "error")
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Dummy struct {
		XMLName xml.Name `xml:"entry"`
	}
	_, err = CreateAtomXmlAs[Dummy](client, context.Background(), "Items", map[string]string{})
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestXmlBytesToStruct_SAPEntry(t *testing.T) {
	type SAPEntry struct {
		XMLName xml.Name `xml:"entry"`
		Content struct {
			Properties struct {
				Material     string `xml:"Material"`
				MaterialType string `xml:"MaterialType"`
			} `xml:"http://schemas.microsoft.com/ado/2007/08/dataservices/metadata properties"`
		} `xml:"http://www.w3.org/2005/Atom content"`
	}

	xmlData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"
       xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
       xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <content type="application/xml">
    <m:properties>
      <d:Material>MAT001</d:Material>
      <d:MaterialType>FERT</d:MaterialType>
    </m:properties>
  </content>
</entry>`)

	entry, err := XmlBytesToStruct[SAPEntry](xmlData)
	if err != nil {
		t.Fatalf("XmlBytesToStruct error: %v", err)
	}
	if entry.Content.Properties.Material != "MAT001" {
		t.Errorf("Material = %q, want MAT001", entry.Content.Properties.Material)
	}
}

func TestCreateXmlAs_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":"400","message":{"value":"Bad"}}}`)
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	type Dummy struct {
		ID int `json:"ID"`
	}
	_, err = CreateXmlAs[Dummy](client, context.Background(), "Items", map[string]string{})
	if err == nil {
		t.Error("expected error for 400 status")
	}
}

func TestDecodeXMLResponse_MultiNamespace(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom"
       xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
       xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <content type="application/xml">
    <m:properties>
      <d:Category>Electronics</d:Category>
      <d:Weight unit="kg">2.5</d:Weight>
    </m:properties>
  </content>
</entry>`

	result, err := decodeXMLResponse([]byte(xmlBody), ODataV2)
	if err != nil {
		t.Fatalf("decodeXMLResponse error: %v", err)
	}
	if result["Category"] != "Electronics" {
		t.Errorf("Category = %q, want Electronics", result["Category"])
	}
}

func TestCreateWithRawXML_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	client, err := New(WithBaseURL(server.URL), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer client.Close()

	_, err = client.createWithRawXML(context.Background(), "Items", map[string]string{})
	if err == nil {
		t.Error("expected error for network failure")
	}
}
