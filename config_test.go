package main

import (
	"fmt"
	"os"
	"encoding/json"
	"testing"
)

func TestOpenConfig(t *testing.T) {
	name := "test_std_geostream.json"
	filepath := configPath(name)
	_, e, _ := configFile(name)
	if e {
		os.Remove(filepath)
	}
	f, err := os.Create(filepath)
	if err != nil {
		t.Errorf("Could not open file!!")
		return
	}
	sc := &Config{TwitterUser: "foo", ConsumerKey: "bar", GeoIndexMongoUrl: "localhost"}
	j, err := json.Marshal(sc)
	if err != nil {
		t.Errorf("Could not marshal the server config")
		return
	}
	if len(j) == 0 {
		t.Errorf("Expected more than 0 bytes from server config")
		return
	}
	w, err := f.Write(j)
	if w != len(j) {
		t.Errorf("Didn't write any bytes to the file something wrong!!")
		return
	}
	f.Close()
	loaded, err := loadConfig(name)
	if err != nil {
		t.Errorf(fmt.Sprintf("Erorr calling laodSErverConfig with path %s error: %s", filepath, err))
		return
	}
	if sc.TwitterUser != loaded.TwitterUser {
		t.Errorf(fmt.Sprintf("Expect %s but loaded from file %s", sc.TwitterUser, loaded.TwitterUser))
	}
	if sc.ConsumerKey != loaded.ConsumerKey {
		t.Errorf(fmt.Sprintf("Expect %s but loaded from file %s", sc.ConsumerKey, loaded.ConsumerKey))
	}
	if sc.GeoIndexMongoUrl != loaded.GeoIndexMongoUrl {
		t.Errorf(fmt.Sprintf("Expect %s but loaded from file %s", sc.GeoIndexMongoUrl, loaded.GeoIndexMongoUrl))
	}

}
