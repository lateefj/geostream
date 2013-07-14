package main

import (
	"fmt"
	"testing"
	"github.com/lateefj/httpstream"
)

func TestGetTweets(t *testing.T) {
	stream := make(chan *httpstream.Tweet)
	go func() {
		err := GetTweets(stream)
		if err != nil {
			t.Errorf("error from GetTweets: %s", err)
		}
	}()
	i := 0
	for s := range stream {
		if s.Coordinates != nil {
			fmt.Printf("(COORDS) @%s: %s {%s}\n", s.User.ScreenName, s.Text, s.Coordinates)
		} else if s.Place != nil {
			fmt.Printf("(PLACE) @%s: %s {%s}\n", s.User.ScreenName, s.Text, s.Place)
		} else {
			fmt.Printf("(Doesn't have Coordinates or Place?)@%s: %s {%s}\n", s.User.ScreenName, s.Text)
		}
		if i > 25 {
			close(stream)
		}
		i++
	}
}
