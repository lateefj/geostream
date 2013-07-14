geostream
=========

Simple example application that uses Golang, Mongodb with source of data from streaming Twitter. DEMO APP ONLY DO NOT USE EXECPT FOR REFERENCE

Very simple Javascript UI that displays on a Google map Twitter posts on a map.

There is a basic configuration file that needs these parameters (config_sample.json):

```javascript

{
  "TWITTER_USER": "lateefjackson",
  "GOOGLE_MAPS_API": "REQUIRED",
  "CONSUMER_KEY": "REQUIRED",
  "CONSUMER_SECRET": "REQUIRED",
  "ACCESS_TOKEN_KEY": "REQUIRED",
  "ACCESS_TOKEN_SECRET": "REQUIRED",
  "GEO_INDEX_MONGO_URL": "mongodb://localhost:27017"
}
```

Install
-------

1. Edit configuration file (config_sample.json)
2. go get && go build
3. ./geostream --config=config_sample.json
4. Browser: http://localhost:8000

Options
-------
./geostrem --stream=true/false by default the built in streaming is on however this can be turned off by just passing the --stream=false

Python Client
-------------

Python setup: cd client; pip install -r requirements.txt

Python script command: python tweets.py $CONFIG_FILE


TODO:
-----
 * Clean code mess
 * Implement sharded solution for goespacial data

