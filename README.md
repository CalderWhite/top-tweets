# top-tweets

Shows words that are in high use right now, while filtering out words that are at an "average" usage.

## Some cool things I did for this

- Used a map for hot data since it had lower overhead than tries
- Compressed the maps after a fixed period using a [SlimTrie](https://github.com/openacid/slim) data structure (similar to [HAT-Trie](https://tessil.github.io/2017/06/22/hat-trie.html))
- Used some ideas from signal theory to deterministically extract "interesting" words (words of emerging popularity) among hundreds of thousands of words
- Managed to make this extremely light weight. The program is highly performant on an AWS `t4g.nano` (2 vCPU, 0.5 GiB RAM)


## Building + Running
To Build:
```
npm install
npm run build
docker build . -t top-tweets
```


To Run:
```
docker run --rm -it -e TWITTER_BEARER="$TWITTER_BEARER" -p 8080:8080 top-tweets
```
