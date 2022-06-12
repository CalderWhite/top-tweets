package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"cloud.google.com/go/translate"
	"github.com/CalderWhite/top-tweets/lib"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

// NOTE: This is now 100% coupled with twitter_worker because of the translateCache
//       twitter_worker won't run without the translateCache since it expects it for backups.
var translateCache map[string]string = make(map[string]string)

type WordPair struct {
	Word        string `json:"word"`
	Count       int    `json:"count"`
	Translation string `json:"translation"`
}

func translateText(targetLanguage, text string) (string, error) {
	// text := "The Go Gopher is cute"
	ctx := context.Background()

	lang, err := language.Parse(targetLanguage)
	if err != nil {
		return "", fmt.Errorf("language.Parse: %v", err)
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	resp, err := client.Translate(ctx, []string{text}, lang, nil)
	if err != nil {
		return "", fmt.Errorf("Translate: %v", err)
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("Translate returned empty response to text: %s", text)
	}
	return resp[0].Text, nil
}

/**
 * NOTE: Caching should be considered for all endpoints, since they all run their respective queries fully.
 */

func main() {
	prod := os.Getenv("TOP_TWEETS_MODE") == "PRODUCTION"
	r := gin.New()
	r.Use(gzip.Gzip(gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".ico", ".png", ".jpg"}),
		gzip.WithExcludedPaths([]string{"/api/chunks/stream"}),
	), gin.LoggerWithWriter(gin.DefaultWriter, "/api/words/top"),
       gin.Recovery())
	var buildRoot string
	if prod {
		exec.Command("cp", "-r", "./build", "./tmp").Output()
		buildRoot = "tmp/build"
	} else {
		buildRoot = "build"
	}
	r.Static("/static", buildRoot+"/static")

	api := r.Group("/api")

	api.GET("/translate", func(c *gin.Context) {
		q := c.Request.URL.Query()
		wordList, wordFound := q["word"]
		if !wordFound {
			c.JSON(400, gin.H{
				"message": "You must provide a <word> in the query string.",
				"status":  "error",
				"code":    400,
			})
			return
		}
		wordb64 := wordList[0]
		wordb, err := base64.URLEncoding.DecodeString(wordb64)
		if err != nil {
			c.JSON(500, gin.H{
				"message": fmt.Sprintf("%v", err),
				"status":  "error",
				"code":    500,
			})
			return
		}
		word := string(wordb)
		cacheHit, ok := translateCache[word]
		if ok {
			c.JSON(200, gin.H{
				"translation": cacheHit,
			})
		} else {
			translation, err := translateText("en", word)
			if err != nil {
				c.JSON(500, gin.H{
					"message": fmt.Sprintf("%v", err),
					"status":  "error",
					"code":    500,
				})
			} else {
				translateCache[word] = translation
				c.JSON(200, gin.H{
					"translation": translation,
				})
			}
		}
	})

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

		var words []WordRankingPair
		if limit == 100 {
			words = make([]WordRankingPair, len(topCache))
			copy(words, topCache)
		} else {
			words = getTop(limit)
		}
		// reverse words so highest is first.
		for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
			words[i], words[j] = words[j], words[i]
		}

		for i, wordPair := range words {
			translation, foundTranslation := translateCache[wordPair.Word]
			if foundTranslation {
				words[i].Translation = translation
			}
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
	api.GET("/word", func(c *gin.Context) {
		q := c.Request.URL.Query()
		wordList, wordFound := q["word"]
		if !wordFound {
			c.JSON(400, gin.H{
				"message": "You must provide a <word> in the query string.",
				"status":  "error",
				"code":    400,
			})
			return
		}
		word := wordList[0]
		period, periodFound := q["period"]

		translation, foundTranslation := translateCache[word]
		tText := ""
		if foundTranslation {
			tText = translation
		}

		if !periodFound || period[0] == "focus" {
			count := globalDiff.Get(word)
			c.JSON(200, WordPair{
				Word:        word,
				Count:       count,
				Translation: tText,
			})
		} else if period[0] == "long" {
			count := longGlobalDiff.Get(word)
			c.JSON(200, WordPair{
				Word:        word,
				Count:       count,
				Translation: tText,
			})
		} else {
			c.JSON(400, gin.H{
				"status":  "error",
				"code":    400,
				"message": "Period parameter must be either 'focus' or 'long'.",
			})
			return
		}
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
		c.File(buildRoot + "/favicon.ico")
	})

	r.GET("/", func(c *gin.Context) {
		c.File(buildRoot + "/index.html")
	})

	go tweetsWorker()
	if !prod {
		r.Run("0.0.0.0:8080")
	} else {
		r.RunTLS("0.0.0.0:8080", "/etc/letsencrypt/live/toptweets.calderwhite.com/fullchain.pem", "/etc/letsencrypt/live/toptweets.calderwhite.com/privkey.pem")
	}
}
