package geofence

import "github.com/uwatu/uwatu-core/internal/models"

// IsInside checks if a point is inside a polygon using the Ray-Casting algorithm.
func IsInside(point models.Point, polygon models.Polygon) bool {
	// A polygon must have at least 3 points to form a shape
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1 // The last vertex

	// Loop through all edges of the polygon
	for i := 0; i < len(polygon); i++ {
		// Grab the two points that make up the current line segment
		p1 := polygon[i]
		p2 := polygon[j]

		// 1. Check if the point's Y is strictly between the line segment's Y values.
		isBetweenY := (p1.Lat > point.Lat) != (p2.Lat > point.Lat)

		if isBetweenY {
			// 2. Calculate the X intersection of the line segment at the point's Y value.
			intersectX := (p2.Lon-p1.Lon)*(point.Lat-p1.Lat)/(p2.Lat-p1.Lat) + p1.Lon

			// 3. If the point's X is to the left of the intersection, flip the boolean.
			if point.Lon < intersectX {
				inside = !inside
			}
		}

		j = i // Move to the next edge
	}

	return inside
}
