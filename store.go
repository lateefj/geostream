package main

import (
	"io"
	"fmt"
	"log"
	"time"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/lateefj/httpstream"
	//"github.com/araddon/httpstream"
	polyclip "github.com/akavel/polyclip-go"
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
var COORDS_INDEX = mgo.Index{Key: []string{"Coordinates.Coordinates:$2d"}} // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)
var PLACE_INDEX = mgo.Index{Key: []string{"Place.BoundingBox:$2d"}}        // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)

// Interface to allow for multiple implemntations for benchmarking different strateggies
// After single node implementation I hope to be able to create a multinode implementation 
type Geostore interface {
	Setup()                            // Allow for initialization mainly for setting up specific types of collections 
	Store(chan *httpstream.Tweet)      // Using channel if the twitter streaming API ever works
	Search(Polygon) []httpstream.Tweet // Search API
}

// Centralize this so can be cached potentially
func MongoSession(url string) (*mgo.Session, error) {
	s, err := mgo.Dial(url)
	if err != nil {
		log.Printf("Failed to get mongo session for %s error: %s", url, err)
		return nil, err
	}
	return s, nil
}

// Centralize collection retrieveal 
func MongoCollection(s *mgo.Session, db, name string) *mgo.Collection {
	c := s.DB(db).C(name)
	return c
}

var mgoSession *mgo.Session

func geoIndexSession() *mgo.Session {
	if mgoSession == nil {
		config, err := GetConfig()
		if err != nil {
			fmt.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
			panic(err)
			return nil
		}
		mgoSession, err = MongoSession(config.GeoIndexMongoUrl)
		if err != nil {
			panic(err)
		}
	}

	// Optional. Switch the session to a monotonic behavior.
	//session.SetMode(mgo.Monotonic, true)
	// Connection pooling 
	return mgoSession.Copy()
}

type SingleGeostore struct {
	DBName         string
	CollectionName string
}

func (sg *SingleGeostore) tweetCollection() *mgo.Collection {
	s := geoIndexSession()
	c := s.DB(sg.DBName).C(sg.CollectionName)
	info := &mgo.CollectionInfo{false, false, true, TWEET_COLLECTION_MAX_BYTES, TWEET_COLLECTION_MAX_DOCS}
	c.Create(info)
	return c
}
// Just make sure there is a geospacial index on the coordinates
func (sg *SingleGeostore) Setup() {
	c := sg.tweetCollection()
	c.EnsureIndex(COORDS_INDEX)
	c.EnsureIndex(PLACE_INDEX)
}

func (sg *SingleGeostore) Store(tweets chan *httpstream.Tweet) {
	c := sg.tweetCollection()
	for {
		if t, ok := <-tweets; ok {
			//log.Printf("Inserting tweet %s", t.Text)
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
// Search for a specific polygon
func (sg *SingleGeostore) Search(poly Polygon) []httpstream.Tweet {
	resp := make([]httpstream.Tweet, 0)
	c := sg.tweetCollection()
	coords := poly.Coordinates()
	q := bson.M{"coordinates": bson.M{"$geoWithin": bson.M{"$polygon": coords}}}
	c.Find(q).All(&resp)
	return resp
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

// Experimental distributed geostorage system
// Simply put the idea is that post can be distributed over many nodes but the data can still be queried as if it is continous space. This will implement the interface Geostore. It is a pretty straightforward divide and conqour implementation. By breaking up the lon/lat into pieces and then storing which server / dbname that piece is on we can then send the query on to that part. 
// Given a polygon we query the index which returns all the nodes we need to send queries to to get the data. Once the data is returned we just make sure there are no duplicates. For inserting the data we query which node the lat/lon is in and send it to that node/dbname.

// Way to associate a polygon with a host, datase and collection
type QuadrantLookup struct {
	Poly       Polygon
	Host       string
	DB         string
	Collection string
}

type DistributedGeostore struct {
	GeoIdxDBName   string
	GeoIdxCollName string
}

func (dg *DistributedGeostore) Configure(qls []QuadrantLookup) {
}
// The idea is to make sure the nodes have the quodrantes distributed evenly across them
func (dg *DistributedGeostore) Setup() {
	/*c := geoIndexCollection()
	c.EnsureIndex(COORDS_INDEX)*/
	//info := &mgo.CollectionInfo{false, false, true, TWEET_COLLECTION_MAX_BYTES, TWEET_COLLECTION_MAX_DOCS}
	//c.Create(info)
}

func (dg *DistributedGeostore) geoIndexCollection() *mgo.Collection {
	s := geoIndexSession()
	c := s.DB(dg.GeoIdxDBName).C(dg.GeoIdxCollName)
	return c
}

func (dg *DistributedGeostore) getCollection(host, db, name string) *mgo.Collection {
	s, err := MongoSession(host)
	if err != nil {
		fmt.Printf("Failed to get collection from host: %s db %s name %s error: %s", host, db, name, err)

	}
	return MongoCollection(s, db, name)
}

// Given a point find the collection associated with it
func (dg *DistributedGeostore) CollectionForPoint(p Point) *mgo.Collection {
	gc := dg.geoIndexCollection()
	pts := make([]polyclip.Point, 0)
	pts = append(pts, p.Point)
	pts = append(pts, p.Point)
	poly := Polygon{pts}
	q := bson.M{"Polygon": bson.M{"$geoWithin": bson.M{"$polygon": poly.Coordinates()}}}
	//q := bson.M{"Polygon": bson.M{"$geoWithin": bson.M{"$geometry": bson.M{"type": "Point", "coordinates": p.Coordinates()}}}}
	ql := &QuadrantLookup{}
	err := gc.Find(q).One(&ql)
	if err != nil {
		fmt.Printf("Failed to find polygon for point %f, %f error: %s", p.X, p.Y, err)
		return nil
	}
	return dg.getCollection(ql.Host, ql.DB, ql.Collection)
}

// Given a polygon get the collections that cross it
func (dg *DistributedGeostore) CollectionsForPoly(poly Polygon) []*mgo.Collection {
	cols := make([]*mgo.Collection, 0)
	qls := make([]*QuadrantLookup, 0)
	q := bson.M{"coordinates": bson.M{"$geoWithin": bson.M{"$polygon": poly.Coordinates()}}}
	gc := dg.geoIndexCollection()
	gc.Find(q).All(&qls)
	for _, ql := range qls {
		cols = append(cols, dg.getCollection(ql.Host, ql.DB, ql.Collection))
	}
	return cols
}

// Stores a stream of tweets
func (dg *DistributedGeostore) Store(tweets chan *httpstream.Tweet) {
	for {
		if t, ok := <-tweets; ok {
			// Create a point from the tweet
			p := NewPoint(t.Coordinates.Coordinates[0], t.Coordinates.Coordinates[1])
			// Caching the collection in CollectionForPoint probably makes sense!
			c := dg.CollectionForPoint(p)
			err := c.Insert(t)
			if err != nil && err != io.EOF {
				fmt.Printf("(Failed to insert) @%s: %s\n", t.User.ScreenName, t.Text)
				fmt.Printf("Error: %s\n", err)
			}
		} else {
			return
		}
	}
}
// Search for a specific polygon
// TODO: Use a set so there are no duplicates
func (dg *DistributedGeostore) Search(poly Polygon) []httpstream.Tweet {
	resp := make([]httpstream.Tweet, 0)
	cols := dg.CollectionsForPoly(poly)
	for _, c := range cols {
		coords := poly.Coordinates()
		q := bson.M{"coordinates": bson.M{"$geoWithin": bson.M{"$polygon": coords}}}
		tmp := make([]httpstream.Tweet, 0)
		c.Find(q).All(&tmp)
		resp = append(resp, tmp...)
	}
	return resp
}
