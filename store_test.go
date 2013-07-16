package main

import (
	"encoding/json"
	"fmt"
	"github.com/lateefj/httpstream"
	"io/ioutil"
	"os"
	"testing"
	"time"
	//"github.com/araddon/httpstream"
	polyclip "github.com/akavel/polyclip-go"
)

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
	tc := make(chan *httpstream.Tweet, 0)
	gs.Setup()
	go gs.Store(tc)
	tl := readSampleTweets()
	if len(tl.Tweets) < 1 {
		t.Errorf("Failed to read sample tweets")
		return nil
	}

	fmt.Printf("Tweet read in are %d\n", len(tl.Tweets))
	for _, t := range tl.Tweets {
		if t.User != nil {
			//fmt.Printf("(%d) @%s: %s\n", i, t.User.ScreenName, t.Text)
			tc <- &t
		}
	}
	close(tc)

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
	dg := &DistributedGeostore{"test_" + TWEET_DB, "test_" + QUAD_COLLECTION, nil}
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

func buildTestData(gs Geostore, initLat, initLon int, searchArea BoundingBox) ([]Point, []Point) {
	tChan := make(chan *httpstream.Tweet)
	go gs.Store(tChan)

	allPoints := make([]Point, 0)
	expectedPoints := make([]Point, 0)
	var lat, lon float64
	lat = float64(initLat)
	lon = float64(initLon)
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
	i := 0
	for x := xinc; x <= 360; x = x + xinc {
		lat = float64(initLat + x)
		for y := yinc; y <= 180; y = y + yinc {
			lon = float64(initLon + y)
			poly := Polygon{polyclip.Polygon{{{preLat, preLon}, {lat, preLon}, {lat, lon}, {preLat, lon}}}}
			//fmt.Printf("%f, %f : %f, %f\n", preLat, preLon, lat, lon)
			collName := fmt.Sprintf("%s_%d", dg.GeoIdxCollName, i)
			q := QuadrantLookup{poly, config.GeoIndexMongoUrl, dg.GeoIdxDBName, collName}
			quads = append(quads, q)
			preLat = lat
			preLon = lon
			i++
		}
	}
	return quads
}

func TestDistributedGestoreSample(t *testing.T) {
	dg := distributedGeostoreInstance()        // Must be here or collection won't get dropped before these are added
	quads := buildTestQuads(dg, -180, -90, 40) // if the increment (last param) is set to small then it will overload dev laptop
	dg.Configure(quads)
	tl := storeSampleTweets(dg, t)
	// Wait for the inserts to complete
	time.Sleep(1 * time.Second)
	size := 0
	for _, q := range dg.Quads {
		qc := dg.getCollection(q.Host, q.DB, q.Collection)
		s, _ := qc.Count()
		size += s
		qc.DropCollection()
	}
	if size != len(tl.Tweets) {
		t.Errorf("\nExpected %d tweets but had %d\n", len(tl.Tweets), size)
	}
}
func TestDGSearch(t *testing.T) {
	dg := distributedGeostoreInstance()        // Must be here or collection won't get dropped before these are added
	quads := buildTestQuads(dg, -180, -90, 20) // if the increment (last param) is set to small then it will overload dev laptop
	dg.Configure(quads)
	stl := NewPoint(-90, -45)
	sbr := NewPoint(100, 50)
	searchArea := BoundingBox{stl, sbr}
	allPoints, expectedPoints := buildTestData(dg, -180, -90, searchArea)
	time.Sleep(1 * time.Second) // Let the collection storage catch its breath
	fmt.Printf("Size of all points is %d\n", len(allPoints))
	fmt.Printf("Size of expectedPoints is %d\n", len(expectedPoints))

	searchPoly := Polygon{polyclip.Polygon{{{-90, -45}, {-90, 50}, {100, 50}, {100, -45}}}}
	tws := dg.Search(searchPoly)
	if len(tws) != len(expectedPoints) {
		t.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
	// Do cleanup at the end
	for _, q := range dg.Quads {
		qc := dg.getCollection(q.Host, q.DB, q.Collection)
		qc.DropCollection()
	}
}

func TestSGSearch(t *testing.T) {
	sg := singleGeostoreInstance()
	stl := NewPoint(80, 35)
	sbr := NewPoint(100, 50)
	searchArea := BoundingBox{stl, sbr}
	allPoints, expectedPoints := buildTestData(sg, 90, 45, searchArea)
	if len(expectedPoints) < 1 {
		t.Errorf("There should have been some points generated in the bounding box!!")
	}
	fmt.Printf("Size of all points is %d\n", len(allPoints))
	fmt.Printf("Size of expectedPoints is %d\n", len(expectedPoints))
	time.Sleep(1 * time.Second) // Hack hack
	c := sg.tweetCollection()
	size, _ := c.Count()
	if size != len(allPoints) {
		t.Errorf("Expected %d but got %d tweets in the database", len(allPoints), size)
	}

	tws := sg.SearchBox(searchArea, 1000)
	if len(tws) != len(expectedPoints) {
		t.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
	searchPoly := Polygon{polyclip.Polygon{{{80, 35}, {80, 50}, {100, 50}, {100, 35}}}}
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
	_, expectedPoints := buildTestData(sg, -180, -90, searchArea)
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
	_, expectedPoints := buildTestData(sg, -180, -90, searchArea)
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
	_, expectedPoints := buildTestData(sg, -180, -90, searchArea)
	searchPoly := Polygon{polyclip.Polygon{{{122, 42}, {122, 47}, {125, 47}, {125, 42}}}}
	tws := sg.Search(searchPoly)
	if len(tws) != len(expectedPoints) {
		b.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}
}
