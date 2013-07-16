// This is a wrapper around github.com/akavel/polyclip-go mainly to support helper functions. Would be great to depricate these functions soon.

package main

import (
	polyclip "github.com/akavel/polyclip-go"
)

// Basic point
type Point struct {
	polyclip.Point
}

func (p *Point) Coordinates() []float64 {
	c := make([]float64, 2)
	c[0] = p.X
	c[1] = p.Y
	return c
}

func NewPoint(x, y float64) Point {
	return Point{polyclip.Point{x, y}}
}

// Basic definition of a bounding box for searching a square area would be better to be a polygon but for example purposes this is fine
type BoundingBox struct {
	BottomLeft Point
	TopRight   Point
}

// Pretty standard checking to see if the point is in the bounding box
func (bb *BoundingBox) Contains(p Point) bool {
	if (p.X >= bb.BottomLeft.X && p.X <= bb.TopRight.X) && (p.Y >= bb.BottomLeft.Y && p.Y <= bb.TopRight.Y) {
		return true
	}
	return false
}

type Polygon struct {
	polyclip.Contour
}

// Generate list of coordinates for based on Points
func (p *Polygon) Coordinates() [][]float64 {
	c := make([][]float64, len(p.Contour))
	for i, x := range p.Contour {
		pc := make([]float64, 2)
		pc[0] = x.X
		pc[1] = x.Y
		c[i] = pc
	}
	return c
}

// Simple idea is that if any of the passed in points are contained by this polygon then they overlap. This is such a big hack that there is a better way to do this: TODO: FIX THIS HACK 
func (poly *Polygon) Overlap(op Polygon) bool {
	// If any points are contained by the polygon passed in
	for _, p := range op.Contour {
		if poly.Contains(p) {
			return true
		}
	}
	// Reverse check of polygons
	for _, p := range poly.Contour {
		if op.Contains(p) {
			return true
		}
	}
	// If any of the points are equal then they overlap
	for _, x := range op.Contour {
		for _, y := range poly.Contour {
			if x.Equals(y) {
				return true
			}
		}
	}
	return false
}
