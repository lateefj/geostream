package main

import (
	"io"
	"fmt"
	"log"
	"time"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/araddon/httpstream"
)

const (
	TWEET_DB                   = "tdb"
	TWEET_COLLECTION           = "tweets"
	TWEET_COLLECTION_MAX_BYTES = 8589934592 // 8 Gig
	TWEET_COLLECTION_MAX_DOCS  = 100000     // Max docs as per specs
)

type TweetletUser struct {
	Screenname string `json:"screen_name"`
}
type Tweetlet struct {
	Text        string                `json:"text"`
	Coordinates httpstream.Coordinate `json:"coordinates"`
	User        TweetletUser          `json:"user"`
}

// Index for coordinates
var COORDS_INDEX = mgo.Index{Key: []string{"coordinates:$2d"}} // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)

// Basic point
type Point struct {
	X, Y float64
}

func (p *Point) Coordinates() []float64 {
	c := make([]float64, 2)
	c[0] = p.X
	c[1] = p.Y
	return c
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
	Points []Point
}

// Generate list of coordinates for based on Points
func (p *Polygon) Coordinates() [][]float64 {
	c := make([][]float64, len(p.Points))
	for i, x := range p.Points {
		pc := make([]float64, 2)
		pc[0] = x.X
		pc[1] = x.Y
		c[i] = pc
	}
	return c
}

// Interface to allow for multiple implemntations for benchmarking different strateggies
// After single node implementation I hope to be able to create a multinode implementation 
type Geostore interface {
	Setup()                                    // Allow for initialization mainly for setting up specific types of collections 
	Store(chan *httpstream.Tweet)              // Using channel if the twitter streaming API ever works
	SearchBox(BoundingBox) []*httpstream.Tweet // Convenience API
	Search(Polygon) []*httpstream.Tweet        // Expected use API
}

type SingleGeostore struct {
	DBName         string
	CollectionName string
}

var mgoSession *mgo.Session

func (sg *SingleGeostore) geoIndexSession() *mgo.Session {
	if mgoSession == nil {
		config, err := GetConfig()
		if err != nil {
			fmt.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
			panic(err)
			return nil
		}
		mgoSession, err = mgo.Dial(config.GeoIndexMongoUrl)
		if err != nil {
			panic(err)
		}
	}

	// Optional. Switch the session to a monotonic behavior.
	//session.SetMode(mgo.Monotonic, true)

	return mgoSession.Copy()
}

func (sg *SingleGeostore) tweetCollection() *mgo.Collection {
	s := sg.geoIndexSession()
	c := s.DB(sg.DBName).C(sg.CollectionName)
	info := &mgo.CollectionInfo{false, false, true, TWEET_COLLECTION_MAX_BYTES, TWEET_COLLECTION_MAX_DOCS}
	c.Create(info)
	return c
}
// Just make sure there is a geospacial index on the coordinates
func (sg *SingleGeostore) Setup() {
	c := sg.tweetCollection()
	c.EnsureIndex(COORDS_INDEX)
}

func (sg *SingleGeostore) Store(tweets chan *httpstream.Tweet) {
	c := sg.tweetCollection()
	for {
		if t, ok := <-tweets; ok {
			err := c.Insert(t)
			if err != nil && err != io.EOF {
				fmt.Printf("(Failed to insert) @%s: %s\n", t.User.ScreenName, t.Text)
				fmt.Printf("Error: %s\n", err)
				//panic(err)
			}
		} else {
			return
		}
	}
}
// Experimental to improve performance
func (sg *SingleGeostore) FastSearchBox(bb BoundingBox, limit int) []Tweetlet {
	resp := make([]Tweetlet, 0)
	c := sg.tweetCollection()
	cords := make([][]float64, 2)
	cords[0] = bb.BottomLeft.Coordinates()
	cords[1] = bb.TopRight.Coordinates()
	q := bson.M{"coordinates": bson.M{"$geoWithin": bson.M{"$box": cords}}}
	s := time.Now()
	nq := c.Find(q)
	if limit > 0 {
		nq = nq.Limit(limit)
	}
	iter := nq.Iter()
	e := time.Now()
	t := e.Sub(s)
	log.Printf("MongoDB query took: %f", t.Seconds())
	s = time.Now()
	iter.All(&resp) // WHY SO SLOW???
	e = time.Now()
	t = e.Sub(s)
	iter.Close()
	log.Printf("Marshalling to go structures took: %f", t.Seconds())
	return resp
}
// Search a specific bounding box
// Performance bottleneck seems to be marhsaling the objects maybe could just return a bson.M to increase performance
func (sg *SingleGeostore) SearchBox(bb BoundingBox, limit int) []httpstream.Tweet {
	resp := make([]httpstream.Tweet, 0)
	c := sg.tweetCollection()
	cords := make([][]float64, 2)
	cords[0] = bb.BottomLeft.Coordinates()
	cords[1] = bb.TopRight.Coordinates()
	q := bson.M{"coordinates": bson.M{"$geoWithin": bson.M{"$box": cords}}}
	s := time.Now()
	nq := c.Find(q)
	if limit > 0 {
		nq = nq.Limit(limit)
	}
	e := time.Now()
	t := e.Sub(s)
	log.Printf("MongoDB query took: %f", t.Seconds())
	s = time.Now()
	nq.All(&resp) // Large json objects like httpstream.Tweet are a bit large if it was smaller would make a lot more sense
	e = time.Now()
	t = e.Sub(s)
	log.Printf("Marshalling to go structures took: %f", t.Seconds())
	return resp
}
// Search a specific bounding box
func (sg *SingleGeostore) Search(poly Polygon) []httpstream.Tweet {
	resp := make([]httpstream.Tweet, 0)
	c := sg.tweetCollection()
	coords := poly.Coordinates()
	q := bson.M{"coordinates": bson.M{"$geoWithin": bson.M{"$polygon": coords}}}
	c.Find(q).All(&resp)
	return resp
}
