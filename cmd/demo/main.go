// Package main implements a comprehensive OData v4 test server supporting both XML and JSON formats.
package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DataSize represents the magnitude of records to generate.
type DataSize int

const (
	DataSizeSmall   DataSize = iota // 10 records
	DataSizeMedium                  // 10,000 records
	DataSizeLarge                   // 1,000,000 records
	DataSizeHuge                    // 100,000,000 records
	DataSizeMassive                 // 1,000,000,000 records
)

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
	default:
		return DataSizeSmall
	}
}

// DataGenerator generates Product records on-the-fly without buffering.
type DataGenerator interface {
	Next() (*Product, bool)
	Total() int64
	Reset()
}

// ProductGenerator implements DataGenerator for Product streams.
type ProductGenerator struct {
	size         DataSize
	current      int64
	total        int64
	rng          *rand.Rand
	seed         int64
	delay        time.Duration
	productNames []string
	categories   []string
}

func NewProductGenerator(size DataSize, seed int64, delay time.Duration) *ProductGenerator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	//nolint:gosec // Weak random number generator is fine for demo reproducibility
	rng := rand.New(rand.NewSource(seed))

	return &ProductGenerator{
		size:  size,
		total: size.Count(),
		rng:   rng,
		seed:  seed,
		delay: delay,
		productNames: []string{
			"Laptop", "Ultrabook", "Desktop", "Tablet", "iPad", "Smartphone", "Monitor", "Mouse", "Keyboard",
		},
		categories: []string{
			"Electronics", "Accessories", "Furniture", "Clothing", "Books", "Sports",
		},
	}
}

func (pg *ProductGenerator) Next() (*Product, bool) {
	if pg.current >= pg.total {
		return nil, false
	}

	if pg.delay > 0 {
		time.Sleep(pg.delay)
	}

	id := pg.current + 1
	productName := pg.productNames[pg.rng.Intn(len(pg.productNames))]
	category := pg.categories[pg.rng.Intn(len(pg.categories))]

	price := 5.0 + pg.rng.Float64()*(5000.0-5.0)
	price = math.Round(price*100) / 100

	inStock := pg.rng.Float64() < 0.8

	pg.current++

	return &Product{
		ID:       int(id),
		Name:     fmt.Sprintf("%s #%d", productName, id),
		Price:    price,
		Category: category,
		InStock:  inStock,
	}, true
}

func (pg *ProductGenerator) Total() int64 {
	return pg.total
}

func (pg *ProductGenerator) Reset() {
	pg.current = 0
	//nolint:gosec // Weak random number generator is fine for demo reproducibility
	pg.rng = rand.New(rand.NewSource(pg.seed))
}

const metadataXML = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <edmx:DataServices>
    <Schema Namespace="DemoService" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Price" Type="Edm.Decimal"/>
        <Property Name="Category" Type="Edm.String"/>
        <Property Name="InStock" Type="Edm.Boolean"/>
      </EntityType>
      <EntityType Name="Category">
        <Key><PropertyRef Name="ID"/></Key>
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

type Product struct {
	ID       int     `json:"ID" xml:"ID"`
	Name     string  `json:"Name" xml:"Name"`
	Price    float64 `json:"Price" xml:"Price"`
	Category string  `json:"Category" xml:"Category"`
	InStock  bool    `json:"InStock" xml:"InStock"`
}

type Category struct {
	ID          int    `json:"ID" xml:"ID"`
	Name        string `json:"Name" xml:"Name"`
	Description string `json:"Description" xml:"Description"`
}

var products = []Product{
	{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics", InStock: true},
	{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics", InStock: true},
	{ID: 10, Name: "Webcam", Price: 79.99, Category: "Electronics", InStock: true},
}

type PerformanceMetrics struct {
	StartTime      time.Time
	TotalRecords   int64
	EstimatedMB    float64
	GenerationTime time.Duration
	DataSize       DataSize
}

func getAcceptFormat(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if strings.Contains(strings.ToLower(accept), "application/xml") {
		return "xml"
	}
	return "json"
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(data)
}

func writeString(w http.ResponseWriter, s string) {
	_, _ = w.Write([]byte(s))
}

func writeBytes(w http.ResponseWriter, b []byte) {
	_, _ = w.Write(b)
}

func streamProducts(w http.ResponseWriter, generator DataGenerator, format string, metrics *PerformanceMetrics) {
	startGen := time.Now()

	// Pre-set headers
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Total-Records", strconv.FormatInt(generator.Total(), 10))
	w.Header().Set("X-Data-Size", metrics.DataSize.String())

	if format == "xml" {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		writeString(w, `<?xml version="1.0" encoding="utf-8"?>`)
		writeString(w, "\n<response>\n")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		writeString(w, "{\n  \"value\": [\n")
	}

	flusher, _ := w.(http.Flusher)
	count := 0

	for {
		product, hasMore := generator.Next()
		if !hasMore {
			break
		}

		if format == "xml" {
			xmlData, err := xml.Marshal(product)
			if err == nil {
				writeString(w, "    ")
				writeBytes(w, xmlData)
				writeString(w, "\n")
			}
		} else {
			data, err := json.Marshal(product)
			if err == nil {
				writeBytes(w, data)
				if int64(count) < generator.Total()-1 {
					writeString(w, ",")
				}
				writeString(w, "\n")
			}
		}

		count++
		if count%100 == 0 && flusher != nil {
			flusher.Flush()
		}
	}

	if format == "xml" {
		writeString(w, "  </response>\n")
	} else {
		writeString(w, "  ]\n}\n")
	}

	metrics.GenerationTime = time.Since(startGen)
	if flusher != nil {
		flusher.Flush()
	}
}

func handleMetadata(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = fmt.Fprint(w, metadataXML)
}

func handleService(w http.ResponseWriter, r *http.Request) {
	format := getAcceptFormat(r)
	if format == "xml" {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = fmt.Fprint(w, `<service xmlns="http://www.w3.org/2007/app">OK</service>`)
	} else {
		writeJSONResponse(w, map[string]string{"status": "ok"})
	}
}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	dataSize := GetDataSizeFromString(query.Get("size"))
	format := query.Get("format")
	if format == "" {
		format = getAcceptFormat(r)
	}

	if dataSize == DataSizeSmall {
		writeJSONResponse(w, map[string]interface{}{"value": products})
		return
	}

	generator := NewProductGenerator(dataSize, 0, 0)
	streamProducts(w, generator, format, &PerformanceMetrics{DataSize: dataSize})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/odata/v4/$metadata", handleMetadata)
	mux.HandleFunc("/odata/v4/", handleService)
	mux.HandleFunc("/odata/v4/Products", handleProducts)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writeString(w, "OData Demo Server\n")
	})

	server := &http.Server{
		Addr:         ":9999",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 300 * time.Second,
	}

	log.Printf("Server starting on http://localhost:9999")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
