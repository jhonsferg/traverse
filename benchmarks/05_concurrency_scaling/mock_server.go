package benchmarks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MockODataServer provides a configurable mock OData v4 server for benchmarking
type MockODataServer struct {
	server       *httptest.Server
	URL          string
	config       ServerConfig
	mu           sync.RWMutex
	requestCount int64
	errorRate    float64 // 0.0 to 1.0
}

// ServerConfig configures mock server behavior
type ServerConfig struct {
	// Latency simulates network delay
	Latency time.Duration

	// DefaultPageSize for $top if not specified
	DefaultPageSize int

	// MaxRecords available in the mock dataset
	MaxRecords int

	// ResponseDelay adds artificial delay to simulate processing
	ResponseDelay time.Duration

	// EnableMetadata returns $metadata endpoint
	EnableMetadata bool

	// RecordSize affects JSON response size (number of fields)
	RecordSize RecordSize
}

// RecordSize configures response payload size
type RecordSize string

const (
	RecordSizeSmall  RecordSize = "small"  // ~100 bytes per record
	RecordSizeMedium RecordSize = "medium" // ~500 bytes per record
	RecordSizeLarge  RecordSize = "large"  // ~2KB per record
)

// NewMockODataServer creates a new mock OData server
func NewMockODataServer(config ServerConfig) *MockODataServer {
	if config.DefaultPageSize == 0 {
		config.DefaultPageSize = 50
	}
	if config.MaxRecords == 0 {
		config.MaxRecords = 100000
	}

	ms := &MockODataServer{
		config: config,
	}

	ms.server = httptest.NewServer(http.HandlerFunc(ms.handleRequest))
	ms.URL = ms.server.URL
	return ms
}

// Close shuts down the mock server
func (m *MockODataServer) Close() {
	m.server.Close()
}

// GetRequestCount returns total requests served
func (m *MockODataServer) GetRequestCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCount
}

// SetErrorRate sets error rate (0.0-1.0)
func (m *MockODataServer) SetErrorRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRate = rate
}

func (m *MockODataServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.requestCount++
	m.mu.Unlock()

	// Simulate latency
	if m.config.Latency > 0 {
		time.Sleep(m.config.Latency)
	}

	// Simulate response delay
	if m.config.ResponseDelay > 0 {
		time.Sleep(m.config.ResponseDelay)
	}

	// Check error rate
	if shouldError(m.errorRate) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "500",
				"message": "Server error (simulated)",
			},
		})
		return
	}

	path := r.URL.Path
	query := r.URL.RawQuery

	// Route requests
	if strings.HasSuffix(path, "/$metadata") {
		m.handleMetadata(w, r)
	} else if strings.HasSuffix(path, "/$count") {
		m.handleCount(w, r, query)
	} else if strings.Contains(path, "/Products") {
		m.handleProducts(w, r, query)
	} else if strings.Contains(path, "/Orders") {
		m.handleOrders(w, r, query)
	} else if strings.HasPrefix(path, "/") && strings.Contains(path, "(") {
		// Handle by key: /Products(1)
		m.handleByKey(w, r, path)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "404",
				"message": "Not found",
			},
		})
	}
}

func (m *MockODataServer) handleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprint(w, mockMetadata)
}

func (m *MockODataServer) handleCount(w http.ResponseWriter, r *http.Request, query string) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%d", m.config.MaxRecords)
}

func (m *MockODataServer) handleProducts(w http.ResponseWriter, r *http.Request, query string) {
	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters
	top := m.config.DefaultPageSize
	skip := 0
	filter := ""

	if r.URL.Query().Get("$top") != "" {
		if v, err := strconv.Atoi(r.URL.Query().Get("$top")); err == nil {
			top = v
		}
	}
	if r.URL.Query().Get("$skip") != "" {
		if v, err := strconv.Atoi(r.URL.Query().Get("$skip")); err == nil {
			skip = v
		}
	}
	if r.URL.Query().Get("$filter") != "" {
		filter = r.URL.Query().Get("$filter")
	}

	// Generate records
	records := m.generateRecords(skip, top, filter)

	response := map[string]interface{}{
		"value": records,
	}

	// Add next link if there are more records
	if skip+top < m.config.MaxRecords && r.URL.Query().Get("$count") == "true" {
		response["@odata.count"] = m.config.MaxRecords
	}

	if skip+top < m.config.MaxRecords {
		response["@odata.nextLink"] = fmt.Sprintf("%s/Products?$skip=%d&$top=%d", m.URL, skip+top, top)
	}

	json.NewEncoder(w).Encode(response)
}

func (m *MockODataServer) handleOrders(w http.ResponseWriter, r *http.Request, query string) {
	w.Header().Set("Content-Type", "application/json")

	top := m.config.DefaultPageSize
	skip := 0

	if r.URL.Query().Get("$top") != "" {
		if v, err := strconv.Atoi(r.URL.Query().Get("$top")); err == nil {
			top = v
		}
	}
	if r.URL.Query().Get("$skip") != "" {
		if v, err := strconv.Atoi(r.URL.Query().Get("$skip")); err == nil {
			skip = v
		}
	}

	records := m.generateOrderRecords(skip, top)

	response := map[string]interface{}{
		"value": records,
	}

	json.NewEncoder(w).Encode(response)
}

func (m *MockODataServer) handleByKey(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path like "/Products(1)"
	start := strings.Index(path, "(")
	end := strings.Index(path, ")")
	if start == -1 || end == -1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := path[start+1 : end]

	record := m.generateRecord(id)
	json.NewEncoder(w).Encode(record)
}

func (m *MockODataServer) generateRecords(skip, top int, filter string) []map[string]interface{} {
	records := []map[string]interface{}{}

	maxRecords := skip + top
	if maxRecords > m.config.MaxRecords {
		maxRecords = m.config.MaxRecords
	}

	for i := skip; i < maxRecords; i++ {
		record := m.generateRecord(fmt.Sprintf("%d", i))

		// Apply filter if present
		if filter != "" && !m.matchesFilter(record, filter) {
			continue
		}

		records = append(records, record)
		if len(records) >= top {
			break
		}
	}

	return records
}

func (m *MockODataServer) generateOrderRecords(skip, top int) []map[string]interface{} {
	records := []map[string]interface{}{}

	maxRecords := skip + top
	if maxRecords > m.config.MaxRecords {
		maxRecords = m.config.MaxRecords
	}

	for i := skip; i < maxRecords; i++ {
		record := map[string]interface{}{
			"@odata.type": "#ODataDemo.Order",
			"ID":          i,
			"OrderDate":   time.Now().AddDate(0, 0, -i).Format(time.RFC3339),
			"ShipName":    fmt.Sprintf("Shipper %d", i),
			"ShipAddress": fmt.Sprintf("%d Main St", i),
			"ShipCity":    fmt.Sprintf("City %d", i),
			"Freight":     float64(i) * 10.5,
		}

		records = append(records, record)
	}

	return records
}

func (m *MockODataServer) generateRecord(id string) map[string]interface{} {
	baseRecord := map[string]interface{}{
		"@odata.type":      "#ODataDemo.Product",
		"ID":               id,
		"Name":             fmt.Sprintf("Product %s", id),
		"Description":      fmt.Sprintf("Description for product %s", id),
		"ReleaseDate":      time.Now().AddDate(-1, 0, 0).Format(time.RFC3339),
		"DiscontinuedDate": nil,
		"Rating":           4,
		"Price":            99.99,
	}

	// Add more fields based on RecordSize
	switch m.config.RecordSize {
	case RecordSizeMedium:
		for i := 0; i < 5; i++ {
			baseRecord[fmt.Sprintf("Field%d", i)] = fmt.Sprintf("Value %d for %s", i, id)
		}
	case RecordSizeLarge:
		for i := 0; i < 20; i++ {
			baseRecord[fmt.Sprintf("Field%d", i)] = fmt.Sprintf("Value %d for %s - this is a longer string with more content", i, id)
		}
	}

	return baseRecord
}

func (m *MockODataServer) matchesFilter(record map[string]interface{}, filter string) bool {
	// Simple filter matching for common patterns
	// In real scenarios, would need proper OData filter parsing

	if strings.Contains(filter, "Name eq") {
		// Extract expected value
		parts := strings.Split(filter, "'")
		if len(parts) >= 2 {
			expectedName := parts[1]
			return record["Name"] == expectedName
		}
	}

	if strings.Contains(filter, "Price gt") {
		// Extract price threshold
		parts := strings.Fields(filter)
		if len(parts) >= 3 {
			if threshold, err := strconv.ParseFloat(parts[2], 64); err == nil {
				if price, ok := record["Price"].(float64); ok {
					return price > threshold
				}
			}
		}
	}

	return true
}

func shouldError(errorRate float64) bool {
	if errorRate <= 0 {
		return false
	}
	// Simple pseudo-random: would be better with rand.Float64
	return errorRate > 0.5
}

// Mock OData $metadata response
const mockMetadata = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="ODataDemo" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Description" Type="Edm.String"/>
        <Property Name="ReleaseDate" Type="Edm.DateTimeOffset"/>
        <Property Name="DiscontinuedDate" Type="Edm.DateTimeOffset"/>
        <Property Name="Rating" Type="Edm.Int16"/>
        <Property Name="Price" Type="Edm.Decimal"/>
      </EntityType>
      <EntityType Name="Order">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="OrderDate" Type="Edm.DateTimeOffset"/>
        <Property Name="ShipName" Type="Edm.String"/>
        <Property Name="ShipAddress" Type="Edm.String"/>
        <Property Name="ShipCity" Type="Edm.String"/>
        <Property Name="Freight" Type="Edm.Decimal"/>
      </EntityType>
      <EntityContainer Name="DemoService">
        <EntitySet Name="Products" EntityType="ODataDemo.Product"/>
        <EntitySet Name="Orders" EntityType="ODataDemo.Order"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`
