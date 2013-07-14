package main

import (
	"testing"
	polyclip "github.com/akavel/polyclip-go"
)

func TestBoundingBox(t *testing.T) {

	bl := NewPoint(0, 0)
	tr := NewPoint(10, 10)
	bb := BoundingBox{bl, tr}
	p := NewPoint(1, 1)
	if !bb.Contains(p) {
		t.Errorf("Expected point 1, 1 to be in 10x10 bounding box :(")
	}
	p = NewPoint(11, 1)
	if bb.Contains(p) {
		t.Errorf("Expected point 11, 1 to be outside of 10x10 bounding box :(")
	}
	p = NewPoint(1, 11)
	if bb.Contains(p) {
		t.Errorf("Expected point 1, 11 to be outside of 10x10 bounding box :(")
	}
	p = NewPoint(11, 11)
	if bb.Contains(p) {
		t.Errorf("Expected point 11, 11 to be outside of 10x10 bounding box :(")
	}
	p = NewPoint(2, 2)
	if !bb.Contains(p) {
		t.Errorf("Expected point 2, 2 to be in 10x10 bounding box :(")
	}
	p = NewPoint(0, 2)
	if !bb.Contains(p) {
		t.Errorf("Expected point 0, 2 to be in 10x10 bounding box :(")
	}
}

func TestPolygon(t *testing.T) {
	pts := make([]polyclip.Point, 0)
	pts = append(pts, NewPoint(1, 1).Point)
	pts = append(pts, NewPoint(1, 3).Point)
	pts = append(pts, NewPoint(3, 1).Point)
	pts = append(pts, NewPoint(3, 3).Point)
	p := Polygon{pts}
	coords := p.Coordinates()
	if len(coords) != len(pts) {
		t.Errorf("Expected %d coordinates but got %d", len(pts), len(coords))
	}
}
