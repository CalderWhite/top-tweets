package main

import (
    "log"
    "strconv"
    "github.com/gin-gonic/gin"
    "github.com/openacid/slim/trie"
)

func main() {
    r := gin.Default()
    r.Static("/static", "./build/static")

    /**
     * Gets the top [limit] words (default 100), adjusted by the longGlobalDiff.
     * This adjustment allows top to produce emerging and interesting words, instead of 
     * stopwords like "the" or "los" (in spanish), etc.
     */
    r.GET("/api/words/top", func(c *gin.Context) {
        q := c.Request.URL.Query()
        limitParam, found := q["limit"]
        var limit int
        // a limit of 100 by default
        if !found {
            limit = 100
        } else {
            var err error
            limit, err = strconv.Atoi(limitParam[0])
            if err != nil {
                c.JSON(400, gin.H{
                    "message": "Error! Limit query param must be integer.",
                })
                return
            }
        }

        words := getTop(limit)
        // reverse words so highest is first.
        for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
          words[i], words[j] = words[j], words[i]
        }
        c.JSON(200, gin.H{
          "words": words,
          "total": globalTweetCount,
        })
    })

    /**
     * Returns the count for the given [:word].
     * period = [ focus | long ]
     * For the [long] period, we use longGlobalDiff. For [focus] we use globalDiff.
     */
    r.GET("/api/word/:word", func(c *gin.Context) {
        q := c.Request.URL.Query()
        period, periodFound := q["period"]

        var count int64
        if !periodFound || period[0] == "focus" {
            count, _ = globalDiff.trie.Get(c.Param("word")).(int64)
        } else if period[0] == "long" {
            count, _ = longGlobalDiff.trie.Get(c.Param("word")).(int64)
        } else {
            c.JSON(400, gin.H{
                "status": "error",
                "code": 400,
                "message": "Period parameter must be either 'focus' or 'long'.",
            })
            return
        }

        c.JSON(200, gin.H{
            "word": c.Param("word"),
            "count": count,
        })
    })

    /*
     * Produces a protobuf serialized snapshot of the current globalDiff or longGlobalDiff.
     * period = [ focus | long ]
     * the period determines which globalDiff is being used for the snapshot.
     * 
     * NOTE: The returned data is binary.
     */
    r.GET("/api/snapshot", func (c *gin.Context) {
        q := c.Request.URL.Query()
        period, periodFound := q["period"]

        if !periodFound || period[0] == "focus" {
            bytes, err := globalDiff.GetStrie().Marshal()
            if err == nil {
                c.Data(200, "application", bytes)
            } else {
                c.JSON(500, gin.H{
                    "status": "error",
                    "code": 500,
                    "message": "Server ran into error marshalling data.",
                })
            }
        } else if period[0] == "long" {
            bytes, err := longGlobalDiff.GetStrie().Marshal()
            if err == nil {
                c.Data(200, "application", bytes)
            } else {
                c.JSON(500, gin.H{
                    "status": "error",
                    "code": 500,
                    "message": "Server ran into error marshalling data.",
                })
            }
        } else {
            c.JSON(400, gin.H{
                "status": "error",
                "code": 400,
                "message": "Period parameter must be either 'focus' or 'long'.",
            })
        }
    })

    r.GET("/api/chunks/last", func (c *gin.Context) {
        diff, ok := wordDiffQueue.Last().(*trie.SlimTrie)
        if ok {
            diff.ScanFrom("", true, true, func(word []byte, value []byte) bool {
                _, count := trieCodec.Decode(value)
                log.Println(string(word), count)
                return true
            })
            bytes, err := diff.Marshal()
            if err == nil {
                c.Data(200, "application", bytes)
            } else {
                c.JSON(500, gin.H{
                    "status": "error",
                    "code": 500,
                    "message": "Server ran into error marshalling data.",
                })
            }
        } else {
            c.JSON(500, gin.H {
                "status": "error",
                "code": 500,
                "message": "Encountered error reading the latest chunk.",
            })
        }
    })

    r.GET("/", func(c *gin.Context) {
        c.File("build/index.html")
    })


    go tweetsWorker()
    r.Run("0.0.0.0:8080")
}
