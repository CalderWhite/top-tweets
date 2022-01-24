package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/CalderWhite/top-tweets/server/lib"

	"github.com/openacid/slim/trie"
)

// mtrie --> Mutable trie
// strie --> Static trie

/**
 * Idea: Remove words that are present in larger moving averages to see what is currently trending.
 * For example, take the 3 day moving average of words and blacklist them all.
 * This should catch things like the laughing emoji, stopwords, etc. This way we don't have to hardcode
 * blacklisted words.
 *
 * Then, what is left are the words that are not used often.
 *
 * This will also help with supporting all languages. I need stopwords for every single language on twitter
 * without this method.
 */

// the amount of tweets required to trigger a push to the wordDiffQueue
const AGG_SIZE int = 300

// the number of AGG_SIZE tweet blocks that will be considered at one time. Once exceeded, we will start deleteing blocks
/**
 * (100) * (100) -- last 4 minutes
 * (300) * (100) -- last 12 minutes
 *
 *
 */
const FOCUS_PERIOD int = 4

// the number of tweets between WordDiff.Compress() calls
const COMPRESS_PERIOD int = 5000

type StreamDataSchema struct {
	Data struct {
		Text      string    `json:"text"`
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		AuthorID  string    `json:"author_id"`
	} `json:"data"`
}

type WordPair struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

var wordDiffQueue *lib.CircularQueue = lib.NewCircularQueue(FOCUS_PERIOD)
var globalDiff *lib.WordDiff = lib.NewWordDiff()
var longGlobalDiff *lib.WordDiff64 = lib.NewWordDiff64()
var globalTweetCount int64

func streamTweets(tweets chan<- StreamDataSchema) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://api.twitter.com/2/tweets/sample/stream", nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("TWITTER_BEARER"))
	resp, err := client.Do(req)

	if err != nil {
		log.Println("Error performing request to twitter stream:", err)
		return
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Fatal("Did not get 200 OK response from twitter API.", string(body))
		return
	}

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("Got error while reading bytes:", err)
			// try to read again. Usually it is because the twitter API had nothing to give.
			continue
		}

		data := StreamDataSchema{}
		if err := json.Unmarshal(line, &data); err != nil {
			log.Println("failed to unmarshal bytes:", err)
			log.Println(string(line))
			// try to read again. Usually it is because the twitter API had nothing to give.
			continue
		}

		tweets <- data
	}
}

// this may include removing the @ symbol in the future, among other things.
func sanatizeWord(word string) string {
	return strings.ToLower(word)
}

func isValidWord(word string) bool {
	// primitive filter to get rid of uninteresting words.
	// more sophisticated algorithm idea at the top.
	if len(word) < 3 {
		return false
	}

	return true
}

func processTweets(tweets <-chan StreamDataSchema) {
	urlRule := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[\w]*))?)`)
	delimRule := regexp.MustCompile(` |"|\.|\,|\!|\?|\:|、|\n`)

	diff := lib.NewWordDiff()
	tweetCount := 0
	for tweet := range tweets {
		globalTweetCount++
		tweetCount++
		tweetCount %= AGG_SIZE
		// this is inefficient. If our process is slowing down, make this is a custom parser.
		sanatizedText := urlRule.ReplaceAllString(tweet.Data.Text, "")
		tokens := delimRule.Split(sanatizedText, -1)
		for _, token := range tokens {
			word := sanatizeWord(token)
			validWord := isValidWord(word)
			if validWord {
				globalDiff.IncWord(word)
				longGlobalDiff.IncWord(word)
				diff.IncWord(word)
			}
		}

		if globalTweetCount%int64(COMPRESS_PERIOD) == 0 {
			globalDiff.Compress()
			longGlobalDiff.Compress()
		}

		if tweetCount == 0 {
			if wordDiffQueue.IsFull() {
				oldestDiff, ok := wordDiffQueue.Dequeue().(*trie.SlimTrie)
				if !ok {
					log.Panic("Could not convert dequeued object to WordDiff.")
				}
				globalDiff.SubTrie16(oldestDiff)
			}

			// XXX: The data disappears here
			sTrie := diff.GetSlimTrie16()
			wordDiffQueue.Enqueue(sTrie)

			diff = lib.NewWordDiff()
		}
	}
}

func tweetsWorker() {
	tweets := make(chan StreamDataSchema)
	go processTweets(tweets)

	// As per the twitter API documentation, the stream can die at times.
	// So, we must restart the stream when it breaks.
	for {
		streamTweets(tweets)
	}
}

func getTop(topAmount int) []WordPair {
	top := make([]WordPair, topAmount)
	if globalTweetCount/int64((FOCUS_PERIOD*AGG_SIZE)) == 0 {
		return make([]WordPair, 0)
	}

	foundNonZero := false

	globalDiff.Lock()
	longGlobalDiff.Lock()
	defer globalDiff.Unlock()
	defer longGlobalDiff.Unlock()
	globalDiff.WalkUnlocked(func(word string, count int) {
		longCount := longGlobalDiff.GetUnlocked(word)
		if longCount == 0 {
			return
		}

		// normalize the count. So, if the count is less than the average usage of the the word over LONG_PERIOD, then it will me negative.
		// In theory, this method should return higher counts for words that are being used more than average.
		// (LONG_PERIOD / FOCUS_PERIOD) = the factor FOCUS_PERIOD is multiplied by for the LONG_PERIOD
		// then we divide the long count by that factor to make it the average usage over the past LONG_PERIOD
		count -= int(longCount / (globalTweetCount / int64(FOCUS_PERIOD*AGG_SIZE)))

		if count > 0 && count > top[0].Count {
			foundNonZero = true
			for i := 0; i < len(top); i++ {
				if count <= top[i].Count {
					// subtract one since the previous index is the one we are greater than
					i -= 1

					// shift all those less than <count> back 1
					copy(top[:i], top[1:i+1])
					// overwrite the current element
					top[i] = WordPair{Word: word, Count: count}
					break
				} else if i == len(top)-1 {
					// shift all those less than <count> back 1
					copy(top[:i], top[1:i+1])
					// overwrite the current element
					top[i] = WordPair{Word: word, Count: count}
					break
				}
			}
		}
	})

	if foundNonZero {
		// this usually doesn't happen because there are so many words with 1, but in the odd event that there isn't
		// we don't want to return zeros because they mess up the UI. Lol.
		for i, v := range top {
			if v.Count == 0 {
				return top[i+1:]
			}
		}
		return top
	} else {
		return make([]WordPair, 0)
	}
}
