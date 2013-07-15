geostream
=========

![Image](screenshot.png?raw=true)

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

##Replace all the REQUIRED with values in your configuration file of which you can get from https://dev.twitter.com/ (My Applications)

Source Install
--------------
1. git clone git@github.com:lateefj/geostream.git
2. Edit configuration file (config_sample.json)
3. go get && go build
4. ./geostream --config=config_sample.json
5. Browser: http://localhost:8000

Options
-------
./geostrem --stream=true/false by default the built in streaming is on however this can be turned off by just passing the --stream=false

Python Client
-------------

Python setup: cd client; pip install -r requirements.txt

Python script command: python tweets.py $CONFIG_FILE

Distributed Geospacial Data:
----------------------------
There is an initial implementation of a way to distribute the geospacial data across many nodes (MongoDB collections). Still tracking down some edge case bugs but the concept is implmented. The other part that is tricky is setting up how the data is disributed accross a cluster of servers.


TODO:
-----
 * Clean code mess

