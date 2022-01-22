package main

import (
    "strconv"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    r.Static("/static", "./build/static")

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

    r.GET("/api/word/:word", func(c *gin.Context) {
        q := c.Request.URL.Query()
        period, periodFound := q["period"]

        var count int
        if !periodFound || period[0] == "focus" {
            count, _ = globalDiff.trie.Get(c.Param("word")).(int)
        } else if period[0] == "long" {
            count, _ = longGlobalDiff.trie.Get(c.Param("word")).(int)
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

    r.GET("/", func(c *gin.Context) {
        c.File("build/index.html")
    })


    go tweetsWorker()
    r.Run("0.0.0.0:8080")
}
