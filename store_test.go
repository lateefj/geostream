package main

import (
	"os"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
	"testing"
	"github.com/araddon/httpstream"
)

type TweetList struct {
	Tweets []httpstream.Tweet
}

func readSampleTweets() *TweetList {
	f, _ := os.Open("sample.json")
	b, _ := ioutil.ReadAll(f)

	tl := &TweetList{make([]httpstream.Tweet, 0)}
	err := json.Unmarshal(b, &tl.Tweets)
	if err != nil {
		fmt.Printf("FAILED to unmarshal tweet list error: %s", err)
	}
	return tl

}

func TestLoadSample(t *testing.T) {
	tl := readSampleTweets()
	if len(tl.Tweets) < 1 {
		t.Errorf("Failed to read sample tweets")
		return
	}

	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	tc := make(chan *httpstream.Tweet, len(tl.Tweets))
	fmt.Printf("Tweet read in are %d\n", len(tl.Tweets))
	for i, t := range tl.Tweets {
		if t.User != nil {
			fmt.Printf("(%d) @%s: %s\n", i, t.User.ScreenName, t.Text)
			tc <- &t
		}
	}

	sg.Setup()
	go sg.Store(tc)

	time.Sleep(1 * time.Second)
	close(tc)
}

func singleGeostoreInstance() *SingleGeostore {
	sg := &SingleGeostore{"test_" + TWEET_DB, "test_" + TWEET_COLLECTION}
	c := sg.tweetCollection()
	// TODO: Not sure I like this but namespace should be good enough to prevent accidental production delete
	c.DropCollection()
	return sg
}
func TestSingleGeostore(t *testing.T) {
	tl := readSampleTweets()
	if len(tl.Tweets) < 1 {
		t.Errorf("Failed to read sample tweets")
		return
	}

	sg := singleGeostoreInstance() // Must be here or collection won't get dropped before these are added
	tc := make(chan *httpstream.Tweet, len(tl.Tweets))
	fmt.Printf("Tweet read in are %d\n", len(tl.Tweets))
	for i, t := range tl.Tweets {
		if t.User != nil {
			fmt.Printf("(%d) @%s: %s\n", i, t.User.ScreenName, t.Text)
			tc <- &t
		}
	}

	sg.Setup()
	go sg.Store(tc)
	c := sg.tweetCollection()
	// Wait for the inserts to complete
	time.Sleep(1 * time.Second)
	size, _ := c.Count()
	if size != len(tl.Tweets) {
		t.Errorf("\nExpected %d tweets but only had %d\n", tl.Tweets, size)
	}
}

func buildTweet(tChan chan *httpstream.Tweet, p Point) {
	u := &httpstream.User{
		Name:       fmt.Sprintf("%f %f", p.X, p.Y),
		ScreenName: fmt.Sprintf("%f,%f", p.X, p.Y),
	}

	c := &httpstream.Coordinate{p.Coordinates(), "point"}
	t := &httpstream.Tweet{
		User:        u,
		Text:        fmt.Sprintf("Tweeting location %f, %f", p.X, p.Y),
		Coordinates: c,
	}

	tChan <- t

}

func TestBoundingBox(t *testing.T) {

	bl := Point{0, 0}
	tr := Point{10, 10}
	bb := BoundingBox{bl, tr}
	p := Point{1, 1}
	if !bb.Contains(p) {
		t.Errorf("Expected point 1, 1 to be in 10x10 bounding box :(")
	}
	p = Point{11, 1}
	if bb.Contains(p) {
		t.Errorf("Expected point 11, 1 to be outside of 10x10 bounding box :(")
	}
	p = Point{1, 11}
	if bb.Contains(p) {
		t.Errorf("Expected point 1, 11 to be outside of 10x10 bounding box :(")
	}
	p = Point{11, 11}
	if bb.Contains(p) {
		t.Errorf("Expected point 11, 11 to be outside of 10x10 bounding box :(")
	}
	p = Point{2, 2}
	if !bb.Contains(p) {
		t.Errorf("Expected point 2, 2 to be in 10x10 bounding box :(")
	}
	p = Point{0, 2}
	if !bb.Contains(p) {
		t.Errorf("Expected point 0, 2 to be in 10x10 bounding box :(")
	}
}

func TestPolygon(t *testing.T) {
	pts := make([]Point, 0)
	pts = append(pts, Point{1, 1})
	pts = append(pts, Point{1, 3})
	pts = append(pts, Point{3, 1})
	pts = append(pts, Point{3, 3})
	p := Polygon{pts}
	coords := p.Coordinates()
	if len(coords) != len(pts) {
		t.Errorf("Expected %d coordinates but got %d", len(pts), len(coords))
	}
}
func buildTestData(initLat, initLon int, searchArea BoundingBox) ([]Point, []Point) {
	tChan := make(chan *httpstream.Tweet)
	sg := singleGeostoreInstance()
	go sg.Store(tChan)

	allPoints := make([]Point, 0)
	expectedPoints := make([]Point, 0)
	var lat, lon float64
	for x := 0; x <= 360; x++ {
		lat = float64(initLat + x)
		for y := 0; y <= 180; y++ {
			lon = float64(initLon + y)
			p := Point{lat, lon}
			allPoints = append(allPoints, p)
			if searchArea.Contains(p) {
				expectedPoints = append(expectedPoints, p)
			}
			buildTweet(tChan, p)
		}
	}
	close(tChan)
	return allPoints, expectedPoints
}
func TestSGSearch(t *testing.T) {
	sg := singleGeostoreInstance()
	stl := Point{122, 42}
	sbr := Point{125, 47}
	searchArea := BoundingBox{stl, sbr}
	allPoints, expectedPoints := buildTestData(100, 40, searchArea)
	fmt.Printf("Size of all points is %d\n", len(allPoints))
	fmt.Printf("Size of expectedPoints is %d\n", len(expectedPoints))
	c := sg.tweetCollection()
	time.Sleep(1 * time.Second) // Hack hack 
	size, _ := c.Count()
	if size != len(allPoints) {
		t.Errorf("Expected %d but got %d tweets in the database", len(allPoints), size)
	}

	tws := sg.SearchBox(searchArea, 1000)
	if len(tws) != len(expectedPoints) {
		t.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
	pts := make([]Point, 0)
	pts = append(pts, Point{122, 42})
	pts = append(pts, Point{122, 47})
	pts = append(pts, Point{125, 47})
	pts = append(pts, Point{125, 42})
	searchPoly := Polygon{pts}
	tws = sg.Search(searchPoly)
	if len(tws) != len(expectedPoints) {
		t.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}

func Benchmark_SingleBoxSearch(b *testing.B) {
	sg := singleGeostoreInstance()
	stl := Point{122, 42}
	sbr := Point{125, 47}
	searchArea := BoundingBox{stl, sbr}
	_, expectedPoints := buildTestData(-180, -90, searchArea)
	tws := sg.SearchBox(searchArea, 0)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}

func Benchmark_SingleBoxFastSearch(b *testing.B) {
	sg := singleGeostoreInstance()
	stl := Point{122, 42}
	sbr := Point{125, 47}
	searchArea := BoundingBox{stl, sbr}
	_, expectedPoints := buildTestData(-180, -90, searchArea)
	tws := sg.FastSearchBox(searchArea, 0)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}

func Benchmark_SinglePolygonSearch(b *testing.B) {
	sg := singleGeostoreInstance()
	stl := Point{122, 42}
	sbr := Point{125, 47}
	searchArea := BoundingBox{stl, sbr}
	_, expectedPoints := buildTestData(-180, -90, searchArea)
	pts := make([]Point, 0)
	pts = append(pts, Point{122, 42})
	pts = append(pts, Point{122, 47})
	pts = append(pts, Point{125, 47})
	pts = append(pts, Point{125, 42})
	searchPoly := Polygon{pts}
	tws := sg.Search(searchPoly)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}
