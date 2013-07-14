package main

import (
	"os"
	"log"
	"strings"
	"encoding/json"
	//"github.com/araddon/httpstream"
	"github.com/lateefj/httpstream"
	oauth "github.com/araddon/goauth"
)

const (
	LOCATIONS = "-180,-90,180,90"
)

func GetTweets(stream chan *httpstream.Tweet) error {
	config, err := GetConfig()
	if err != nil {
		log.Printf("FAILED TO READ CONFIGURATION FILE!!! error: %s", err)
		return err
	}
	locs := strings.Split(LOCATIONS, ",")
	httpstream.OauthCon = &oauth.OAuthConsumer{
		Service:          "twitter",
		RequestTokenURL:  "http://twitter.com/oauth/request_token",
		AccessTokenURL:   "http://twitter.com/oauth/access_token",
		AuthorizationURL: "http://twitter.com/oauth/authorize",
		ConsumerKey:      config.ConsumerKey,
		ConsumerSecret:   config.ConsumerSecret,
		CallBackURL:      "oob",
		UserAgent:        "go/httpstream",
	}
	httpstream.SetLogger(log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile), "DEBUG")
	// make a go channel for sending from listener to processor
	// we buffer it, to help ensure we aren't backing up twitter or else they cut us off
	done := make(chan bool)

	at := oauth.AccessToken{Id: "",
		Token:    config.AccessTokenKey,
		Secret:   config.AccessTokenSecret,
		UserRef:  config.TwitterUser,
		Verifier: "",
		Service:  "twitter",
	}

	client := httpstream.NewOAuthClient(&at, httpstream.OnlyTweetsFilter(func(line []byte) {
		go func() {
			t := &httpstream.Tweet{}
			err := json.Unmarshal(line, t)
			if err != nil {
				log.Printf("Failed to unmarshal tweet %s", err)
			} else if t.Coordinates != nil || t.Place != nil { // Seem that this excludes some bounding boxes :(
				stream <- t
			} else {
				log.Printf("Tweet didn't have coordinates that I expected..\n%s\n", string(line))
			}
		}()

	}))
	err = client.Filter([]int64{}, []string{}, []string{}, locs, true, done)
	//err = client.Filter([]int64{}, []string{}, []string{"en"}, locs, false, done)
	return err
}
