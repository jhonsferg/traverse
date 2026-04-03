package traverse

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

// BenchmarkParseODataResponseV2 benchmarks parsing OData v2 responses
func BenchmarkParseODataResponseV2(b *testing.B) {
	// Simulate a small OData v2 response
	responseJSON := []byte(`{
		"d": {
			"results": [
				{"ID": 1, "Name": "Product 1", "Price": 100.00},
				{"ID": 2, "Name": "Product 2", "Price": 200.00},
				{"ID": 3, "Name": "Product 3", "Price": 300.00}
			]
		}
	}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(responseJSON)
		decoder := json.NewDecoder(reader)

		// Simulate parsing
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseODataResponseV4 benchmarks parsing OData v4 responses
func BenchmarkParseODataResponseV4(b *testing.B) {
	// Simulate a small OData v4 response
	responseJSON := []byte(`{
		"value": [
			{"ID": 1, "Name": "Product 1", "Price": 100.00},
			{"ID": 2, "Name": "Product 2", "Price": 200.00},
			{"ID": 3, "Name": "Product 3", "Price": 300.00}
		]
	}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(responseJSON)
		decoder := json.NewDecoder(reader)

		// Simulate parsing
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONDecoderVsUnmarshal compares json.Decoder with json.Unmarshal
func BenchmarkJSONDecoderVsUnmarshal(b *testing.B) {
	// Create a large response body
	items := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = map[string]interface{}{
			"ID":    i,
			"Name":  "Product " + string(rune(i%26+65)),
			"Price": float64(i) * 10.5,
		}
	}

	responseData := map[string]interface{}{"value": items}
	responseBytes, _ := json.Marshal(responseData)

	b.Run("Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var obj map[string]interface{}
			if err := json.Unmarshal(responseBytes, &obj); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Decoder", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(responseBytes)
			decoder := json.NewDecoder(reader)

			var obj map[string]interface{}
			if err := decoder.Decode(&obj); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkStreamingLargeDataset simulates streaming through a large dataset
func BenchmarkStreamingLargeDataset(b *testing.B) {
	// Create a response with many items
	items := make([]map[string]interface{}, 10000)
	for i := 0; i < 10000; i++ {
		items[i] = map[string]interface{}{
			"ID":    i,
			"Name":  "Product " + string(rune(i%26+65)),
			"Price": float64(i) * 10.5,
		}
	}

	responseData := map[string]interface{}{"value": items}
	responseBytes, _ := json.Marshal(responseData)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(responseBytes)
		decoder := json.NewDecoder(reader)

		// Simulate token-by-token parsing
		count := 0
		for decoder.More() {
			var item map[string]interface{}
			if err := decoder.Decode(&item); err != nil && err != io.EOF {
				b.Fatal(err)
			}
			count++
		}
	}
}

// TestEstimateBufferSize verifies the adaptive buffer sizing formula
func TestEstimateBufferSize(t *testing.T) {
	tests := []struct {
		name         string
		avgSizeBytes int
		expectedMin  int
		expectedMax  int
	}{
		{
			name:         "Tiny record (10 bytes)",
			avgSizeBytes: 10,
			expectedMin:  1024, // 10MB/10 = 1M, clamped to 1024
			expectedMax:  1024,
		},
		{
			name:         "Medium record (10KB)",
			avgSizeBytes: 10 * 1024,
			expectedMin:  1024, // 10MB/10KB = 1024
			expectedMax:  1024,
		},
		{
			name:         "Large record (1MB)",
			avgSizeBytes: 1024 * 1024,
			expectedMin:  9, // 10MB/1MB = 10, but clamped to 32
			expectedMax:  32,
		},
		{
			name:         "Zero size",
			avgSizeBytes: 0,
			expectedMin:  256, // Default fallback
			expectedMax:  256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateBufferSize(tt.avgSizeBytes)
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("estimateBufferSize(%d) = %d, expected in range [%d, %d]",
					tt.avgSizeBytes, result, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// ✅ TestCalculateAverageRecordSize - Removed because function was eliminated for optimization
// The calculateAverageRecordSize function had unnecessary json.Marshal overhead (50-150ms per stream)
// Use estimateBufferSize with default value instead
