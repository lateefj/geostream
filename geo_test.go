package main

import (
	polyclip "github.com/akavel/polyclip-go"
	"testing"
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
	p := Polygon{polyclip.Polygon{{{1, 1}, {1, 3}, {3, 3}, {3, 1}}}}
	coords := p.Coordinates()
	if len(coords) != len(p.Polygon[0]) {
		t.Errorf("Expected %d coordinates but got %d", len(p.Polygon[0]), len(coords))
	}
	if !p.Overlaps(p) {
		t.Errorf("Expected p to overlap with itself")
	}
	p2 := Polygon{polyclip.Polygon{{{0, 0}, {0, 7}, {4, 7}, {4, 0}}}}
	if !p2.Overlaps(p) {
		t.Errorf("Expected p to overlap with p2")
	}
	if !p.Overlaps(p2) {
		t.Errorf("Expected p2 to overlap with p")
	}
	p3 := Polygon{polyclip.Polygon{{{10, 10}, {10, 17}, {14, 17}, {14, 10}}}}
	if p3.Overlaps(p) {
		t.Errorf("p3 should not have any overlap with p")
	}
	if p.Overlaps(p3) {
		t.Errorf("p should not have any overlap with p3")
	}
	//TODO: Add cross to verify that this will work with that type of polygon
}
