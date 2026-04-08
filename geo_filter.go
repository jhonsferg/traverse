package traverse

import (
	"bytes"
	"fmt"
)

// ---------------------------------------------------------------------------
// OData Geospatial filter functions (OData spec section 5.1.1.8)
//
// These builder functions produce OData $filter expressions for spatial queries:
//
//   geo.distance(property, point) - distance between a property and a point
//   geo.intersects(property, polygon) - whether a point property intersects a polygon
//   geo.length(linestring) - length of a line string
//
// All functions accept or return *QueryBuilder so they can be chained with
// the existing Filter() / FilterBy() / Where() pattern.
//
// All functions use bufferPool to avoid heap allocations for the string
// construction. The resulting expression string is short-lived and is owned
// by the query's filterExpr field.
// ---------------------------------------------------------------------------

// GeoDistanceFilter returns an OData geo.distance() filter expression string.
//
// The generated expression can be used directly with Filter():
//
//	client.From("Shops").Filter(traverse.GeoDistanceFilter("Location", traverse.GeographyPoint{13.408, 52.518}, "le", 1000))
//	// $filter=geo.distance(Location,geography'SRID=4326;POINT(13.408 52.518)') le 1000
//
// The operator must be one of: lt, le, gt, ge, eq, ne.
// The maxDistance is the threshold value (in the same unit as the CRS - typically metres).
func GeoDistanceFilter(property string, point GeographyPoint, operator string, maxDistance float64) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geo.distance(")
	buf.WriteString(property)
	buf.WriteByte(',')
	buf.WriteString(point.ODataLiteral())
	buf.WriteString(") ")
	buf.WriteString(operator)
	buf.WriteByte(' ')
	formatCoordTo(buf, maxDistance)
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// GeoDistanceFilterGeom returns an OData geo.distance() filter expression for geometry coordinates.
func GeoDistanceFilterGeom(property string, point GeometryPoint, operator string, maxDistance float64) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geo.distance(")
	buf.WriteString(property)
	buf.WriteByte(',')
	buf.WriteString(point.ODataLiteral())
	buf.WriteString(") ")
	buf.WriteString(operator)
	buf.WriteByte(' ')
	formatCoordTo(buf, maxDistance)
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// GeoIntersectsFilter returns an OData geo.intersects() filter expression string.
//
// geo.intersects(property, polygon) returns true when the point property
// falls within the given polygon boundary.
//
//	client.From("POI").Filter(traverse.GeoIntersectsFilter("Coordinates", myPolygon))
//	// $filter=geo.intersects(Coordinates,geography'SRID=4326;POLYGON(...)')
func GeoIntersectsFilter(property string, polygon GeographyPolygon) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geo.intersects(")
	buf.WriteString(property)
	buf.WriteByte(',')
	buf.WriteString(polygon.ODataLiteral())
	buf.WriteByte(')')
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// GeoIntersectsFilterGeom returns an OData geo.intersects() filter expression for geometry.
func GeoIntersectsFilterGeom(property string, polygon GeometryPolygon) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geo.intersects(")
	buf.WriteString(property)
	buf.WriteByte(',')
	buf.WriteString(polygon.ODataLiteral())
	buf.WriteByte(')')
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// GeoLengthFilter returns an OData geo.length() filter expression string.
//
// geo.length(linestring_property) returns the length of the LineString in
// the same unit as the coordinate reference system.
//
//	client.From("Routes").Filter(traverse.GeoLengthFilter("Path", "le", 50000))
//	// $filter=geo.length(Path) le 50000
func GeoLengthFilter(property string, operator string, threshold float64) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("geo.length(")
	buf.WriteString(property)
	buf.WriteString(") ")
	buf.WriteString(operator)
	buf.WriteByte(' ')
	formatCoordTo(buf, threshold)
	s := buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	return s
}

// ---------------------------------------------------------------------------
// QueryBuilder integration - GeoDistance, GeoIntersects, GeoLength methods
// ---------------------------------------------------------------------------

// GeoDistance adds a geo.distance() filter to the query.
//
// GeoDistance filters entities whose spatial property is within (or beyond)
// the specified distance from the given geographic point.
//
// Example:
//
//	client.From("Restaurants").
//	    GeoDistance("Location", traverse.GeographyPoint{Longitude: 13.408, Latitude: 52.518}, "le", 500)
//	// $filter=geo.distance(Location,geography'SRID=4326;POINT(13.408 52.518)') le 500
func (q *QueryBuilder) GeoDistance(property string, point GeographyPoint, operator string, distance float64) *QueryBuilder {
	if err := validateGeoOperator(operator); err != nil {
		q.lastError = err
		return q
	}
	expr := GeoDistanceFilter(property, point, operator, distance)
	return q.appendGeoFilter(expr)
}

// GeoDistanceGeom adds a geo.distance() filter for geometry coordinates.
func (q *QueryBuilder) GeoDistanceGeom(property string, point GeometryPoint, operator string, distance float64) *QueryBuilder {
	if err := validateGeoOperator(operator); err != nil {
		q.lastError = err
		return q
	}
	expr := GeoDistanceFilterGeom(property, point, operator, distance)
	return q.appendGeoFilter(expr)
}

// GeoIntersects adds a geo.intersects() filter to the query.
//
// GeoIntersects filters entities where the spatial point property lies within
// the given polygon.
//
// Example:
//
//	poly := traverse.GeographyPolygon{ExteriorRing: []traverse.GeographyPoint{...}}
//	client.From("POI").GeoIntersects("Coordinates", poly)
//	// $filter=geo.intersects(Coordinates,geography'SRID=4326;POLYGON(...)')
func (q *QueryBuilder) GeoIntersects(property string, polygon GeographyPolygon) *QueryBuilder {
	expr := GeoIntersectsFilter(property, polygon)
	return q.appendGeoFilter(expr)
}

// GeoIntersectsGeom adds a geo.intersects() filter for geometry coordinates.
func (q *QueryBuilder) GeoIntersectsGeom(property string, polygon GeometryPolygon) *QueryBuilder {
	expr := GeoIntersectsFilterGeom(property, polygon)
	return q.appendGeoFilter(expr)
}

// GeoLength adds a geo.length() filter to the query.
//
// GeoLength filters entities where the length of the spatial LineString property
// satisfies the comparison.
//
// Example:
//
//	client.From("Routes").GeoLength("Path", "le", 50000)
//	// $filter=geo.length(Path) le 50000
func (q *QueryBuilder) GeoLength(property string, operator string, threshold float64) *QueryBuilder {
	if err := validateGeoOperator(operator); err != nil {
		q.lastError = err
		return q
	}
	expr := GeoLengthFilter(property, operator, threshold)
	return q.appendGeoFilter(expr)
}

// appendGeoFilter appends a geo expression to the existing filter using 'and',
// or sets it as the sole filter when no prior filter is present.
func (q *QueryBuilder) appendGeoFilter(expr string) *QueryBuilder {
	if q.filterExpr == "" {
		q.filterExpr = expr
	} else {
		buf := bufferPool.Get().(*bytes.Buffer)
		buf.WriteString(q.filterExpr)
		buf.WriteString(" and ")
		buf.WriteString(expr)
		q.filterExpr = buf.String()
		buf.Reset()
		bufferPool.Put(buf)
	}
	return q
}

// validateGeoOperator returns an error for disallowed comparison operators.
func validateGeoOperator(op string) error {
	switch op {
	case "lt", "le", "gt", "ge", "eq", "ne":
		return nil
	default:
		return fmt.Errorf("traverse: invalid geo filter operator %q: must be lt, le, gt, ge, eq, or ne", op)
	}
}
