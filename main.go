package main

import (
	"fmt"
	"log"
	"flag"
	//"time"
	"bytes"
	"strconv"
	"strings"
	"net/http"
	"io/ioutil"
	"text/template"
	"encoding/json"
	//"github.com/araddon/httpstream"
	"github.com/lateefj/httpstream"
	"code.google.com/p/gorilla/mux"
	"bitbucket.org/lateefj/httphacks"
)
// map.setCenter(new GLatLng(45.5236,122.6750), 13);
const (
	SERVER_NAME = "lhj.me (geostream) 0.1"
)

func searchHandler(w http.ResponseWriter, req *http.Request) {
	bs := req.FormValue("box")
	parts := strings.Split(bs, ",")
	f1, _ := strconv.ParseFloat(parts[0], 64)
	f2, _ := strconv.ParseFloat(parts[1], 64)
	f3, _ := strconv.ParseFloat(parts[2], 64)
	f4, _ := strconv.ParseFloat(parts[3], 64)
	bb := BoundingBox{NewPoint(f1, f2), NewPoint(f3, f4)}
	log.Printf("%f, %f : %f, %f", bb.BottomLeft.X, bb.BottomLeft.Y, bb.TopRight.X, bb.TopRight.Y)
	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	//twts := sg.SearchBox(bb, 100)
	twts := sg.FastSearchBox(bb, 100)
	log.Printf("Length of search results %d", len(twts))
	d, _ := json.Marshal(twts)
	w.Write(d)

}
func addHandler(w http.ResponseWriter, req *http.Request) {
	//p := req.FormValue("tweet")
	mf, _, err := req.FormFile("tweet")
	if err != nil {
		log.Printf("Failed to get tweet file ", err)
	}
	b, err := ioutil.ReadAll(mf)
	if err != nil {
		log.Printf("Failed to read all bytes from tweet file: ", err)
	}

	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	t := &httpstream.Tweet{}
	err = json.Unmarshal(b, t)
	if err != nil {
		log.Printf("ERROR UNMARSHALING TWEET need work here", err)
		return
	}
	c := make(chan *httpstream.Tweet)
	go sg.Store(c)
	c <- t
	close(c)

}
func InitRest(r *mux.Router) {
	r.HandleFunc("/search", httphacks.JSONWrap(searchHandler))
	r.HandleFunc("/add", httphacks.JSONWrap(addHandler))
}

type HeaderContext struct {
	Config
	Path string
}

func homeHandler(w http.ResponseWriter, req *http.Request) {
	config, err := GetConfig()
	if err != nil {
		log.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
		panic(err)
	}
	// Need to create a buffer so can parse
	buff := bytes.NewBufferString("")
	headerTempl := template.Must(template.ParseFiles("header.html"))
	headerTempl.Execute(buff, &HeaderContext{Config: *config, Path: ""})

	homeTempl := template.Must(template.ParseFiles("home.html"))
	homeTempl.Execute(w, buff.String())
}

// Way to export the header so it an be embedded in http://lhj.me/
func headerHandler(w http.ResponseWriter, req *http.Request) {
	config, err := GetConfig()
	if err != nil {
		log.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
		panic(err)
	}
	path := req.FormValue("path")
	// Need to create a buffer so can parse
	headerTempl := template.Must(template.ParseFiles("header.html"))
	headerTempl.Execute(w, &HeaderContext{Config: *config, Path: path})

}

// This just consumes tweets
// TODO: Need to handle failures and reconnect in some way!!!
func streamProducer() {
	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	c := make(chan *httpstream.Tweet)
	go sg.Store(c)
	GetTweets(c)
}

var startTwitterStream *bool = flag.Bool("stream", true, "Start the Twitter streaming in the web server ")
var port *int = flag.Int("port", 8000, "Port number to start geostream on")

func main() {
	flag.Parse()
	if *startTwitterStream { // Flag for having the streaming built into server
		log.Printf("Staring twitter streaming service in web server ...")
		go streamProducer()
	}
	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	sg.Setup()
	r := mux.NewRouter()
	s := httphacks.Server{SERVER_NAME, nil, r}
	fmt.Printf("Starting server: %s\n", s.Name)

	apiR := r.PathPrefix("/api/").Subrouter()
	InitRest(apiR)
	// Static files 
	//http.Handle("/js/", http.StripPrefix("/js", http.FileServer(http.Dir("js/"))))
	//http.Handle("/css/", http.StripPrefix("/css", http.FileServer(http.Dir("css/"))))
	//http.Handle("/dart/", http.StripPrefix("/dart", http.FileServer(http.Dir("dart/"))))

	// Templates 
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/header", headerHandler)

	// Hook up Gorilla MUX
	http.Handle("/", r)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
