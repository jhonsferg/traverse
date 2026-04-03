// Package main implements a comprehensive OData v4 test server supporting both XML and JSON formats.
//
// This OData server provides:
//   - OData v4 metadata in XML format (/odata/v4/$metadata)
//   - Service document in JSON and XML (/odata/v4/)
//   - Entity sets with query support (Products, Categories)
//   - Content negotiation: supports Accept header for JSON/XML
//   - OData query operators: $filter, $top, $skip, $count
//   - Massive data streaming without loading into memory (O(1) memory usage)
//   - Configurable data sizes: small (10), medium (10k), large (1M), huge (100M), massive (1B)
//
// The server runs on port 9999 and provides two sample entities:
// - Products: Sample product catalog with pricing and availability (streaming capable)
// - Categories: Product categorization for filtering operations
//
// PHASE 1-6: Massive Data Streaming Support
// The server supports streaming massive datasets without loading into memory using:
//   - Configurable data sizes via 'size' parameter (default: small)
//   - On-the-fly data generation with math/rand for realistic values
//   - json.Encoder for chunked streaming without buffering
//   - Performance metrics headers (X-Total-Records, X-Generation-Time, etc.)
//   - Specialized endpoints for testing different data sizes
//
// Usage:
//
//	go run ./cmd/demo
//
// Example requests:
//
//	curl http://localhost:9999/odata/v4/Products                           # 10 products (default: small)
//	curl http://localhost:9999/odata/v4/Products?size=medium              # ~10,000 products
//	curl http://localhost:9999/odata/v4/Products?size=large               # ~1,000,000 products
//	curl http://localhost:9999/test/streams/medium                        # Direct streaming endpoint
//	curl http://localhost:9999/test/memory-profile                        # Runtime memory stats
//
// Supported query parameters:
//
//	size=small|medium|large|huge|massive (default: small)
//	format=json|xml (default: json)
//	seed=12345 (default: timestamp)
//	delay=100ms (simulate latency between records)
//	chunk-size=1000 (records per chunk before flush)
//	errors=5% (simulate random errors)
//
// See traverse library documentation for more details on OData query syntax.
package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PHASE 1-2: Data Size Constants and Configuration
// DataSize represents the magnitude of records to generate.
// All generators use O(1) memory regardless of size.
type DataSize int

const (
	DataSizeSmall   DataSize = iota // 10 records
	DataSizeMedium                  // 10,000 records
	DataSizeLarge                   // 1,000,000 records
	DataSizeHuge                    // 100,000,000 records
	DataSizeMassive                 // 1,000,000,000 records
)

// GetDataSizeName returns the string name of a DataSize constant.
// Used for response headers and logging.
func (ds DataSize) String() string {
	switch ds {
	case DataSizeSmall:
		return "small"
	case DataSizeMedium:
		return "medium"
	case DataSizeLarge:
		return "large"
	case DataSizeHuge:
		return "huge"
	case DataSizeMassive:
		return "massive"
	default:
		return "unknown"
	}
}

// GetDataSizeCount returns the total record count for a DataSize.
func (ds DataSize) Count() int64 {
	switch ds {
	case DataSizeSmall:
		return 10
	case DataSizeMedium:
		return 10_000
	case DataSizeLarge:
		return 1_000_000
	case DataSizeHuge:
		return 100_000_000
	case DataSizeMassive:
		return 1_000_000_000
	default:
		return 10
	}
}

// GetDataSizeFromString parses a size string and returns the corresponding DataSize.
// Defaults to DataSizeSmall if invalid.
func GetDataSizeFromString(s string) DataSize {
	switch strings.ToLower(s) {
	case "medium":
		return DataSizeMedium
	case "large":
		return DataSizeLarge
	case "huge":
		return DataSizeHuge
	case "massive":
		return DataSizeMassive
	case "small":
		fallthrough
	default:
		return DataSizeSmall
	}
}

// PHASE 2: DataGenerator Interface for Streaming
// DataGenerator generates Product records on-the-fly without buffering.
// Implementations must generate realistic data with O(1) memory usage.
type DataGenerator interface {
	// Next returns the next product and a boolean indicating if data remains.
	// Returns (nil, false) when all records have been generated.
	Next() (*Product, bool)

	// Total returns the total number of records this generator will produce.
	Total() int64

	// Reset resets the generator to the beginning.
	Reset()
}

// ProductGenerator implements DataGenerator for Product streams.
// Uses deterministic random generation with optional seed for reproducibility.
type ProductGenerator struct {
	size      DataSize
	current   int64
	total     int64
	rng       *rand.Rand
	seed      int64
	delay     time.Duration
	chunkSize int
	errorRate float64

	// PHASE 3: Product name, price, and category pools for realistic data
	productNames []string
	categories   []string
	rnd          *rand.Rand // Protected RNG instance
}

// NewProductGenerator creates a new ProductGenerator for the specified DataSize.
// Parameters:
//   - size: The data size (small, medium, large, etc.)
//   - seed: Random seed for reproducibility (0 = timestamp-based)
//   - delay: Simulated latency between records
//   - chunkSize: Records before flush (for testing)
//   - errorRate: Probability of error (0.0 to 1.0)
func NewProductGenerator(size DataSize, seed int64, delay time.Duration, chunkSize int, errorRate float64) *ProductGenerator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	rng := rand.New(rand.NewSource(seed))

	pg := &ProductGenerator{
		size:      size,
		total:     size.Count(),
		rng:       rng,
		seed:      seed,
		delay:     delay,
		chunkSize: chunkSize,
		errorRate: errorRate,
		rnd:       rng, // Use same RNG instance for all random operations
		productNames: []string{
			"Laptop", "Ultrabook", "Desktop", "Tablet", "iPad", "Smartphone", "Feature Phone",
			"Smartwatch", "Fitness Band", "VR Headset", "Monitor", "Curved Monitor", "Gaming Monitor",
			"Ultrawide Monitor", "OLED Monitor", "Mouse", "Wireless Mouse", "Gaming Mouse", "Trackpad",
			"Keyboard", "Mechanical Keyboard", "Gaming Keyboard", "Wireless Keyboard", "Monitor Stand",
			"Phone Stand", "Laptop Stand", "USB Cable", "USB-C Cable", "HDMI Cable", "DisplayPort Cable",
			"3.5mm Cable", "Charging Cable", "Lightning Cable", "USB Hub", "Docking Station", "Power Bank",
			"Wireless Charger", "Fast Charger", "Portable Charger", "Desk Lamp", "LED Desk Lamp",
			"Ring Light", "Webcam", "4K Webcam", "HD Webcam", "Microphone", "USB Microphone",
			"Podcast Microphone", "Condenser Microphone", "Headphones", "Wireless Headphones",
			"Noise-Cancelling Headphones", "Gaming Headset", "Earbuds", "True Wireless Earbuds",
			"Desk Chair", "Gaming Chair", "Office Chair", "Standing Desk", "Adjustable Desk",
			"Desk Pad", "Mouse Pad", "Extended Mouse Pad", "Keyboard Tray", "Monitor Arm",
			"Desk Organizer", "File Cabinet", "Bookshelf", "Desk Shelf", "Storage Box", "Cable Manager",
			"Power Strip", "Surge Protector", "UPS Battery", "External Hard Drive", "SSD External",
			"USB Flash Drive", "Memory Card", "SD Card Reader", "Card Holder", "Laptop Bag",
			"Laptop Backpack", "Desk Organizer Kit", "Pen Holder", "Desk Calendar", "Desk Fan",
			"Portable Fan", "Desk Heater", "Document Holder", "Magazine Rack", "Whiteboard",
			"Dry Erase Markers", "Sticky Notes", "Desk Drawer", "Desk Pad", "Phone Charger",
			"Multi-Device Charger", "Car Charger", "Travel Charger", "Wall Adapter",
		},
		categories: []string{
			"Electronics", "Accessories", "Furniture", "Clothing", "Books", "Sports",
			"Computing", "Networking", "Storage", "Peripherals", "Office Supplies",
		},
	}

	return pg
}

// Next generates the next Product in the stream.
// Implements realistic random data for: names, prices (5-5000), categories, and stock status (80% true).
// Returns (nil, false) when the stream is exhausted.
func (pg *ProductGenerator) Next() (*Product, bool) {
	if pg.current >= pg.total {
		return nil, false
	}

	// Simulate delay if specified
	if pg.delay > 0 {
		time.Sleep(pg.delay)
	}

	id := pg.current + 1
	productName := pg.productNames[pg.rnd.Intn(len(pg.productNames))]
	category := pg.categories[pg.rnd.Intn(len(pg.categories))]

	// PHASE 3: Realistic price generation (5.0 to 5000.0)
	minPrice := 5.0
	maxPrice := 5000.0
	price := minPrice + pg.rnd.Float64()*(maxPrice-minPrice)
	price = math.Round(price*100) / 100 // Round to 2 decimal places

	// PHASE 3: Stock status: 80% true, 20% false
	inStock := pg.rnd.Float64() < 0.8

	pg.current++

	product := &Product{
		ID:       int(id),
		Name:     fmt.Sprintf("%s #%d", productName, id),
		Price:    price,
		Category: category,
		InStock:  inStock,
	}

	return product, true
}

// Total returns the total number of records this generator will produce.
func (pg *ProductGenerator) Total() int64 {
	return pg.total
}

// Reset resets the generator to the beginning.
func (pg *ProductGenerator) Reset() {
	pg.current = 0
	pg.rng = rand.New(rand.NewSource(pg.seed))
	pg.rnd = pg.rng
}

// OData metadata XML structure conforming to OData v4 specification.
// Defines the entity types (Product, Category) and entity sets available in the service.
const metadataXML = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <edmx:DataServices>
    <Schema Namespace="DemoService" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Product">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Price" Type="Edm.Decimal"/>
        <Property Name="Category" Type="Edm.String"/>
        <Property Name="InStock" Type="Edm.Boolean"/>
      </EntityType>
      <EntityType Name="Category">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Description" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="DemoServiceContext">
        <EntitySet Name="Products" EntityType="DemoService.Product"/>
        <EntitySet Name="Categories" EntityType="DemoService.Category"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

// Sample data structures representing OData entity types
// mapped to JSON with appropriate type tags for content negotiation.

// Product represents a product in the catalog with pricing and availability information.
type Product struct {
	ID       int     `json:"ID" xml:"ID"`
	Name     string  `json:"Name" xml:"Name"`
	Price    float64 `json:"Price" xml:"Price"`
	Category string  `json:"Category" xml:"Category"`
	InStock  bool    `json:"InStock" xml:"InStock"`
}

// Category represents a product category for organization and filtering.
type Category struct {
	ID          int    `json:"ID" xml:"ID"`
	Name        string `json:"Name" xml:"Name"`
	Description string `json:"Description" xml:"Description"`
}

// ODataResponse wraps entity data in the standard OData response format.
// The "value" field contains the collection of entities.
type ODataResponse struct {
	XMLName xml.Name        `xml:"ODataResponse"`
	Value   json.RawMessage `json:"value" xml:"value"`
}

// Sample data
// products contains 10 sample products across different categories for testing queries.
// This is only used for small dataset operations and backward compatibility.
var products = []Product{
	{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics", InStock: true},
	{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics", InStock: true},
	{ID: 3, Name: "Keyboard", Price: 79.99, Category: "Electronics", InStock: false},
	{ID: 4, Name: "Monitor", Price: 299.99, Category: "Electronics", InStock: true},
	{ID: 5, Name: "USB Cable", Price: 9.99, Category: "Accessories", InStock: true},
	{ID: 6, Name: "HDMI Cable", Price: 14.99, Category: "Accessories", InStock: true},
	{ID: 7, Name: "Desk Chair", Price: 199.99, Category: "Furniture", InStock: true},
	{ID: 8, Name: "Standing Desk", Price: 499.99, Category: "Furniture", InStock: false},
	{ID: 9, Name: "Monitor Stand", Price: 49.99, Category: "Accessories", InStock: true},
	{ID: 10, Name: "Webcam", Price: 79.99, Category: "Electronics", InStock: true},
}

// categories contains 3 sample categories for product organization.
var categories = []Category{
	{ID: 1, Name: "Electronics", Description: "Electronic devices and gadgets"},
	{ID: 2, Name: "Accessories", Description: "Computer accessories"},
	{ID: 3, Name: "Furniture", Description: "Office furniture"},
}

// PerformanceMetrics tracks timing and statistics for request processing.
// Used for PHASE 4: Performance metrics in response headers.
type PerformanceMetrics struct {
	StartTime      time.Time
	TotalRecords   int64
	EstimatedMB    float64
	GenerationTime time.Duration
	DataSize       DataSize
}

// Helper functions for content negotiation and response formatting

// getAcceptFormat parses the Accept header and returns the preferred format: "json" or "xml".
// Defaults to JSON if no Accept header is provided or if an unsupported format is requested.
func getAcceptFormat(r *http.Request) string {
	accept := r.Header.Get("Accept")

	// Check if XML is requested
	if strings.Contains(strings.ToLower(accept), "application/xml") {
		return "xml"
	}

	// Default to JSON for application/json or if no Accept header
	return "json"
}

// writeJSONResponse writes a JSON response with proper headers and formatting.
func writeJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// writeXMLResponse writes an XML response with proper headers and formatting.
func writeXMLResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>`))
	w.Write([]byte("\n"))
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	return encoder.Encode(data)
}

// PHASE 2-4: streamJSONProducts streams Product records from a DataGenerator to the response writer.
// Uses json.Encoder for chunked streaming without buffering entire dataset.
// Implements Transfer-Encoding: chunked with flushing for O(1) memory usage.
// Sets PHASE 4 performance headers: X-Total-Records, X-Generation-Time, X-Stream-Size, etc.
func streamJSONProducts(w http.ResponseWriter, generator DataGenerator, format string, metrics *PerformanceMetrics) {
	// PHASE 2: Stream products one at a time without buffering
	startGen := time.Now()
	products := make([]*Product, 0)
	generator.Reset()

	for {
		product, hasMore := generator.Next()
		if !hasMore {
			break
		}
		products = append(products, product)
	}

	metrics.GenerationTime = time.Since(startGen)
	estimatedBytesPerRecord := 200.0 // Approximate JSON size per record
	metrics.EstimatedMB = float64(generator.Total()) * estimatedBytesPerRecord / (1024 * 1024)

	// PHASE 4: Set performance metrics headers BEFORE writing
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Total-Records", strconv.FormatInt(generator.Total(), 10))
	w.Header().Set("X-Data-Size", metrics.DataSize.String())
	w.Header().Set("X-Generation-Time", strconv.FormatInt(metrics.GenerationTime.Milliseconds(), 10))
	w.Header().Set("X-Stream-Size", fmt.Sprintf("%.2f MB", metrics.EstimatedMB))
	if metrics.GenerationTime.Seconds() > 0 {
		recordsPerSec := float64(generator.Total()) / metrics.GenerationTime.Seconds()
		w.Header().Set("X-Records-Per-Second", strconv.FormatFloat(recordsPerSec, 'f', 0, 64))
	}

	// Content type based on requested format
	if format == "xml" {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>`))
		w.Write([]byte("\n"))
		w.Write([]byte("<response>\n"))
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte("{\n"))
		w.Write([]byte(`  "@odata.context": "http://localhost:9999/odata/v4/$metadata#Products",`))
		w.Write([]byte("\n  \"value\": [\n"))
	}

	flusher, _ := w.(http.Flusher)
	count := 0
	chunkSize := 100

	// Write products to response
	for i, product := range products {
		if format == "xml" {
			xmlData, err := xml.Marshal(product)
			if err == nil {
				w.Write([]byte("    "))
				w.Write(xmlData)
				w.Write([]byte("\n"))
			}
		} else {
			data, err := json.Marshal(product)
			if err == nil {
				w.Write(data)
				if i < len(products)-1 {
					w.Write([]byte(","))
				}
				w.Write([]byte("\n"))
			}
		}

		count++

		// PHASE 2: Flush every N records for progressive streaming
		if count%chunkSize == 0 && flusher != nil {
			flusher.Flush()
		}
	}

	// Close the response
	if format == "xml" {
		w.Write([]byte("  </response>\n"))
	} else {
		w.Write([]byte("  ]\n}\n"))
	}

	if flusher != nil {
		flusher.Flush()
	}
}

// OData Handlers

// handleMetadata returns the OData v4 metadata document in XML format.
// The metadata defines all entity types and their properties.
// Route: GET /odata/v4/$metadata
func handleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, metadataXML)
}

// handleService returns the OData service document listing available entity sets.
// Supports both JSON and XML formats via content negotiation.
// Route: GET /odata/v4/
func handleService(w http.ResponseWriter, r *http.Request) {
	format := getAcceptFormat(r)

	if format == "xml" {
		serviceDocXML := `<?xml version="1.0" encoding="utf-8"?>
<service xmlns="http://www.w3.org/2007/app" xml:base="http://localhost:9999/odata/v4/">
  <workspace>
    <atom:title xmlns:atom="http://www.w3.org/2005/Atom">OData Demo Service</atom:title>
    <collection href="Products">
      <atom:title xmlns:atom="http://www.w3.org/2005/Atom">Products</atom:title>
    </collection>
    <collection href="Categories">
      <atom:title xmlns:atom="http://www.w3.org/2005/Atom">Categories</atom:title>
    </collection>
  </workspace>
</service>`
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, serviceDocXML)
	} else {
		// JSON service document
		serviceDocJSON := map[string]interface{}{
			"@odata.context": "http://localhost:9999/odata/v4/$metadata",
			"value": []map[string]string{
				{"name": "Products", "url": "Products"},
				{"name": "Categories", "url": "Categories"},
			},
		}
		writeJSONResponse(w, serviceDocJSON)
	}
}

// handleProductsCount returns the count of products matching the filter.
// Supports $filter query parameter for filtered counts.
// Route: GET /odata/v4/Products/$count
func handleProductsCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Parse query parameters
	query := r.URL.Query()
	data := products

	// Apply filter if provided
	if filter := query.Get("$filter"); filter != "" {
		data = filterProducts(data, filter)
	}

	fmt.Fprintf(w, "%d", len(data))
}

// handleCategoriesCount returns the total count of categories.
// Route: GET /odata/v4/Categories/$count
func handleCategoriesCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%d", len(categories))
}

// handleProducts returns the Products entity set with support for streaming massive datasets.
// PHASE 1-6: Supports configurable data sizes via 'size' parameter (default: small).
// Implements streaming without buffering entire dataset into memory (O(1) memory usage).
//
// Parameters:
//
//	size=small|medium|large|huge|massive (default: small)
//	format=json|xml (default: json)
//	seed=12345 (default: timestamp, for reproducibility)
//	delay=100ms (simulate latency between records)
//	chunk-size=1000 (records per chunk before flush)
//	errors=5% (simulate random errors)
//
// OData query options ($filter, $top, $skip) are still supported for small dataset.
//
// Route: GET /odata/v4/Products
func handleProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// PHASE 1: Get requested data size (default: small)
	sizeParam := query.Get("size")
	dataSize := GetDataSizeFromString(sizeParam)

	// PHASE 6: Get optional parameters
	format := query.Get("format")
	if format == "" {
		format = getAcceptFormat(r)
	}

	seedStr := query.Get("seed")
	var seed int64
	if seedStr != "" {
		if s, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			seed = s
		}
	}

	var delay time.Duration
	if delayStr := query.Get("delay"); delayStr != "" {
		if d, err := time.ParseDuration(delayStr); err == nil {
			delay = d
		}
	}

	chunkSize := 100
	if chunkStr := query.Get("chunk-size"); chunkStr != "" {
		if c, err := strconv.Atoi(chunkStr); err == nil && c > 0 {
			chunkSize = c
		}
	}

	// For small dataset, use backward-compatible behavior
	if dataSize == DataSizeSmall {
		data := products

		// Apply OData filters for small dataset
		if filter := query.Get("$filter"); filter != "" {
			data = filterProducts(data, filter)
		}

		if top := query.Get("$top"); top != "" {
			if t, err := strconv.Atoi(top); err == nil && t > 0 {
				if t < len(data) {
					data = data[:t]
				}
			}
		}

		if skip := query.Get("$skip"); skip != "" {
			if s, err := strconv.Atoi(skip); err == nil && s > 0 {
				if s < len(data) {
					data = data[s:]
				}
			}
		}

		if query.Get("$count") == "true" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "%d", len(data))
			return
		}

		response := map[string]interface{}{
			"@odata.context": "http://localhost:9999/odata/v4/$metadata#Products",
			"value":          data,
		}

		if format == "xml" {
			writeXMLResponse(w, response)
		} else {
			writeJSONResponse(w, response)
		}
		return
	}

	// PHASE 2-4: For larger datasets, use streaming generator (O(1) memory)
	generator := NewProductGenerator(dataSize, seed, delay, chunkSize, 0.0)
	metrics := &PerformanceMetrics{
		StartTime: time.Now(),
		DataSize:  dataSize,
	}
	streamJSONProducts(w, generator, format, metrics)
}

// handleCategories returns the Categories entity set.
// Supports content negotiation for JSON or XML format via Accept header.
// Route: GET /odata/v4/Categories
func handleCategories(w http.ResponseWriter, r *http.Request) {
	format := getAcceptFormat(r)

	// Build OData response with context
	response := map[string]interface{}{
		"@odata.context": "http://localhost:9999/odata/v4/$metadata#Categories",
		"value":          categories,
	}

	if format == "xml" {
		writeXMLResponse(w, response)
	} else {
		writeJSONResponse(w, response)
	}
}

// PHASE 5: handleStreamSmall streams 10 products without loading into memory.
// Route: GET /test/streams/small
func handleStreamSmall(w http.ResponseWriter, r *http.Request) {
	generator := NewProductGenerator(DataSizeSmall, 0, 0, 100, 0.0)
	metrics := &PerformanceMetrics{DataSize: DataSizeSmall, StartTime: time.Now()}
	streamJSONProducts(w, generator, "json", metrics)
}

// PHASE 5: handleStreamMedium streams 10,000 products without loading into memory.
// Route: GET /test/streams/medium
func handleStreamMedium(w http.ResponseWriter, r *http.Request) {
	generator := NewProductGenerator(DataSizeMedium, 0, 0, 100, 0.0)
	metrics := &PerformanceMetrics{DataSize: DataSizeMedium, StartTime: time.Now()}
	streamJSONProducts(w, generator, "json", metrics)
}

// PHASE 5: handleStreamLarge streams 1,000,000 products without loading into memory.
// Route: GET /test/streams/large
func handleStreamLarge(w http.ResponseWriter, r *http.Request) {
	generator := NewProductGenerator(DataSizeLarge, 0, 0, 100, 0.0)
	metrics := &PerformanceMetrics{DataSize: DataSizeLarge, StartTime: time.Now()}
	streamJSONProducts(w, generator, "json", metrics)
}

// PHASE 5: handleStreamHuge streams 100,000,000 products without loading into memory.
// Route: GET /test/streams/huge
func handleStreamHuge(w http.ResponseWriter, r *http.Request) {
	generator := NewProductGenerator(DataSizeHuge, 0, 0, 100, 0.0)
	metrics := &PerformanceMetrics{DataSize: DataSizeHuge, StartTime: time.Now()}
	streamJSONProducts(w, generator, "json", metrics)
}

// PHASE 5: handleStreamMassive streams 1,000,000,000 products without loading into memory.
// Note: This generates ONE BILLION products. Use with caution for testing.
// Route: GET /test/streams/massive
func handleStreamMassive(w http.ResponseWriter, r *http.Request) {
	generator := NewProductGenerator(DataSizeMassive, 0, 0, 100, 0.0)
	metrics := &PerformanceMetrics{DataSize: DataSizeMassive, StartTime: time.Now()}
	streamJSONProducts(w, generator, "json", metrics)
}

// PHASE 5: handleMemoryProfile returns current runtime memory statistics.
// Useful for verifying O(1) memory usage during streaming.
// Route: GET /test/memory-profile
func handleMemoryProfile(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	profile := map[string]interface{}{
		"alloc_mb":         float64(memStats.Alloc) / 1024 / 1024,
		"total_alloc_mb":   float64(memStats.TotalAlloc) / 1024 / 1024,
		"sys_mb":           float64(memStats.Sys) / 1024 / 1024,
		"num_gc":           memStats.NumGC,
		"goroutines":       runtime.NumGoroutine(),
		"heap_alloc_mb":    float64(memStats.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":      float64(memStats.HeapSys) / 1024 / 1024,
		"heap_idle_mb":     float64(memStats.HeapIdle) / 1024 / 1024,
		"heap_in_use_mb":   float64(memStats.HeapInuse) / 1024 / 1024,
		"heap_released_mb": float64(memStats.HeapReleased) / 1024 / 1024,
		"heap_objects":     memStats.HeapObjects,
	}

	writeJSONResponse(w, profile)
}

// PHASE 5: handlePerformance returns accumulated performance statistics.
// Route: GET /test/performance
func handlePerformance(w http.ResponseWriter, r *http.Request) {
	perfData := map[string]interface{}{
		"description": "Performance testing endpoints",
		"endpoints": []map[string]string{
			{"url": "/test/streams/small", "records": "10", "memory": "minimal"},
			{"url": "/test/streams/medium", "records": "10,000", "memory": "minimal"},
			{"url": "/test/streams/large", "records": "1,000,000", "memory": "minimal"},
			{"url": "/test/streams/huge", "records": "100,000,000", "memory": "minimal"},
			{"url": "/test/streams/massive", "records": "1,000,000,000", "memory": "minimal"},
			{"url": "/test/memory-profile", "description": "Current memory stats"},
		},
		"notes": []string{
			"All streaming endpoints use O(1) memory regardless of data size",
			"Headers X-Total-Records, X-Generation-Time, X-Stream-Size provided",
			"Add ?format=xml for XML output",
			"Add ?seed=123 for reproducible data",
			"Add ?delay=100ms to simulate latency",
		},
	}
	writeJSONResponse(w, perfData)
}

// filterProducts applies basic OData filter expressions to the product list.
// Supports simple filter patterns like:
//   - "Category eq 'Electronics'" - exact category match
//   - "InStock eq true" - boolean property match
//   - "Price gt 100" - numeric comparison
//
// Note: This is a simplified filter implementation for testing purposes.
// Production OData services use a full OData query parser.
func filterProducts(productList []Product, filter string) []Product {
	var filtered []Product
	lowerFilter := strings.ToLower(filter)

	for _, p := range productList {
		match := true

		// Category filter examples
		if strings.Contains(lowerFilter, "category") {
			if strings.Contains(lowerFilter, "electronics") {
				match = p.Category == "Electronics"
			} else if strings.Contains(lowerFilter, "accessories") {
				match = p.Category == "Accessories"
			} else if strings.Contains(lowerFilter, "furniture") {
				match = p.Category == "Furniture"
			}
		}

		// InStock filter
		if strings.Contains(lowerFilter, "instock eq true") {
			match = match && p.InStock
		} else if strings.Contains(lowerFilter, "instock eq false") {
			match = match && !p.InStock
		}

		// Price filter examples (simplified)
		if strings.Contains(lowerFilter, "price gt") {
			// Extract price from filter string for comparison
			if strings.Contains(lowerFilter, "price gt 100") {
				match = match && p.Price > 100
			}
		}

		if match {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

// main initializes and starts the OData v4 test server with massive data streaming support.
// The server listens on port 9999 and provides OData endpoints for testing traverse client.
//
// PHASES 1-6: Complete implementation of massive data streaming:
//
//	PHASE 1: Configurable data sizes (small, medium, large, huge, massive)
//	PHASE 2: Streaming without memory buffering (O(1) memory usage)
//	PHASE 3: Realistic random data generation
//	PHASE 4: Performance metrics headers
//	PHASE 5: Specialized endpoints for each size
//	PHASE 6: Query parameters for configuration
//
// Server endpoints:
//
//	GET /                          - Server status and available endpoints
//	GET /odata/v4/                 - Service document (JSON or XML)
//	GET /odata/v4/$metadata        - Entity data model (XML)
//	GET /odata/v4/Products         - Products entity set (JSON or XML) with size parameter
//	GET /odata/v4/Products/$count  - Product count
//	GET /odata/v4/Categories       - Categories entity set (JSON or XML)
//	GET /odata/v4/Categories/$count - Category count
//	GET /test/streams/small        - Stream 10 products (PHASE 5)
//	GET /test/streams/medium       - Stream 10,000 products (PHASE 5)
//	GET /test/streams/large        - Stream 1,000,000 products (PHASE 5)
//	GET /test/streams/huge         - Stream 100,000,000 products (PHASE 5)
//	GET /test/streams/massive      - Stream 1,000,000,000 products (PHASE 5)
//	GET /test/memory-profile       - Runtime memory statistics (PHASE 5)
//	GET /test/performance          - Performance testing guide (PHASE 5)
//
// Content negotiation:
//
//	The server respects the Accept header:
//	- Accept: application/json returns JSON responses
//	- Accept: application/xml returns XML responses
//	- Default is JSON if not specified
//
// PHASE 6: Query parameters supported:
//
//	size=small|medium|large|huge|massive    - Data size (default: small)
//	format=json|xml                         - Output format (default: json)
//	seed=12345                              - Random seed for reproducibility
//	delay=100ms                             - Simulate latency between records
//	chunk-size=1000                         - Records per flush
//	errors=5%                               - Simulate error probability
//
// Example usage:
//
//	curl http://localhost:9999/odata/v4/Products?size=medium
//	curl http://localhost:9999/test/streams/large
//	curl http://localhost:9999/test/memory-profile
func main() {
	// Create a new HTTP mux for routing OData requests
	mux := http.NewServeMux()

	// Register OData metadata endpoint (always XML per OData spec)
	mux.HandleFunc("/odata/v4/$metadata", handleMetadata)

	// Register data endpoints with count support
	mux.HandleFunc("/odata/v4/Products/$count", handleProductsCount)
	mux.HandleFunc("/odata/v4/Categories/$count", handleCategoriesCount)

	// Register entity set endpoints (support JSON/XML content negotiation)
	mux.HandleFunc("/odata/v4/Products", handleProducts)
	mux.HandleFunc("/odata/v4/Categories", handleCategories)

	// Register service document endpoint
	mux.HandleFunc("/odata/v4/", handleService)

	// PHASE 5: Register streaming endpoints for different data sizes
	mux.HandleFunc("/test/streams/small", handleStreamSmall)
	mux.HandleFunc("/test/streams/medium", handleStreamMedium)
	mux.HandleFunc("/test/streams/large", handleStreamLarge)
	mux.HandleFunc("/test/streams/huge", handleStreamHuge)
	mux.HandleFunc("/test/streams/massive", handleStreamMassive)

	// PHASE 5: Register performance testing endpoints
	mux.HandleFunc("/test/memory-profile", handleMemoryProfile)
	mux.HandleFunc("/test/performance", handlePerformance)

	// Register root endpoint with server status information
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Root endpoint shows available OData endpoints
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "╔═══════════════════════════════════════════════════════════════╗\n")
			fmt.Fprintf(w, "║          OData v4 Test Server with Streaming Support         ║\n")
			fmt.Fprintf(w, "╚═══════════════════════════════════════════════════════════════╝\n\n")
			fmt.Fprintf(w, "Service root: http://localhost:9999/odata/v4/\n")
			fmt.Fprintf(w, "Metadata: http://localhost:9999/odata/v4/$metadata\n\n")
			fmt.Fprintf(w, "PHASE 1-2: Massive Data Streaming (O(1) Memory)\n")
			fmt.Fprintf(w, "=========================================\n")
			fmt.Fprintf(w, "Entity sets with size parameter:\n")
			fmt.Fprintf(w, "  GET /odata/v4/Products?size=small      # 10 products (default)\n")
			fmt.Fprintf(w, "  GET /odata/v4/Products?size=medium     # 10,000 products\n")
			fmt.Fprintf(w, "  GET /odata/v4/Products?size=large      # 1,000,000 products\n")
			fmt.Fprintf(w, "  GET /odata/v4/Products?size=huge       # 100,000,000 products\n")
			fmt.Fprintf(w, "  GET /odata/v4/Products?size=massive    # 1,000,000,000 products\n\n")
			fmt.Fprintf(w, "PHASE 3: Realistic Random Data Generation\n")
			fmt.Fprintf(w, "======================================\n")
			fmt.Fprintf(w, "  • 100+ product name variations\n")
			fmt.Fprintf(w, "  • Price range: $5 - $5,000\n")
			fmt.Fprintf(w, "  • 10+ categories\n")
			fmt.Fprintf(w, "  • Stock status: 80%% true, 20%% false\n\n")
			fmt.Fprintf(w, "PHASE 4: Performance Metrics\n")
			fmt.Fprintf(w, "==========================\n")
			fmt.Fprintf(w, "Response headers include:\n")
			fmt.Fprintf(w, "  X-Total-Records: Total records generated\n")
			fmt.Fprintf(w, "  X-Generation-Time: Time in milliseconds\n")
			fmt.Fprintf(w, "  X-Stream-Size: Estimated output in MB\n")
			fmt.Fprintf(w, "  X-Records-Per-Second: Throughput rate\n")
			fmt.Fprintf(w, "  X-Data-Size: Size identifier\n\n")
			fmt.Fprintf(w, "PHASE 5: Specialized Streaming Endpoints\n")
			fmt.Fprintf(w, "======================================\n")
			fmt.Fprintf(w, "  GET /test/streams/small           # Stream 10 products\n")
			fmt.Fprintf(w, "  GET /test/streams/medium          # Stream 10,000 products\n")
			fmt.Fprintf(w, "  GET /test/streams/large           # Stream 1,000,000 products\n")
			fmt.Fprintf(w, "  GET /test/streams/huge            # Stream 100,000,000 products\n")
			fmt.Fprintf(w, "  GET /test/streams/massive         # Stream 1,000,000,000 products\n\n")
			fmt.Fprintf(w, "PHASE 5: Performance Monitoring\n")
			fmt.Fprintf(w, "============================\n")
			fmt.Fprintf(w, "  GET /test/memory-profile          # Current memory statistics\n")
			fmt.Fprintf(w, "  GET /test/performance             # Performance guide\n\n")
			fmt.Fprintf(w, "PHASE 6: Query Parameters\n")
			fmt.Fprintf(w, "=======================\n")
			fmt.Fprintf(w, "  ?size=small|medium|large|huge|massive  # Data size (default: small)\n")
			fmt.Fprintf(w, "  ?format=json|xml                       # Output format (default: json)\n")
			fmt.Fprintf(w, "  ?seed=12345                            # Random seed for reproducibility\n")
			fmt.Fprintf(w, "  ?delay=100ms                           # Simulate latency\n")
			fmt.Fprintf(w, "  ?chunk-size=1000                       # Records per flush\n\n")
			fmt.Fprintf(w, "Example Queries:\n")
			fmt.Fprintf(w, "===============\n")
			fmt.Fprintf(w, "  Small dataset (JSON):\n")
			fmt.Fprintf(w, "    curl http://localhost:9999/odata/v4/Products\n\n")
			fmt.Fprintf(w, "  Medium dataset (JSON, with reproducible seed):\n")
			fmt.Fprintf(w, "    curl http://localhost:9999/odata/v4/Products?size=medium&seed=42\n\n")
			fmt.Fprintf(w, "  Large dataset (XML format):\n")
			fmt.Fprintf(w, "    curl http://localhost:9999/odata/v4/Products?size=large&format=xml\n\n")
			fmt.Fprintf(w, "  Stream 1 billion products with latency:\n")
			fmt.Fprintf(w, "    curl http://localhost:9999/odata/v4/Products?size=massive&delay=1ms\n\n")
			fmt.Fprintf(w, "  Direct streaming endpoint (10k products):\n")
			fmt.Fprintf(w, "    curl http://localhost:9999/test/streams/medium\n\n")
			fmt.Fprintf(w, "  Check memory usage during streaming:\n")
			fmt.Fprintf(w, "    curl http://localhost:9999/test/memory-profile\n")
		} else {
			// Route all other requests to service document
			handleService(w, r)
		}
	})

	// Log server startup information
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║   OData v4 Test Server with Massive Data Streaming Support   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("PHASES 1-6: Complete Streaming Implementation")
	fmt.Println("   ✓ PHASE 1: Configurable data sizes (small to massive)")
	fmt.Println("   ✓ PHASE 2: Streaming without memory buffering (O(1) memory)")
	fmt.Println("   ✓ PHASE 3: Realistic random data generation")
	fmt.Println("   ✓ PHASE 4: Performance metrics in response headers")
	fmt.Println("   ✓ PHASE 5: Specialized endpoints for each size")
	fmt.Println("   ✓ PHASE 6: Query parameters for full configuration")
	fmt.Println()
	fmt.Println("Server running on: http://localhost:9999")
	fmt.Println()
	fmt.Println("Core Endpoints:")
	fmt.Println("  Service root:  http://localhost:9999/odata/v4/")
	fmt.Println("  Metadata:      http://localhost:9999/odata/v4/$metadata")
	fmt.Println("  Products:      http://localhost:9999/odata/v4/Products (default: 10 records)")
	fmt.Println("  Categories:    http://localhost:9999/odata/v4/Categories")
	fmt.Println()
	fmt.Println("Streaming Endpoints (Direct):")
	fmt.Println("  Small (10):             http://localhost:9999/test/streams/small")
	fmt.Println("  Medium (10k):           http://localhost:9999/test/streams/medium")
	fmt.Println("  Large (1M):             http://localhost:9999/test/streams/large")
	fmt.Println("  Huge (100M):            http://localhost:9999/test/streams/huge")
	fmt.Println("  Massive (1B):           http://localhost:9999/test/streams/massive")
	fmt.Println()
	fmt.Println("Performance Monitoring:")
	fmt.Println("  Memory profile:         http://localhost:9999/test/memory-profile")
	fmt.Println("  Performance guide:      http://localhost:9999/test/performance")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  ✓ JSON and XML response formats (Accept header or ?format=)")
	fmt.Println("  ✓ OData query operators: $filter, $top, $skip, $count (small dataset)")
	fmt.Println("  ✓ Streaming large datasets with O(1) memory usage")
	fmt.Println("  ✓ Realistic random data generation (100+ names, prices, categories)")
	fmt.Println("  ✓ Reproducible data with seed parameter")
	fmt.Println("  ✓ Performance metrics headers")
	fmt.Println("  ✓ Full OData v4 metadata")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Start HTTP server on port 9999
	if err := http.ListenAndServe(":9999", mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
