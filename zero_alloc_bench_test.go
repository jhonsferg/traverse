package traverse

import (
	"testing"
)

// BenchmarkDateTimeValueBytes benchmarks DateTimeValueBytes
func BenchmarkDateTimeValueBytes(b *testing.B) {
	t := DateTime{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DateTimeValueBytes(t.Time())
	}
}

// BenchmarkDecimalValueBytes benchmarks DecimalValueBytes
func BenchmarkDecimalValueBytes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DecimalValueBytes(float64(i) * 1.5)
	}
}

// BenchmarkFormatParameterBytes benchmarks formatParameterBytes
func BenchmarkFormatParameterBytes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = formatParameterBytes("param", "value")
	}
}

// BenchmarkFormatParameter benchmarks formatParameter (string version)
func BenchmarkFormatParameter(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = formatParameter("param", "value")
	}
}
