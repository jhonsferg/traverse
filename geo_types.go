package traverse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Geospatial primitive types (OData Edm.Geography* / Edm.Geometry*)
//
// OData defines two families of spatial types:
//   - Geography: coordinate reference system is WGS84 (SRID=4326, lat/lon on earth surface)
//   - Geometry: coordinate reference system is a flat/projected plane (Euclidean)
//
// The URL literal format for OData filter expressions uses geography'SRID=4326;POINT(lon lat)'
// notation. GeoJSON uses {"type":"Point","coordinates":[lon,lat]} notation.
//
// All structs in this file are safe to copy by value. Pooled allocations are
// available via GeoPointPool for hot-path code (see F3 zero-alloc plan).
// ---------------------------------------------------------------------------

// GeographyPoint represents a WGS84 geographic coordinate (Edm.GeographyPoint).
//
// Coordinates follow the OData/WKT convention: Longitude first, then Latitude.
// This matches GeoJSON (RFC 7946) order.
//
// Example OData literal:  geography'SRID=4326;POINT(13.408 52.518)'
// Example GeoJSON:        {"type":"Point","coordinates":[13.408,52.518]}
type GeographyPoint struct {
	// Longitude is the east-west coordinate (X axis), range -180 to 180.
	Longitude float64
	// Latitude is the north-south coordinate (Y axis), range -90 to 90.
	Latitude float64
}

// GeometryPoint represents a point in a flat (Euclidean) coordinate system (Edm.GeometryPoint).
//
// Example OData literal:  geometry'SRID=0;POINT(1.5 2.5)'
// Example GeoJSON:        {"type":"Point","coordinates":[1.5,2.5]}
type GeometryPoint struct {
	X float64
	Y float64
}

// GeographyLineString represents a geographic line string (Edm.GeographyLineString).
// A LineString requires at least 2 points.
type GeographyLineString struct {
	Points []GeographyPoint
}

// GeometryLineString represents a planar line string (Edm.GeometryLineString).
type GeometryLineString struct {
	Points []GeometryPoint
}

// GeographyPolygon represents a geographic polygon (Edm.GeographyPolygon).
// ExteriorRing is the outer boundary; InteriorRings are holes.
// All rings must be closed (first point == last point).
type GeographyPolygon struct {
	ExteriorRing  []GeographyPoint
	InteriorRings [][]GeographyPoint
}

// GeometryPolygon represents a planar polygon (Edm.GeometryPolygon).
type GeometryPolygon struct {
	ExteriorRing  []GeometryPoint
	InteriorRings [][]GeometryPoint
}

// GeographyMultiPoint represents a collection of geographic points (Edm.GeographyMultiPoint).
type GeographyMultiPoint struct {
	Points []GeographyPoint
}

// GeometryMultiPoint represents a collection of planar points (Edm.GeometryMultiPoint).
type GeometryMultiPoint struct {
	Points []GeometryPoint
}

// ---------------------------------------------------------------------------
// sync.Pool for hot-path geographic point allocations (F3)
// ---------------------------------------------------------------------------

// geoPointPool is a pool of *GeographyPoint to reduce allocations in
// geospatial filter loops where many temporary points are created.
var geoPointPool = sync.Pool{
	New: func() interface{} { return new(GeographyPoint) },
}

// geometryPointPool is a pool of *GeometryPoint to reduce allocations.
var geometryPointPool = sync.Pool{
	New: func() interface{} { return new(GeometryPoint) },
}

// AcquireGeographyPoint returns a *GeographyPoint from the pool.
// The caller must call ReleaseGeographyPoint when done.
func AcquireGeographyPoint(lon, lat float64) *GeographyPoint {
	p := geoPointPool.Get().(*GeographyPoint)
	p.Longitude = lon
	p.Latitude = lat
	return p
}

// ReleaseGeographyPoint returns the point to the pool.
func ReleaseGeographyPoint(p *GeographyPoint) {
	p.Longitude = 0
	p.Latitude = 0
	geoPointPool.Put(p)
}

// AcquireGeometryPoint returns a *GeometryPoint from the pool.
func AcquireGeometryPoint(x, y float64) *GeometryPoint {
	p := geometryPointPool.Get().(*GeometryPoint)
	p.X = x
	p.Y = y
	return p
}

// ReleaseGeometryPoint returns the point to the pool.
func ReleaseGeometryPoint(p *GeometryPoint) {
	p.X = 0
	p.Y = 0
	geometryPointPool.Put(p)
}

// ---------------------------------------------------------------------------
// OData URL literal serialization (WKT-based)
// ---------------------------------------------------------------------------

// ODataLiteral returns the OData URL literal for a GeographyPoint.
// Format: geography'SRID=4326;POINT(lon lat)'
func (p GeographyPoint) ODataLiteral() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geography'SRID=4326;POINT(")
	formatCoordTo(buf, p.Longitude)
	buf.WriteByte(' ')
	formatCoordTo(buf, p.Latitude)
	buf.WriteString(")'")
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// ODataLiteral returns the OData URL literal for a GeometryPoint.
// Format: geometry'SRID=0;POINT(x y)'
func (p GeometryPoint) ODataLiteral() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geometry'SRID=0;POINT(")
	formatCoordTo(buf, p.X)
	buf.WriteByte(' ')
	formatCoordTo(buf, p.Y)
	buf.WriteString(")'")
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// ODataLiteral returns the OData URL literal for a GeographyLineString.
// Format: geography'SRID=4326;LINESTRING(lon1 lat1,lon2 lat2,...)'
func (ls GeographyLineString) ODataLiteral() string {
	return "geography'SRID=4326;LINESTRING(" + geoPointsToWKT(ls.Points) + ")'"
}

// ODataLiteral returns the OData URL literal for a GeometryLineString.
func (ls GeometryLineString) ODataLiteral() string {
	return "geometry'SRID=0;LINESTRING(" + geomPointsToWKT(ls.Points) + ")'"
}

// ODataLiteral returns the OData URL literal for a GeographyPolygon.
// Format: geography'SRID=4326;POLYGON((ring1),(ring2),...)'
func (poly GeographyPolygon) ODataLiteral() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geography'SRID=4326;POLYGON((")
	buf.WriteString(geoPointsToWKT(poly.ExteriorRing))
	buf.WriteByte(')')
	for _, ring := range poly.InteriorRings {
		buf.WriteString(",(")
		buf.WriteString(geoPointsToWKT(ring))
		buf.WriteByte(')')
	}
	buf.WriteByte(')')
	buf.WriteByte('\'')
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// ODataLiteral returns the OData URL literal for a GeometryPolygon.
func (poly GeometryPolygon) ODataLiteral() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geometry'SRID=0;POLYGON((")
	buf.WriteString(geomPointsToWKT(poly.ExteriorRing))
	buf.WriteByte(')')
	for _, ring := range poly.InteriorRings {
		buf.WriteString(",(")
		buf.WriteString(geomPointsToWKT(ring))
		buf.WriteByte(')')
	}
	buf.WriteByte(')')
	buf.WriteByte('\'')
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// ---------------------------------------------------------------------------
// GeoJSON serialization (C3)
// ---------------------------------------------------------------------------

// MarshalJSON implements json.Marshaler for GeographyPoint.
// Produces GeoJSON Point: {"type":"Point","coordinates":[lon,lat]}
func (p GeographyPoint) MarshalJSON() ([]byte, error) {
	return marshalGeoJSONPoint(p.Longitude, p.Latitude)
}

// UnmarshalJSON implements json.Unmarshaler for GeographyPoint.
func (p *GeographyPoint) UnmarshalJSON(data []byte) error {
	lon, lat, err := unmarshalGeoJSONPoint(data)
	if err != nil {
		return err
	}
	p.Longitude = lon
	p.Latitude = lat
	return nil
}

// MarshalJSON implements json.Marshaler for GeometryPoint.
// Produces GeoJSON Point: {"type":"Point","coordinates":[x,y]}
func (p GeometryPoint) MarshalJSON() ([]byte, error) {
	return marshalGeoJSONPoint(p.X, p.Y)
}

// UnmarshalJSON implements json.Unmarshaler for GeometryPoint.
func (p *GeometryPoint) UnmarshalJSON(data []byte) error {
	x, y, err := unmarshalGeoJSONPoint(data)
	if err != nil {
		return err
	}
	p.X = x
	p.Y = y
	return nil
}

// MarshalJSON implements json.Marshaler for GeographyLineString.
func (ls GeographyLineString) MarshalJSON() ([]byte, error) {
	coords := make([][2]float64, len(ls.Points))
	for i, p := range ls.Points {
		coords[i] = [2]float64{p.Longitude, p.Latitude}
	}
	return marshalGeoJSONGeometry("LineString", coords)
}

// UnmarshalJSON implements json.Unmarshaler for GeographyLineString.
func (ls *GeographyLineString) UnmarshalJSON(data []byte) error {
	coords, err := unmarshalGeoJSONCoords(data, "LineString")
	if err != nil {
		return err
	}
	ls.Points = make([]GeographyPoint, len(coords))
	for i, c := range coords {
		ls.Points[i] = GeographyPoint{Longitude: c[0], Latitude: c[1]}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for GeometryLineString.
func (ls GeometryLineString) MarshalJSON() ([]byte, error) {
	coords := make([][2]float64, len(ls.Points))
	for i, p := range ls.Points {
		coords[i] = [2]float64{p.X, p.Y}
	}
	return marshalGeoJSONGeometry("LineString", coords)
}

// UnmarshalJSON implements json.Unmarshaler for GeometryLineString.
func (ls *GeometryLineString) UnmarshalJSON(data []byte) error {
	coords, err := unmarshalGeoJSONCoords(data, "LineString")
	if err != nil {
		return err
	}
	ls.Points = make([]GeometryPoint, len(coords))
	for i, c := range coords {
		ls.Points[i] = GeometryPoint{X: c[0], Y: c[1]}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for GeographyPolygon.
func (poly GeographyPolygon) MarshalJSON() ([]byte, error) {
	rings := make([][][2]float64, 1+len(poly.InteriorRings))
	rings[0] = geoRingToCoords(poly.ExteriorRing)
	for i, ring := range poly.InteriorRings {
		rings[i+1] = geoRingToCoords(ring)
	}
	return marshalGeoJSONGeometry("Polygon", rings)
}

// UnmarshalJSON implements json.Unmarshaler for GeographyPolygon.
func (poly *GeographyPolygon) UnmarshalJSON(data []byte) error {
	rings, err := unmarshalGeoJSONRings(data, "Polygon")
	if err != nil {
		return err
	}
	if len(rings) == 0 {
		return nil
	}
	poly.ExteriorRing = make([]GeographyPoint, len(rings[0]))
	for i, c := range rings[0] {
		poly.ExteriorRing[i] = GeographyPoint{Longitude: c[0], Latitude: c[1]}
	}
	poly.InteriorRings = make([][]GeographyPoint, len(rings)-1)
	for ri, ring := range rings[1:] {
		poly.InteriorRings[ri] = make([]GeographyPoint, len(ring))
		for i, c := range ring {
			poly.InteriorRings[ri][i] = GeographyPoint{Longitude: c[0], Latitude: c[1]}
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for GeometryPolygon.
func (poly GeometryPolygon) MarshalJSON() ([]byte, error) {
	rings := make([][][2]float64, 1+len(poly.InteriorRings))
	rings[0] = geomRingToCoords(poly.ExteriorRing)
	for i, ring := range poly.InteriorRings {
		rings[i+1] = geomRingToCoords(ring)
	}
	return marshalGeoJSONGeometry("Polygon", rings)
}

// UnmarshalJSON implements json.Unmarshaler for GeometryPolygon.
func (poly *GeometryPolygon) UnmarshalJSON(data []byte) error {
	rings, err := unmarshalGeoJSONRings(data, "Polygon")
	if err != nil {
		return err
	}
	if len(rings) == 0 {
		return nil
	}
	poly.ExteriorRing = make([]GeometryPoint, len(rings[0]))
	for i, c := range rings[0] {
		poly.ExteriorRing[i] = GeometryPoint{X: c[0], Y: c[1]}
	}
	poly.InteriorRings = make([][]GeometryPoint, len(rings)-1)
	for ri, ring := range rings[1:] {
		poly.InteriorRings[ri] = make([]GeometryPoint, len(ring))
		for i, c := range ring {
			poly.InteriorRings[ri][i] = GeometryPoint{X: c[0], Y: c[1]}
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for GeographyMultiPoint.
func (mp GeographyMultiPoint) MarshalJSON() ([]byte, error) {
	coords := make([][2]float64, len(mp.Points))
	for i, p := range mp.Points {
		coords[i] = [2]float64{p.Longitude, p.Latitude}
	}
	return marshalGeoJSONGeometry("MultiPoint", coords)
}

// UnmarshalJSON implements json.Unmarshaler for GeographyMultiPoint.
func (mp *GeographyMultiPoint) UnmarshalJSON(data []byte) error {
	coords, err := unmarshalGeoJSONCoords(data, "MultiPoint")
	if err != nil {
		return err
	}
	mp.Points = make([]GeographyPoint, len(coords))
	for i, c := range coords {
		mp.Points[i] = GeographyPoint{Longitude: c[0], Latitude: c[1]}
	}
	return nil
}

// MarshalJSON implements json.Marshaler for GeometryMultiPoint.
func (mp GeometryMultiPoint) MarshalJSON() ([]byte, error) {
	coords := make([][2]float64, len(mp.Points))
	for i, p := range mp.Points {
		coords[i] = [2]float64{p.X, p.Y}
	}
	return marshalGeoJSONGeometry("MultiPoint", coords)
}

// UnmarshalJSON implements json.Unmarshaler for GeometryMultiPoint.
func (mp *GeometryMultiPoint) UnmarshalJSON(data []byte) error {
	coords, err := unmarshalGeoJSONCoords(data, "MultiPoint")
	if err != nil {
		return err
	}
	mp.Points = make([]GeometryPoint, len(coords))
	for i, c := range coords {
		mp.Points[i] = GeometryPoint{X: c[0], Y: c[1]}
	}
	return nil
}

// ---------------------------------------------------------------------------
// OData literal parsing (for deserializing filter expressions / properties)
// ---------------------------------------------------------------------------

// ParseGeographyPoint parses an OData geography point literal.
// Accepts: geography'SRID=4326;POINT(lon lat)' or POINT(lon lat)
func ParseGeographyPoint(s string) (GeographyPoint, error) {
	lon, lat, err := parseWKTPoint(s, "geography")
	if err != nil {
		return GeographyPoint{}, err
	}
	return GeographyPoint{Longitude: lon, Latitude: lat}, nil
}

// ParseGeometryPoint parses an OData geometry point literal.
// Accepts: geometry'SRID=0;POINT(x y)' or POINT(x y)
func ParseGeometryPoint(s string) (GeometryPoint, error) {
	x, y, err := parseWKTPoint(s, "geometry")
	if err != nil {
		return GeometryPoint{}, err
	}
	return GeometryPoint{X: x, Y: y}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// formatCoordTo appends a float coordinate to the buffer without allocating.
// It uses a stack-allocated scratch array to avoid heap allocation from AppendFloat.
func formatCoordTo(buf *bytes.Buffer, v float64) {
	var scratch [32]byte
	b := strconv.AppendFloat(scratch[:0], v, 'f', -1, 64)
	buf.Write(b)
}

// geoPointsToWKT converts a slice of GeographyPoint to WKT coordinate pairs.
func geoPointsToWKT(pts []GeographyPoint) string {
	if len(pts) == 0 {
		return ""
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	for i, p := range pts {
		if i > 0 {
			buf.WriteByte(',')
		}
		formatCoordTo(buf, p.Longitude)
		buf.WriteByte(' ')
		formatCoordTo(buf, p.Latitude)
	}
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// geomPointsToWKT converts a slice of GeometryPoint to WKT coordinate pairs.
func geomPointsToWKT(pts []GeometryPoint) string {
	if len(pts) == 0 {
		return ""
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	for i, p := range pts {
		if i > 0 {
			buf.WriteByte(',')
		}
		formatCoordTo(buf, p.X)
		buf.WriteByte(' ')
		formatCoordTo(buf, p.Y)
	}
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// geoRingToCoords converts a ring of GeographyPoints to [][2]float64.
func geoRingToCoords(ring []GeographyPoint) [][2]float64 {
	out := make([][2]float64, len(ring))
	for i, p := range ring {
		out[i] = [2]float64{p.Longitude, p.Latitude}
	}
	return out
}

// geomRingToCoords converts a ring of GeometryPoints to [][2]float64.
func geomRingToCoords(ring []GeometryPoint) [][2]float64 {
	out := make([][2]float64, len(ring))
	for i, p := range ring {
		out[i] = [2]float64{p.X, p.Y}
	}
	return out
}

// geoJSONPointGeometry is a compact GeoJSON Point representation.
type geoJSONPointGeometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// marshalGeoJSONPoint produces a GeoJSON Point bytes.
func marshalGeoJSONPoint(x, y float64) ([]byte, error) {
	return json.Marshal(geoJSONPointGeometry{Type: "Point", Coordinates: [2]float64{x, y}})
}

// unmarshalGeoJSONPoint decodes a GeoJSON Point bytes and returns x,y coordinates.
func unmarshalGeoJSONPoint(data []byte) (x, y float64, err error) {
	var g geoJSONPointGeometry
	if err = json.Unmarshal(data, &g); err != nil {
		return
	}
	if !strings.EqualFold(g.Type, "Point") {
		err = fmt.Errorf("traverse: expected GeoJSON type Point, got %q", g.Type)
		return
	}
	return g.Coordinates[0], g.Coordinates[1], nil
}

// marshalGeoJSONGeometry serializes a GeoJSON geometry with arbitrary coordinate structure.
func marshalGeoJSONGeometry(geomType string, coords interface{}) ([]byte, error) {
	return json.Marshal(struct {
		Type        string      `json:"type"`
		Coordinates interface{} `json:"coordinates"`
	}{Type: geomType, Coordinates: coords})
}

// unmarshalGeoJSONCoords deserializes a GeoJSON geometry's coordinates as [][2]float64.
// Used for LineString and MultiPoint types.
func unmarshalGeoJSONCoords(data []byte, expectedType string) ([][2]float64, error) {
	var raw struct {
		Type        string       `json:"type"`
		Coordinates [][2]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if !strings.EqualFold(raw.Type, expectedType) {
		return nil, fmt.Errorf("traverse: expected GeoJSON type %s, got %q", expectedType, raw.Type)
	}
	return raw.Coordinates, nil
}

// unmarshalGeoJSONRings deserializes a GeoJSON Polygon's ring coordinates.
func unmarshalGeoJSONRings(data []byte, expectedType string) ([][][2]float64, error) {
	var raw struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if !strings.EqualFold(raw.Type, expectedType) {
		return nil, fmt.Errorf("traverse: expected GeoJSON type %s, got %q", expectedType, raw.Type)
	}
	return raw.Coordinates, nil
}

// parseWKTPoint extracts lon/lat (or x/y) from an OData WKT point literal.
// Handles: geography'SRID=4326;POINT(lon lat)', POINT(lon lat), POINT (lon lat)
func parseWKTPoint(s, prefix string) (x, y float64, err error) {
	// Strip outer prefix literal if present: geography'...' or geometry'...'
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, prefix+"'") && strings.HasSuffix(s, "'") {
		s = s[len(prefix)+1 : len(s)-1]
	}
	// Strip optional SRID declaration
	if idx := strings.Index(strings.ToUpper(s), "POINT"); idx >= 0 {
		s = s[idx:]
	}
	// Parse POINT(x y) or POINT (x y)
	s = strings.TrimPrefix(strings.TrimSpace(s), "POINT")
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		return 0, 0, fmt.Errorf("traverse: invalid WKT point literal: %q", s)
	}
	inner := s[1 : len(s)-1]
	parts := strings.Fields(inner)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("traverse: invalid WKT point coordinates: %q", inner)
	}
	if x, err = strconv.ParseFloat(parts[0], 64); err != nil {
		return 0, 0, fmt.Errorf("traverse: invalid WKT coordinate %q: %w", parts[0], err)
	}
	if y, err = strconv.ParseFloat(parts[1], 64); err != nil {
		return 0, 0, fmt.Errorf("traverse: invalid WKT coordinate %q: %w", parts[1], err)
	}
	return x, y, nil
}
