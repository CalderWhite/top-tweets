# top-tweets


To Build:
```
docker build . -t top-tweets
```


To Run:
```
docker run --rm -it -e TWITTER_BEARER="$TWITTER_BEARER" -p 8080:8080 top-tweets
```
