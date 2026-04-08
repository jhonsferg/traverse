package traverse

import (
	"fmt"
	"testing"
)

// BenchmarkClientConstruction tests the performance of creating a new client
func BenchmarkClientConstruction(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := New(WithBaseURL("http://localhost:8080/odata"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryBuilderConstruction tests the performance of creating a query builder
func BenchmarkQueryBuilderConstruction(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = client.From("Products")
	}
}

// BenchmarkQueryBuilderSelect benchmarks the Select method
func BenchmarkQueryBuilderSelect(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	qb := client.From("Products")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb.Select("ID", "Name", "Price", "Category", "Stock")
	}
}

// BenchmarkQueryBuilderFilter benchmarks the Filter method
func BenchmarkQueryBuilderFilter(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	qb := client.From("Products")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb.Filter("Status eq 'Active' and Price gt 100")
	}
}

// BenchmarkQueryBuilderChaining benchmarks complex query chaining
func BenchmarkQueryBuilderChaining(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = client.From("Orders").
			Select("OrderID", "CustomerID", "OrderDate", "Total").
			Filter("OrderDate ge 2024-01-01").
			OrderBy("CustomerID").
			OrderByDesc("Total").
			Top(1000).
			Skip(0).
			WithCount()
	}
}

// BenchmarkQueryBuilderExpand benchmarks the Expand method
func BenchmarkQueryBuilderExpand(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	qb := client.From("Orders")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb.Expand("Customer")
		qb.Expand("Items")
		qb.Expand("ShippingAddress")
	}
}

// BenchmarkDateTimeValue benchmarks DateTime literal generation
func BenchmarkDateTimeValue(b *testing.B) {
	t := DateTime{} // Dummy time
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DateTimeValue(t.Time())
	}
}

// BenchmarkGuidValue benchmarks GUID literal generation
func BenchmarkGuidValue(b *testing.B) {
	guid := "550e8400-e29b-41d4-a716-446655440000"
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GuidValue(guid)
	}
}

// BenchmarkDecimalValue benchmarks Decimal literal generation
func BenchmarkDecimalValue(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DecimalValue(float64(i) * 1.5)
	}
}

// BenchmarkODataErrorCreation benchmarks OData error creation
func BenchmarkODataErrorCreation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = &ODataError{
			Code:    fmt.Sprintf("ERROR_%d", i),
			Message: "An error occurred",
		}
	}
}

// BenchmarkODataErrorError benchmarks OData error string formatting
func BenchmarkODataErrorError(b *testing.B) {
	err := &ODataError{
		Code:    "NotFound",
		Message: "Resource not found",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// ─────────────────────────────────────────────────────────────────────────
// PHASE 2 QUERY OPTIMIZATION BENCHMARKS
// ─────────────────────────────────────────────────────────────────────────

// BenchmarkURLCaching benchmarks URL caching efficiency
func BenchmarkURLCaching(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	qb := client.From("Products").
		Select("ID", "Name", "Price").
		Filter("Status eq 'Active'").
		OrderBy("Price").
		Top(100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = qb.buildURL()
	}
}

// BenchmarkURLCachingWithoutCaching benchmarks URL building with forced invalidation
func BenchmarkURLCachingWithoutCaching(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	qb := client.From("Products").
		Select("ID", "Name", "Price").
		Filter("Status eq 'Active'").
		OrderBy("Price").
		Top(100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Force URL rebuild each time
		qb.urlDirty = true
		_ = qb.buildURL()
	}
}

// BenchmarkSelectFields benchmarks Select() method performance
func BenchmarkSelectFields(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	qb := client.From("Products")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb.Select("ID", "Name", "Price", "Category", "Stock", "LastUpdated")
	}
}

// BenchmarkValidateFilter benchmarks filter validation performance
func BenchmarkValidateFilter(b *testing.B) {
	filterExprs := []string{
		"Status eq 'Active'",
		"Name eq 'John' and Age gt 18",
		"startswith(Email, 'admin') or endswith(Email, '@admin.com')",
		"Year(CreatedDate) eq 2024 and Price gt 100",
		"Category in ('Electronics', 'Books', 'Media')",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ValidateFilter(filterExprs[i%len(filterExprs)])
	}
}

// BenchmarkValidateFilterComplex benchmarks validation of complex filters
func BenchmarkValidateFilterComplex(b *testing.B) {
	complexFilter := "((Status eq 'Active' or Status eq 'Pending') and (Price gt 100 and Price lt 1000)) or " +
		"(startswith(Name, 'Special') and endswith(Category, 'Premium'))"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ValidateFilter(complexFilter)
	}
}

// BenchmarkQueryBuilderWithURLCache benchmarks full query building with caching
func BenchmarkQueryBuilderWithURLCache(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb := client.From("Orders").
			Select("OrderID", "CustomerID", "OrderDate", "Total").
			Filter("Status eq 'Completed'").
			OrderBy("OrderDate").
			Top(1000)

		_ = qb.buildURL()
		_ = qb.buildURL() // Should hit cache
	}
}

// BenchmarkTypeCastSegment benchmarks AsType path construction (0 allocs expected after URL cache warm).
func BenchmarkTypeCastSegment(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb := client.From("Employees").AsType("NS.Manager")
		_ = qb.buildURL()
		_ = qb.buildURL() // hit cache
	}
}

// BenchmarkExpandLevels benchmarks Expand with WithExpandLevels (0 allocs for filter portion).
func BenchmarkExpandLevels(b *testing.B) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qb := client.From("Categories").
			Expand("Products", WithExpandLevels(3))
		_ = qb.buildURL()
	}
}
