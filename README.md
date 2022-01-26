# top-tweets

Shows words that are in high use right now, while filtering out words that are at an "average" usage.

## Some cool things I did for this

- Used a columnstore database with a symbol table to highly compress data while without losing the ability to easily query it
- Used some ideas from signal theory to deterministically extract "interesting" words (words of emerging popularity) among hundreds of thousands of words
- Managed to make this extremely light weight. The program is highly performant on an AWS `t4g.nano` (2 vCPU, 0.5 GiB RAM)


## Building + Running

```
docker-compose build
docker-compose up
```


## Development

For debugging, any of the docker containers can be built and run individually.


For the main service:
```
cd web && npm install && npm run build && cd ..
docker build . -t calderwhite/top-tweets
docker run --rm -it -e TWITTER_BEARER -p 8080:8080 --name top-tweets calderwhite/top-tweets
```

For the db sidecar (what downloads the data stream into the database)
```
docker build -f dockerfiles/db-sidecar -t calderwhite/db-sidecar .
docker run --rm -it --name db-sidecar --network=host calderwhite/db-sidecar
```
