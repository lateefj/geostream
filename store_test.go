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
func TestSingleGeostoreSearch(t *testing.T) {
	tChan := make(chan *httpstream.Tweet)
	sg := singleGeostoreInstance()
	go sg.Store(tChan)
	/*tl := Point{-180, 90}
	br := Point{180, -90}
	totalArea := BoundingBox{tl, br}
	*/
	stl := Point{122, 42}
	sbr := Point{125, 47}
	searchArea := BoundingBox{stl, sbr}
	initLat := 100 //-180
	initLon := 40  //-90
	allPoints := make([]Point, 0)
	expectedPoints := make([]Point, 0)
	var lat, lon float64
	//for x := 0; x <= 360; x++ {
	for x := 0; x <= 30; x++ {
		lat = float64(initLat + x)
		//for y := 0; y <= 180; y++ {
		for y := 0; y <= 10; y++ {
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
	fmt.Printf("Size of all points is %d\n", len(allPoints))
	fmt.Printf("Size of expectedPoints is %d\n", len(expectedPoints))
	c := sg.tweetCollection()
	time.Sleep(1 * time.Second)
	size, _ := c.Count()
	if size != len(allPoints) {
		t.Errorf("Expected %d but got %d tweets in the database", len(allPoints), size)
	}

	tws := sg.Search(searchArea)
	if len(tws) != len(expectedPoints) {
		t.Errorf("Expected to have %d tweets found but found %d", len(expectedPoints), len(tws))
	}

}
