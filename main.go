package main

import (
	"log"
	"strconv"
	"strings"
	"net/http"
	"io/ioutil"
	"text/template"
	"encoding/json"
	"github.com/araddon/httpstream"
	"code.google.com/p/gorilla/mux"
	"bitbucket.org/lateefj/httphacks"
)
// map.setCenter(new GLatLng(45.5236,122.6750), 13);
const (
	SERVER_NAME = "lhj.me (geostream) 0.1"
)

func searchHandler(w http.ResponseWriter, req *http.Request) {
	// TODO: IMPLEMENT ME!!!
	b, _ := ioutil.ReadAll(req.Body)
	println(string(b))
	bs := req.FormValue("box")
	parts := strings.Split(bs, ",")
	println(parts)
	f1, _ := strconv.ParseFloat(parts[0], 64)
	f2, _ := strconv.ParseFloat(parts[1], 64)
	f3, _ := strconv.ParseFloat(parts[2], 64)
	f4, _ := strconv.ParseFloat(parts[3], 64)
	bb := BoundingBox{Point{f1, f2}, Point{f3, f4}}
	log.Printf("%f, %f : %f, %f", bb.BottomLeft.X, bb.BottomLeft.Y, bb.TopRight.X, bb.TopRight.Y)
	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	twts := sg.SearchBox(bb, 100)
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

func homeHandler(w http.ResponseWriter, req *http.Request) {
	config, err := GetConfig()
	if err != nil {
		log.Printf("FAILED TO READ CONFIGURATION FILE!!! %s", err)
		panic(err)
	}
	homeTempl := template.Must(template.ParseFiles("home.html"))
	homeTempl.Execute(w, config.GoogleMapsAPI)
}

func main() {
	sg := &SingleGeostore{TWEET_DB, TWEET_COLLECTION}
	sg.Setup()
	r := mux.NewRouter()
	s := httphacks.Server{SERVER_NAME, nil, r}
	println(s.Name)

	apiR := r.PathPrefix("/api/").Subrouter()
	InitRest(apiR)
	// Static files 
	http.Handle("/js/", http.StripPrefix("/js", http.FileServer(http.Dir("js/"))))
	http.Handle("/css/", http.StripPrefix("/css", http.FileServer(http.Dir("css/"))))
	http.Handle("/dart/", http.StripPrefix("/dart", http.FileServer(http.Dir("dart/"))))

	// Templates 
	r.HandleFunc("/", homeHandler)

	// Hook up Gorilla MUX
	http.Handle("/", r)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
