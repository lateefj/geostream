package main

import (
	"os"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
	"testing"
	"github.com/lateefj/httpstream"
	//"github.com/araddon/httpstream"
	polyclip "github.com/akavel/polyclip-go"
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
	for _, t := range tl.Tweets {
		if t.User != nil {
			//fmt.Printf("(%d) @%s: %s\n", i, t.User.ScreenName, t.Text)
			tc <- &t
		}
	}

	sg.Setup()
	go sg.Store(tc)

	time.Sleep(1 * time.Second)
	close(tc)
}

func storeSampleTweets(gs Geostore, t *testing.T) *TweetList {
	tl := readSampleTweets()
	if len(tl.Tweets) < 1 {
		t.Errorf("Failed to read sample tweets")
		return nil
	}

	tc := make(chan *httpstream.Tweet, len(tl.Tweets))
	fmt.Printf("Tweet read in are %d\n", len(tl.Tweets))
	for _, t := range tl.Tweets {
		if t.User != nil {
			//fmt.Printf("(%d) @%s: %s\n", i, t.User.ScreenName, t.Text)
			tc <- &t
		}
	}

	gs.Setup()
	go gs.Store(tc)
	return tl

}

func singleGeostoreInstance() *SingleGeostore {
	sg := &SingleGeostore{"test_" + TWEET_DB, "test_" + TWEET_COLLECTION}
	c := sg.tweetCollection()
	// TODO: Not sure I like this but namespace should be good enough to prevent accidental production delete
	c.DropCollection()
	return sg
}
func distributedGeostoreInstance() *DistributedGeostore {
	dg := &DistributedGeostore{"test_" + TWEET_DB, "test_" + TWEET_COLLECTION}
	c := dg.geoIndexCollection()
	// TODO: Not sure I like this but namespace should be good enough to prevent accidental production delete
	c.DropCollection()
	dg.Setup()
	return dg
}

func TestSingleGeostoreSample(t *testing.T) {
	sg := singleGeostoreInstance() // Must be here or collection won't get dropped before these are added
	tl := storeSampleTweets(sg, t)
	c := sg.tweetCollection()
	// Wait for the inserts to complete
	time.Sleep(1 * time.Second)
	size, _ := c.Count()
	if size != len(tl.Tweets) {
		t.Errorf("\nExpected %d tweets but only had %d\n", tl.Tweets, size)
	}
}

func buildTestQuads(dg *DistributedGeostore, initLat, initLon, xinc int) []QuadrantLookup {
	config, err := GetConfig()
	if err != nil {
		fmt.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
		panic(err)
		return nil
	}
	quads := make([]QuadrantLookup, 0)
	yinc := xinc / 2

	var lat, lon float64
	preLat := float64(initLat)
	preLon := float64(initLon)
	for x := 0; x <= 360; x = x + xinc {
		lat = float64(initLat + x)
		for y := 0; y <= 180; y = y + yinc {
			lon = float64(initLon + y)
			pts := make([]polyclip.Point, 0)
			pts = append(pts, NewPoint(preLat, preLon).Point)
			pts = append(pts, NewPoint(lat, preLon).Point)
			pts = append(pts, NewPoint(lat, lon).Point)
			pts = append(pts, NewPoint(preLat, lon).Point)
			poly := Polygon{pts}
			q := QuadrantLookup{poly, config.GeoIndexMongoUrl, dg.GeoIdxDBName, dg.GeoIdxCollName}
			quads = append(quads, q)
		}
	}
	return quads
}

func TestDistributedGestoreSample(t *testing.T) {
	dg := distributedGeostoreInstance() // Must be here or collection won't get dropped before these are added
	tl := storeSampleTweets(dg, t)
	c := dg.geoIndexCollection()
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
			p := NewPoint(lat, lon)
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
	stl := NewPoint(122, 42)
	sbr := NewPoint(125, 47)
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
	pts := make([]polyclip.Point, 0)
	pts = append(pts, NewPoint(122, 42).Point)
	pts = append(pts, NewPoint(122, 47).Point)
	pts = append(pts, NewPoint(125, 47).Point)
	pts = append(pts, NewPoint(125, 42).Point)
	searchPoly := Polygon{pts}
	tws = sg.Search(searchPoly)
	if len(tws) != len(expectedPoints) {
		t.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}

func Benchmark_SingleBoxSearch(b *testing.B) {
	sg := singleGeostoreInstance()
	stl := NewPoint(122, 42)
	sbr := NewPoint(125, 47)
	searchArea := BoundingBox{stl, sbr}
	_, expectedPoints := buildTestData(-180, -90, searchArea)
	tws := sg.SearchBox(searchArea, 0)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}

func Benchmark_SingleBoxFastSearch(b *testing.B) {
	sg := singleGeostoreInstance()
	stl := NewPoint(122, 42)
	sbr := NewPoint(125, 47)
	searchArea := BoundingBox{stl, sbr}
	_, expectedPoints := buildTestData(-180, -90, searchArea)
	tws := sg.FastSearchBox(searchArea, 0)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}

func Benchmark_SinglePolygonSearch(b *testing.B) {
	sg := singleGeostoreInstance()
	stl := NewPoint(122, 42)
	sbr := NewPoint(125, 47)
	searchArea := BoundingBox{stl, sbr}
	_, expectedPoints := buildTestData(-180, -90, searchArea)
	pts := make([]polyclip.Point, 0)
	pts = append(pts, NewPoint(122, 42).Point)
	pts = append(pts, NewPoint(122, 47).Point)
	pts = append(pts, NewPoint(125, 47).Point)
	pts = append(pts, NewPoint(125, 42).Point)
	searchPoly := Polygon{pts}
	tws := sg.Search(searchPoly)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}
