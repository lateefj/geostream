import sys
import json

from twitter import Twitter, OAuth
from twitter.stream import TwitterStream


if len(sys.argv) < 2:
    print('FAILING NEED CONFIGURATION file as first parameter')
    sys.exit(1)
cpath = sys.argv[1]
print(cpath)
f = open(cpath)
conf = json.loads(f.read())
atk = conf['ACCESS_TOKEN_KEY']
ats = conf['ACCESS_TOKEN_SECRET']
ck = conf['CONSUMER_KEY']
cs = conf['CONSUMER_SECRET']
auth = OAuth(atk, ats, ck, cs)
t = Twitter(auth=auth)
f = open('sample.json', 'w')
f.write(json.dumps(t.search.tweets(geocode='45.5236,122.6750,100km')['statuses']))
f.close()
count = 0

for v in t.search.tweets(geocode='45.5236,122.6750,20km')['statuses']:
    print(v)
    count += 1


print('Count of tweets is {0}'.format(count))
# Seems like nobody has a working stream client !!!
"""
stream = TwitterStream(auth)
for s in stream.statuses.sample():
    print(s)
"""




