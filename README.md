geostream
=========

Simple example application that uses Golang, Mongodb with source of data from streaming Twitter. DEMO APP ONLY DO NOT USE EXECPT FOR REFERENCE

Very simple Javascript UI that displays on a Google map Twitter posts on a map.

There is a basic configuration file that needs these parameters:

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

Currenly this file must be located in: $HOME/env_geostream.json

Python setup: pip install -r requirements.txt

Python script command: python tweets.py $HOME/env_geostream.json


TODO:

 * Twitter streaming in Go currently OAuth was not working
 * Implement sharded solution for goespacial data
 * Make config file a parameter instead of default location of $HOME/env_geostream.json
