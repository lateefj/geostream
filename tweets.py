import sys
import json
from StringIO import StringIO

from tweepy.streaming import StreamListener
from tweepy import OAuthHandler
from tweepy import Stream

import requests



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
headers = {'Content-type': 'application/json', 'Accept': 'application/json'}
class StdOutListener(StreamListener):
    """ A listener handles tweets are the received from the stream.
    This is a basic listener that just prints received tweets to stdout.

    """
    def on_data(self, data):
        t = json.loads(data)
        if 'coordinates' in t.keys() and t['coordinates']:
            print t
            f = StringIO()
            f.write(data)
            f.seek(0)
            files = {'tweet': f}
            requests.post('http://localhost:8000/api/add', files=files)

        return True

    def on_error(self, status):
        print status

if __name__ == '__main__':
    l = StdOutListener()
    auth = OAuthHandler(ck, cs)
    auth.set_access_token(atk, ats)

    stream = Stream(auth, l)
    stream.filter(locations=[-180,-90,180,90])
