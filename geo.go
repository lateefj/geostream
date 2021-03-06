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
	polyclip.Polygon
}

// Generate list of coordinates for based on Points
func (p *Polygon) Coordinates() [][]float64 {
	coords := make([][]float64, 0)
	for _, c := range p.Polygon {
		for _, x := range c {
			pc := make([]float64, 2)
			pc[0] = x.X
			pc[1] = x.Y
			coords = append(coords, pc)
		}
	}
	return coords
}

// Thank to polyclip author for pointed out an obvious solution!!!
func (poly *Polygon) Overlaps(op Polygon) bool {
	result := poly.Polygon.Construct(polyclip.INTERSECTION, op.Polygon)
	return len(result) > 0
}
