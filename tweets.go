package main

import (
//"strings"
//	"os"
//	"log"
//	"github.com/araddon/httpstream"
//	oauth "github.com/akrennmair/goauth"
)

const (
	LOCATIONS = "-180,-90,180,90"
)
/*
func main() {
	config, err := GetConfig()
	if err != nil {
		println("FAILED TO READ CONFIGURATION FILE!!!")
		println(err)
		return
	}
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
	stream := make(chan []byte, 1000)
	done := make(chan bool)

	at := oauth.AccessToken{Id: "",
		Token:    config.AccessTokenKey,
		Secret:   config.AccessTokenSecret,
		UserRef:  config.TwitterUser,
		Verifier: "",
		Service:  "twitter",
	}
	client := httpstream.NewOAuthClient(&at, httpstream.OnlyTweetsFilter(func(line []byte) {
		stream <- line
		// although you can do heavy lifting here, it means you are doing all
		// your work in the same thread as the http streaming/listener
		// by using a go channel, you can send the work to a 
		// different thread/goroutine
	}))
	// TODO: Add filter instead of sample with LOCATIONS however need to wait until OAuth is working with twitter
	err = client.Sample(done)
	if err != nil {
		httpstream.Log(httpstream.ERROR, err.Error())
	} else {

		go func() {
			// while this could be in a different "thread(s)"
			ct := 0
			for tw := range stream {
				println(string(tw))
				// heavy lifting
				ct++
				if ct > 10 {
					done <- true
				}
			}
		}()
		_ = <-done
	}

}
*/
