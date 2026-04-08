package traverse_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse"
	"github.com/jhonsferg/traverse/testutil"
)

// ---------------------------------------------------------------------------
// C1 - Geospatial primitive types
// ---------------------------------------------------------------------------

func TestGeographyPoint_ODataLiteral(t *testing.T) {
	p := traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518}
	got := p.ODataLiteral()
	want := "geography'SRID=4326;POINT(13.408 52.518)'"
	if got != want {
		t.Errorf("ODataLiteral: got %q, want %q", got, want)
	}
}

func TestGeographyPoint_ODataLiteral_NegativeCoords(t *testing.T) {
	p := traverse.GeographyPoint{Longitude: -74.006, Latitude: 40.7128}
	lit := p.ODataLiteral()
	if !strings.HasPrefix(lit, "geography'") {
		t.Errorf("expected geography' prefix, got %q", lit)
	}
	if !strings.Contains(lit, "-74.006") {
		t.Errorf("expected -74.006 in literal, got %q", lit)
	}
}

func TestGeometryPoint_ODataLiteral(t *testing.T) {
	p := traverse.GeometryPoint{X: 1.5, Y: 2.5}
	got := p.ODataLiteral()
	want := "geometry'SRID=0;POINT(1.5 2.5)'"
	if got != want {
		t.Errorf("ODataLiteral: got %q, want %q", got, want)
	}
}

func TestGeographyLineString_ODataLiteral(t *testing.T) {
	ls := traverse.GeographyLineString{
		Points: []traverse.GeographyPoint{
			{Longitude: 0, Latitude: 0},
			{Longitude: 1, Latitude: 1},
			{Longitude: 2, Latitude: 0},
		},
	}
	lit := ls.ODataLiteral()
	if !strings.HasPrefix(lit, "geography'SRID=4326;LINESTRING(") {
		t.Errorf("unexpected literal: %q", lit)
	}
	if !strings.Contains(lit, "0 0,1 1,2 0") {
		t.Errorf("unexpected coordinates in: %q", lit)
	}
}

func TestGeometryLineString_ODataLiteral(t *testing.T) {
	ls := traverse.GeometryLineString{
		Points: []traverse.GeometryPoint{{X: 0, Y: 0}, {X: 1, Y: 1}},
	}
	lit := ls.ODataLiteral()
	if !strings.HasPrefix(lit, "geometry'SRID=0;LINESTRING(") {
		t.Errorf("unexpected literal: %q", lit)
	}
}

func TestGeographyPolygon_ODataLiteral_NoHoles(t *testing.T) {
	poly := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0},
		},
	}
	lit := poly.ODataLiteral()
	if !strings.HasPrefix(lit, "geography'SRID=4326;POLYGON((") {
		t.Errorf("unexpected literal: %q", lit)
	}
	if !strings.HasSuffix(lit, "))'") {
		t.Errorf("unexpected suffix in literal: %q", lit)
	}
}

func TestGeographyPolygon_ODataLiteral_WithHole(t *testing.T) {
	poly := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
		},
		InteriorRings: [][]traverse.GeographyPoint{
			{{1, 1}, {2, 1}, {2, 2}, {1, 2}, {1, 1}},
		},
	}
	lit := poly.ODataLiteral()
	// Must contain two ring groups
	if strings.Count(lit, "),(") < 1 {
		t.Errorf("expected inner ring separator, got: %q", lit)
	}
}

func TestParseGeographyPoint_Valid(t *testing.T) {
	tests := []struct {
		input string
		lon   float64
		lat   float64
	}{
		{"geography'SRID=4326;POINT(13.408 52.518)'", 13.408, 52.518},
		{"POINT(1.5 2.5)", 1.5, 2.5},
		{"geography'SRID=4326;POINT(-74.006 40.7128)'", -74.006, 40.7128},
	}
	for _, tt := range tests {
		p, err := traverse.ParseGeographyPoint(tt.input)
		if err != nil {
			t.Errorf("ParseGeographyPoint(%q): %v", tt.input, err)
			continue
		}
		if p.Longitude != tt.lon || p.Latitude != tt.lat {
			t.Errorf("ParseGeographyPoint(%q): got (%v,%v), want (%v,%v)",
				tt.input, p.Longitude, p.Latitude, tt.lon, tt.lat)
		}
	}
}

func TestParseGeographyPoint_Invalid(t *testing.T) {
	_, err := traverse.ParseGeographyPoint("not a point")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseGeometryPoint_Valid(t *testing.T) {
	p, err := traverse.ParseGeometryPoint("geometry'SRID=0;POINT(3.14 2.72)'")
	if err != nil {
		t.Fatalf("ParseGeometryPoint: %v", err)
	}
	if p.X != 3.14 || p.Y != 2.72 {
		t.Errorf("got (%v,%v), want (3.14,2.72)", p.X, p.Y)
	}
}

// ---------------------------------------------------------------------------
// C1 - Pool acquire/release
// ---------------------------------------------------------------------------

func TestGeoPointPool_AcquireRelease(t *testing.T) {
	p := traverse.AcquireGeographyPoint(13.0, 52.0)
	if p == nil {
		t.Fatal("AcquireGeographyPoint returned nil")
	}
	if p.Longitude != 13.0 || p.Latitude != 52.0 {
		t.Errorf("got (%v,%v), want (13,52)", p.Longitude, p.Latitude)
	}
	traverse.ReleaseGeographyPoint(p)
}

func TestGeometryPointPool_AcquireRelease(t *testing.T) {
	p := traverse.AcquireGeometryPoint(1.0, 2.0)
	if p == nil {
		t.Fatal("AcquireGeometryPoint returned nil")
	}
	if p.X != 1.0 || p.Y != 2.0 {
		t.Errorf("got (%v,%v), want (1,2)", p.X, p.Y)
	}
	traverse.ReleaseGeometryPoint(p)
}

// ---------------------------------------------------------------------------
// C3 - GeoJSON serialization / deserialization
// ---------------------------------------------------------------------------

func TestGeographyPoint_MarshalJSON(t *testing.T) {
	p := traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"type":"Point"`) {
		t.Errorf("missing type field: %s", got)
	}
	if !strings.Contains(got, "13.408") || !strings.Contains(got, "52.518") {
		t.Errorf("missing coordinates: %s", got)
	}
}

func TestGeographyPoint_UnmarshalJSON(t *testing.T) {
	data := []byte(`{"type":"Point","coordinates":[13.408,52.518]}`)
	var p traverse.GeographyPoint
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if p.Longitude != 13.408 || p.Latitude != 52.518 {
		t.Errorf("got (%v,%v), want (13.408,52.518)", p.Longitude, p.Latitude)
	}
}

func TestGeographyPoint_JSONRoundtrip(t *testing.T) {
	original := traverse.GeographyPoint{Longitude: -122.4194, Latitude: 37.7749}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded traverse.GeographyPoint
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Longitude != original.Longitude || decoded.Latitude != original.Latitude {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestGeometryPoint_JSONRoundtrip(t *testing.T) {
	original := traverse.GeometryPoint{X: 3.14, Y: -2.72}
	data, _ := json.Marshal(original)
	var decoded traverse.GeometryPoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.X != original.X || decoded.Y != original.Y {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestGeographyLineString_JSONRoundtrip(t *testing.T) {
	original := traverse.GeographyLineString{
		Points: []traverse.GeographyPoint{
			{Longitude: 0, Latitude: 0},
			{Longitude: 1, Latitude: 1},
			{Longitude: 2, Latitude: 0},
		},
	}
	data, _ := json.Marshal(original)
	var decoded traverse.GeographyLineString
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.Points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(decoded.Points))
	}
	if decoded.Points[1].Longitude != 1 || decoded.Points[1].Latitude != 1 {
		t.Errorf("point 1: got %+v, want (1,1)", decoded.Points[1])
	}
}

func TestGeometryLineString_JSONRoundtrip(t *testing.T) {
	original := traverse.GeometryLineString{
		Points: []traverse.GeometryPoint{{X: 0, Y: 0}, {X: 5, Y: 5}},
	}
	data, _ := json.Marshal(original)
	var decoded traverse.GeometryLineString
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.Points) != 2 || decoded.Points[1].X != 5 {
		t.Errorf("roundtrip mismatch: %+v", decoded)
	}
}

func TestGeographyPolygon_JSONRoundtrip(t *testing.T) {
	original := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
		},
	}
	data, _ := json.Marshal(original)
	var decoded traverse.GeographyPolygon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.ExteriorRing) != 5 {
		t.Errorf("exterior ring: got %d points, want 5", len(decoded.ExteriorRing))
	}
}

func TestGeographyPolygon_JSONRoundtrip_WithHole(t *testing.T) {
	original := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
		},
		InteriorRings: [][]traverse.GeographyPoint{
			{{1, 1}, {2, 1}, {2, 2}, {1, 2}, {1, 1}},
		},
	}
	data, _ := json.Marshal(original)
	var decoded traverse.GeographyPolygon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.InteriorRings) != 1 {
		t.Errorf("interior rings: got %d, want 1", len(decoded.InteriorRings))
	}
	if len(decoded.InteriorRings[0]) != 5 {
		t.Errorf("interior ring 0: got %d points, want 5", len(decoded.InteriorRings[0]))
	}
}

func TestGeographyMultiPoint_JSONRoundtrip(t *testing.T) {
	original := traverse.GeographyMultiPoint{
		Points: []traverse.GeographyPoint{{1, 2}, {3, 4}, {5, 6}},
	}
	data, _ := json.Marshal(original)
	var decoded traverse.GeographyMultiPoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.Points) != 3 {
		t.Errorf("expected 3 points, got %d", len(decoded.Points))
	}
}

func TestGeometryMultiPoint_JSONRoundtrip(t *testing.T) {
	original := traverse.GeometryMultiPoint{
		Points: []traverse.GeometryPoint{{1, 2}, {3, 4}},
	}
	data, _ := json.Marshal(original)
	var decoded traverse.GeometryMultiPoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.Points) != 2 {
		t.Errorf("expected 2 points, got %d", len(decoded.Points))
	}
}

// ---------------------------------------------------------------------------
// C2 - Geospatial filter functions
// ---------------------------------------------------------------------------

func TestGeoDistanceFilter(t *testing.T) {
	p := traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518}
	expr := traverse.GeoDistanceFilter("Location", p, "le", 1000)
	if !strings.HasPrefix(expr, "geo.distance(Location,") {
		t.Errorf("unexpected prefix: %q", expr)
	}
	if !strings.Contains(expr, "geography'SRID=4326;POINT(13.408 52.518)'") {
		t.Errorf("missing geography literal: %q", expr)
	}
	if !strings.HasSuffix(expr, "le 1000") {
		t.Errorf("unexpected suffix: %q", expr)
	}
}

func TestGeoDistanceFilterGeom(t *testing.T) {
	p := traverse.GeometryPoint{X: 1.5, Y: 2.5}
	expr := traverse.GeoDistanceFilterGeom("Pos", p, "lt", 500)
	if !strings.Contains(expr, "geo.distance(Pos,geometry'SRID=0;POINT(1.5 2.5)')") {
		t.Errorf("unexpected expr: %q", expr)
	}
	if !strings.HasSuffix(expr, "lt 500") {
		t.Errorf("unexpected suffix: %q", expr)
	}
}

func TestGeoIntersectsFilter(t *testing.T) {
	poly := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
		},
	}
	expr := traverse.GeoIntersectsFilter("Coordinates", poly)
	if !strings.HasPrefix(expr, "geo.intersects(Coordinates,geography'") {
		t.Errorf("unexpected expr: %q", expr)
	}
}

func TestGeoIntersectsFilterGeom(t *testing.T) {
	poly := traverse.GeometryPolygon{
		ExteriorRing: []traverse.GeometryPoint{{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}},
	}
	expr := traverse.GeoIntersectsFilterGeom("Pos", poly)
	if !strings.HasPrefix(expr, "geo.intersects(Pos,geometry'") {
		t.Errorf("unexpected expr: %q", expr)
	}
}

func TestGeoLengthFilter(t *testing.T) {
	expr := traverse.GeoLengthFilter("Path", "le", 50000)
	want := "geo.length(Path) le 50000"
	if expr != want {
		t.Errorf("GeoLengthFilter: got %q, want %q", expr, want)
	}
}

func TestQueryBuilder_GeoDistance(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Enqueue(testutil.MockResponse{Status: 200, Body: testutil.ODataResponse()})

	client, err := traverse.New(traverse.WithBaseURL(srv.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = client.From("Shops").GeoDistance(
		"Location",
		traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518},
		"le",
		500,
	).Page(t.Context())

	reqs := srv.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	filter := reqs[0].Query.Get("$filter")
	if !strings.Contains(filter, "geo.distance") {
		t.Errorf("filter missing geo.distance: %q", filter)
	}
	if !strings.Contains(filter, "13.408") {
		t.Errorf("filter missing longitude: %q", filter)
	}
}

func TestQueryBuilder_GeoDistance_InvalidOperator(t *testing.T) {
	client, err := traverse.New(traverse.WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = client.From("Shops").GeoDistance(
		"Location",
		traverse.GeographyPoint{Longitude: 0, Latitude: 0},
		"INVALID",
		100,
	).Page(t.Context())
	if err == nil {
		t.Error("expected error for invalid operator")
	}
}

func TestQueryBuilder_GeoIntersects(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Enqueue(testutil.MockResponse{Status: 200, Body: testutil.ODataResponse()})

	client, err := traverse.New(traverse.WithBaseURL(srv.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	poly := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
		},
	}
	_, _ = client.From("POI").GeoIntersects("Coordinates", poly).Page(t.Context())

	reqs := srv.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	filter := reqs[0].Query.Get("$filter")
	if !strings.Contains(filter, "geo.intersects") {
		t.Errorf("filter missing geo.intersects: %q", filter)
	}
}

func TestQueryBuilder_GeoLength(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Enqueue(testutil.MockResponse{Status: 200, Body: testutil.ODataResponse()})

	client, err := traverse.New(traverse.WithBaseURL(srv.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = client.From("Routes").GeoLength("Path", "le", 50000).Page(t.Context())

	reqs := srv.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	filter := reqs[0].Query.Get("$filter")
	if !strings.Contains(filter, "geo.length") {
		t.Errorf("filter missing geo.length: %q", filter)
	}
}

func TestQueryBuilder_GeoFilter_ChainsWithWhere(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Enqueue(testutil.MockResponse{Status: 200, Body: testutil.ODataResponse()})

	client, err := traverse.New(traverse.WithBaseURL(srv.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = client.From("Shops").
		Where("Category").Eq("Coffee").
		GeoDistance("Location", traverse.GeographyPoint{Longitude: 2, Latitude: 48}, "le", 300).
		Page(t.Context())

	reqs := srv.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	filter := reqs[0].Query.Get("$filter")
	if !strings.Contains(filter, "geo.distance") {
		t.Errorf("filter missing geo.distance: %q", filter)
	}
	if !strings.Contains(filter, "Category") {
		t.Errorf("filter missing Category: %q", filter)
	}
}

// ---------------------------------------------------------------------------
// F1 - Zero-alloc benchmarks for geospatial hot paths
// ---------------------------------------------------------------------------

func BenchmarkGeographyPoint_ODataLiteral(b *testing.B) {
	p := traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.ODataLiteral()
	}
}

func BenchmarkGeoDistanceFilter(b *testing.B) {
	p := traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = traverse.GeoDistanceFilter("Location", p, "le", 1000)
	}
}

func BenchmarkGeoIntersectsFilter(b *testing.B) {
	poly := traverse.GeographyPolygon{
		ExteriorRing: []traverse.GeographyPoint{
			{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = traverse.GeoIntersectsFilter("Coordinates", poly)
	}
}

func BenchmarkGeoLengthFilter(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = traverse.GeoLengthFilter("Path", "le", 50000)
	}
}

func BenchmarkGeoPointPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := traverse.AcquireGeographyPoint(13.408, 52.518)
		traverse.ReleaseGeographyPoint(p)
	}
}

func BenchmarkParseGeographyPoint(b *testing.B) {
	lit := "geography'SRID=4326;POINT(13.408 52.518)'"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = traverse.ParseGeographyPoint(lit)
	}
}
