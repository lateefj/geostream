package main

import (
	"fmt"
	"labix.org/v2/mgo"
	//"labix.org/v2/mgo/bson"
	"github.com/araddon/httpstream"
)

const (
	TWEET_DB                   = "tdb"
	TWEET_COLLECTION           = "tweets"
	TWEET_COLLECTION_MAX_BYTES = 8589934592 // 8 Gig
	TWEET_COLLECTION_MAX_DOCS  = 100000     // Max docs as per specs
)

// Index for coordinates
var COORDS_INDEX = mgo.Index{Key: []string{"coordinates:$2d"}} // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)

// Basic definition of a bounding box for searching a square area would be better to be a polygon but for example purposes this is fine
type BoundingBox struct {
	Top    float32
	Left   float32
	Bottom float32
	Right  float32
}

// Standard format
func (bb *BoundingBox) String() string {
	return fmt.Sprintf("%f,%f,%f,%f", bb.Top, bb.Left, bb.Bottom, bb.Right)
}

// Interface to allow for multiple implemntations for benchmarking different strateggies
type Geostore interface {
	Setup()
	Store(chan *httpstream.Tweet)
	Search(BoundingBox) chan *httpstream.Tweet
}

type SingleGeostore struct {
	DBName         string
	CollectionName string
}

func (sg *SingleGeostore) geoIndexSession() *mgo.Session {

	config, err := GetConfig()
	if err != nil {
		fmt.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
		panic(err)
		return nil
	}
	session, err := mgo.Dial(config.GeoIndexMongoUrl)
	if err != nil {
		panic(err)
	}

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	return session
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
	for t := range tweets {
		err := c.Insert(t)
		if err != nil {
			fmt.Printf("(Failed to insert) @%s: %s\n", t.User.ScreenName, t.Text)
			//panic(err)
		}
	}

}
func (sg *SingleGeostore) Search(BoundingBox) chan *httpstream.Tweet {
	resp := make(chan *httpstream.Tweet)
	c := sg.tweetCollection()
	println(c.FullName)

	return resp
}

/*
func main() {
		session, err := mgo.Dial("server1.example.com,server2.example.com")
		if err != nil {
			panic(err)
		}
		defer session.Close()

		// Optional. Switch the session to a monotonic behavior.
		session.SetMode(mgo.Monotonic, true)

		c := session.DB("test").C("people")
		err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
			&Person{"Cla", "+55 53 8402 8510"})
		if err != nil {
			panic(err)
		}

		result := Person{}
		err = c.Find(bson.M{"name": "Ale"}).One(&result)
		if err != nil {
			panic(err)
		}

		fmt.Println("Phone:", result.Phone)
}
*/
