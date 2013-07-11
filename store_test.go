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

func TestSingleGeostore(t *testing.T) {
	tl := readSampleTweets()
	if len(tl.Tweets) < 1 {
		t.Errorf("Failed to read sample tweets")
		return
	}

	for _, t := range tl.Tweets {
		fmt.Printf("@%s: %s\n", t.User.ScreenName, t.Text)
	}
	sg := &SingleGeostore{"test_" + TWEET_DB, "test_" + TWEET_COLLECTION}

	c := sg.tweetCollection()
	// XXX: Testing only in dev mode probably need to set env config file but this is just sample application
	c.DropCollection()
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
	c = sg.tweetCollection()
	// Wait for the inserts to complete
	time.Sleep(1 * time.Second)
	size, _ := c.Count()
	if size != len(tl.Tweets) {
		t.Errorf("\nExpected %d tweets but only had %d\n", tl.Tweets, size)
	}
}
