package main

import (
	"os"
	"fmt"
	"flag"
	"errors"
	"io/ioutil"
	"encoding/json"
)

const (
	SERVER_CONFIG_FILENAME = "env_geostream.json"
)

type Config struct {
	TwitterUser       string `json:"TWITTER_SUER"`
	GoogleMapsAPI     string `json:"GOOGLE_MAPS_API"`
	InitLatitude      string `json:"INIT_LATITUDE"`
	InitLongitude     string `json:"INIT_LONGITUDE"`
	GeoIndexMongoUrl  string `json:"GEO_INDEX_MONGO_URL"`
	ConsumerKey       string `json:"CONSUMER_KEY"`
	ConsumerSecret    string `json:"CONSUMER_SECRET"`
	AccessTokenKey    string `json:"ACCESS_TOKEN_KEY"`
	AccessTokenSecret string `json:"ACCESS_TOKEN_SECRET"`
}

var configParam *string = flag.String("config", "", "Config file location")

func configPath(name string) string {
	flag.Parse()
	if *configParam != "" {
		return *configParam
	}

	home := os.Getenv("HOME")

	return fmt.Sprintf("%s/%s", home, name)
}

func configFile(name string) (*os.File, bool, error) {
	f, err := os.Open(configPath(name))
	exists := os.IsNotExist(err)
	return f, exists, err
}

func OpenConfig(f *os.File, co interface{}) error {
	b, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Printf("Could not all bytes in config file %s", err)
		return err
	}
	return json.Unmarshal(b, co)
}

func loadConfig(name string) (*Config, error) {
	sc := &Config{}
	f, e, err := configFile(name)
	if e {
		fmt.Printf("SERVER CONFIG FILE DOES not exist at: %s\n", configPath(name))
		return nil, errors.New("SERVER CONFIG FILE DOES NOT EXIST")
	}
	if err != nil {
		fmt.Printf("Failed to open config file path %s error: %s\n", configPath(name), err)
		return nil, err
	}
	err = OpenConfig(f, sc)
	if err != nil {
		fmt.Printf("Could not unmashal json %s\n", err)
		return nil, err
	}
	return sc, nil
}

func GetConfig() (*Config, error) {
	return loadConfig(SERVER_CONFIG_FILENAME)
}
