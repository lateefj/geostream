package main

import (
	"io"
	"fmt"
	"log"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/lateefj/httpstream"
	//"github.com/araddon/httpstream"
	//	polyclip "github.com/akavel/polyclip-go"
)

const (
	TWEET_DB                   = "tdb"
	TWEET_COLLECTION           = "tweets"
	QUAD_COLLECTION            = "quads"
	TWEET_COLLECTION_MAX_BYTES = 504857600 // 500 megs
	TWEET_COLLECTION_MAX_DOCS  = 100000    // Max docs as per specs

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
var COORDS_INDEX = mgo.Index{
	Key: []string{"$2d:coordinates.coordinates"},
} // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)
var PLACE_INDEX = mgo.Index{
	Key: []string{"$2d:bounding.coordinates}"},
} // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)

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
	s.SetMode(mgo.Monotonic, true)
	return s, nil
}

// Centralize collection retrieveal 
func MongoCollection(s *mgo.Session, db, name string) *mgo.Collection {
	c := s.DB(db).C(name)
	return c
}

var collectionCache map[string]*mgo.Collection
// Cached collection is needed so we doon't overload the databse with connection requests
func FastCollection(url, db, name string) (*mgo.Collection, error) {
	if collectionCache == nil {
		collectionCache = make(map[string]*mgo.Collection)
	}
	k := fmt.Sprintf("%s-%s-%s", url, db, name)
	if cc, ok := collectionCache[k]; ok {
		return cc, nil
	}
	s, err := MongoSession(url)
	//s.SetMode(mgo.Eventual, true)
	s.SetMode(mgo.Monotonic, true)
	if err != nil {
		log.Printf("Could not get mongo session url: %s", url)
		log.Printf("Error: %s", err)
		return nil, err
	}
	c := MongoCollection(s, db, name)
	collectionCache[k] = c
	return c, nil

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
	mgoSession.SetMode(mgo.Monotonic, true)
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
	q := bson.M{"coordinates.coordinates": bson.M{"$geoWithin": bson.M{"$polygon": coords}}}
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
	q := bson.M{"coordinates.coordinates": bson.M{"$geoWithin": bson.M{"$box": cords}}}
	nq := c.Find(q)
	if limit > 0 {
		nq = nq.Limit(limit)
	}
	nq.All(&resp)
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
	q := bson.M{"coordinates.coordinates": bson.M{"$geoWithin": bson.M{"$box": cords}}}
	nq := c.Find(q)
	if limit > 0 {
		nq = nq.Limit(limit)
	}
	nq.All(&resp) // Large json objects like httpstream.Tweet are a bit large if it was smaller would make a lot more sense
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

var QUAD_INDEX = mgo.Index{
	Key: []string{"$2d:poly.contour"},
} // Docs have a typo (http://godoc.org/labix.org/v2/mgo#Collection.EnsureIndexKey the field name is swapped with the 2d index type)


type DistributedGeostore struct {
	GeoIdxDBName   string
	GeoIdxCollName string
	Quads          []QuadrantLookup
}
// The idea is to make sure the nodes have the quodrantes distributed evenly across them
func (dg *DistributedGeostore) Setup() {
	c := dg.geoIndexCollection()
	err := c.EnsureIndex(QUAD_INDEX) // Make sure there is an index on the polygon
	if err != nil {
		log.Printf("Failed to ensure quad index: %s", err)
	}
}

// Store the associated quad
func (dg *DistributedGeostore) Configure(qls []QuadrantLookup) {
	dg.Quads = qls
	for _, q := range dg.Quads {
		qc := dg.getCollection(q.Host, q.DB, q.Collection)
		qc.EnsureIndex(COORDS_INDEX)
		qc.EnsureIndex(PLACE_INDEX)
	}
}

func (dg *DistributedGeostore) geoIndexCollection() *mgo.Collection {
	s := geoIndexSession()
	c := s.DB(dg.GeoIdxDBName).C(dg.GeoIdxCollName)
	return c
}

func (dg *DistributedGeostore) getCollection(host, db, name string) *mgo.Collection {
	c, _ := FastCollection(host, db, name) // TODO: Need to do something if error
	return c
}

// Given a point find the collection associated with it
// Originally this was going to be implemented using MongoDB however this is not supported by the database: https://jira.mongodb.org/browse/SERVER-2874
// Since the number of these is relatively small we can just do a for loop over them and find any that exist
func (dg *DistributedGeostore) CollectionForPoint(p Point) *mgo.Collection {
	for _, ql := range dg.Quads {
		if ql.Poly.Contains(p.Point) {
			return dg.getCollection(ql.Host, ql.DB, ql.Collection)
		}
	}
	return nil
}

// Given a polygon get the collections that cross it
// Originally was going to query Mongo server however this is should be much faster assuming the size of quadrants is not large
func (dg *DistributedGeostore) CollectionsForPoly(poly Polygon) []*mgo.Collection {
	cols := make([]*mgo.Collection, 0)
	for _, ql := range dg.Quads {
		if ql.Poly.BoundingBox().Overlaps(poly.BoundingBox()) {
			c := dg.getCollection(ql.Host, ql.DB, ql.Collection)
			cols = append(cols, c)
		}
	}
	return cols
}

// Stores a stream of tweets
func (dg *DistributedGeostore) Store(tweets chan *httpstream.Tweet) {
	run := true
	for run {
		if t, ok := <-tweets; ok {
			// Create a point from the tweet
			p := NewPoint(t.Coordinates.Coordinates[0], t.Coordinates.Coordinates[1])
			// Caching the collection in CollectionForPoint probably makes sense!
			c := dg.CollectionForPoint(p)
			if c == nil {
				log.Printf("ERROR: COULD NOT FIND COLLECTION to collection for point: %f, %f", p.X, p.Y)
				continue
			}
			err := c.Insert(t)

			if err != nil && err != io.EOF {
				log.Printf("(Failed to insert) @%s: %s\n", t.User.ScreenName, t.Text)
				log.Printf("Error: %s\n", err)
			}
			//log.Printf("(%s) @%s: %s\n", c.Name, t.User.ScreenName, t.Text)
		} else {
			run = false
		}
	}
}
// Search for a specific polygon
// TODO: Use a set so there are no duplicates
func (dg *DistributedGeostore) Search(poly Polygon) []httpstream.Tweet {
	resp := make([]httpstream.Tweet, 0)
	cols := dg.CollectionsForPoly(poly)
	// These request should be done concurrently example: https://github.com/lateefj/juggler
	for _, c := range cols {
		coords := poly.Coordinates()
		q := bson.M{"coordinates.coordinates": bson.M{"$geoWithin": bson.M{"$polygon": coords}}}
		tmp := make([]httpstream.Tweet, 0)
		c.Find(q).All(&tmp)
		//log.Printf("(%s): FOUND %d tweets in search [ [%f, %f], [%f, %f], [%f, %f], [%f, %f] ]", c.Name, len(tmp), poly.Contour[0].X, poly.Contour[0].Y, poly.Contour[1].X, poly.Contour[1].Y, poly.Contour[2].X, poly.Contour[2].Y, poly.Contour[3].X, poly.Contour[3].Y)
		resp = append(resp, tmp...)
	}
	return resp
}
