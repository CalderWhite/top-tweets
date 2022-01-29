package main

import (
	"io"
	"log"
	"os"
	"strconv"

	"github.com/CalderWhite/top-tweets/lib"
	"github.com/gin-gonic/gin"
)

/**
 * NOTE: Caching should be considered for all endpoints, since they all run their respective queries fully.
 */

func main() {
	r := gin.Default()
	r.Static("/static", "./build/static")

	api := r.Group("/api")

	/**
	 * Gets the top [limit] words (default 100), adjusted by the longGlobalDiff.
	 * This adjustment allows top to produce emerging and interesting words, instead of
	 * stopwords like "the" or "los" (in spanish), etc.
	 */
	api.GET("/words/top", func(c *gin.Context) {
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

		var words []WordPair
		if limit == 100 {
			topCacheMu.Lock()
			words = topCache
			topCacheMu.Unlock()
		} else {
			words = getTop(limit)
		}
		// reverse words so highest is first.
		for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
			words[i], words[j] = words[j], words[i]
		}

		if len(words) > 0 {
			log.Println("/api/top", words[0])
		}

		c.JSON(200, gin.H{
			"words": words,
			"total": globalTweetCount,
		})
	})

	api.GET("/words/unique_count", func(c *gin.Context) {
		q := c.Request.URL.Query()
		period, periodFound := q["period"]
		targetCountStr, targetCountFound := q["count"]

		var targetCount int64
		if targetCountFound {
			var err error
			targetCount, err = strconv.ParseInt(targetCountStr[0], 10, 64)
			if err != nil {
				c.JSON(400, gin.H{
					"status":  "error",
					"code":    400,
					"message": "targetCount must be an int.",
				})
				return
			}
		}

		var total int64 = 0
		if !periodFound || period[0] == "focus" {
			if targetCountFound {
				globalDiff.Walk(func(word string, count int) {
					if count == int(targetCount) {
						total++
					}
				})
			} else {
				globalDiff.Walk(func(word string, count int) {
					total++
				})
			}
		} else if period[0] == "long" {
			if targetCountFound {
				longGlobalDiff.Walk(func(word string, count int) {
					if int64(count) == targetCount {
						total++
					}
				})
			} else {
				longGlobalDiff.Walk(func(word string, count int) {
					total++
				})
			}
		} else {
			c.JSON(400, gin.H{
				"status":  "error",
				"code":    400,
				"message": "Period parameter must be either 'focus' or 'long'.",
			})
			return
		}

		c.JSON(200, gin.H{
			"count": total,
		})
	})

	/**
	 * Returns the count for the given [:word].
	 * period = [ focus | long ]
	 * For the [long] period, we use longGlobalDiff. For [focus] we use globalDiff.
	 */
	api.GET("/word/:word", func(c *gin.Context) {
		q := c.Request.URL.Query()
		period, periodFound := q["period"]

		var count interface{}
		if !periodFound || period[0] == "focus" {
			count = globalDiff.Get(c.Param("word"))
		} else if period[0] == "long" {
			count = longGlobalDiff.Get(c.Param("word"))
		} else {
			c.JSON(400, gin.H{
				"status":  "error",
				"code":    400,
				"message": "Period parameter must be either 'focus' or 'long'.",
			})
			return
		}

		c.JSON(200, gin.H{
			"word":  c.Param("word"),
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
	api.GET("/snapshot", func(c *gin.Context) {
		q := c.Request.URL.Query()
		period, periodFound := q["period"]

		if !periodFound || period[0] == "focus" {
			c.Data(200, "application", globalDiff.Serialize())
		} else if period[0] == "long" {
			c.Data(200, "application", longGlobalDiff.Serialize())
		} else {
			c.JSON(400, gin.H{
				"status":  "error",
				"code":    400,
				"message": "Period parameter must be either 'focus' or 'long'.",
			})
		}
	})

	api.GET("/chunks/last", func(c *gin.Context) {
		diff, ok := wordDiffQueue.Last().(lib.WordDiff)
		if ok {
			c.Data(200, "application", diff.Serialize())
		} else {
			c.JSON(500, gin.H{
				"status":  "error",
				"code":    500,
				"message": "Encountered error reading the latest chunk.",
			})
		}
	})

	api.GET("/chunks/stream", func(c *gin.Context) {
		c.Stream(func(w io.Writer) bool {
			<-chunkUpdateChannel
			w.Write([]byte("update\n"))
			return true
		})
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("build/favicon.ico")
	})

	r.GET("/", func(c *gin.Context) {
		c.File("build/index.html")
	})

	prod := os.Getenv("TOP_TWEETS_MODE") == "PRODUCTION"
	go tweetsWorker()
	if !prod {
		r.Run("0.0.0.0:8080")
	} else {
		r.RunTLS("0.0.0.0:8080", "/etc/letsencrypt/live/toptweets.calderwhite.com/fullchain.pem", "/etc/letsencrypt/live/toptweets.calderwhite.com/privkey.pem")
	}
}
