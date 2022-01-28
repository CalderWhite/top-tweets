package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/CalderWhite/top-tweets/lib"
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
var AGG_SIZE int = 300

// the number of AGG_SIZE tweet blocks that will be considered at one time. Once exceeded, we will start deleteing blocks
/**
 * (100) * (100) -- last 4 minutes
 * (300) * (100) -- last 12 minutes
 * (300) * (300) -- last 36 minutes
 *
 */
var FOCUS_PERIOD int = 300

// after this many tweets, we will prune all (1) counts in longGlobalDiff, and (0) counts in globalDiff
// 0 counts have literally no impact, and 1 counts have an infinitesimal impact on the longGlobalDiff when divided by
// the globalTweetCount
const PRUNE_PERIOD int = 10000

const recoveryFileName = "backups/top_tweets_recovery.dat"

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

// we could use the database for this, but this gobbing this struct
// reduces coupling and is also faster to ship.
// if storing the long diff becomes too great of a burden, we can add db capabilities to reconstruct it.
type RecoveryPoint struct {
	GlobalTweetCount int64
	LongDiff         *lib.WordDiff
	FocusDiff        *lib.WordDiff
	AggSize          int
	FocusPeriod      int
	Diffs            *lib.CircularQueuePublic
}

var wordDiffQueue *lib.CircularQueue = lib.NewCircularQueue(FOCUS_PERIOD)
var globalDiff *lib.WordDiff = lib.NewWordDiff()
var longGlobalDiff *lib.WordDiff = lib.NewWordDiff()
var chunkUpdateChannel = make(chan int)
var globalTweetCount int64

func createBackup() {
	longGlobalDiff.Lock()
	defer longGlobalDiff.Unlock()
	// we don't read from the individual members of the queue, so we can get away with
	// not locking every single one of them
	d := &RecoveryPoint{
		GlobalTweetCount: globalTweetCount,
		LongDiff:         longGlobalDiff,
		FocusDiff:        globalDiff,
		AggSize:          AGG_SIZE,
		FocusPeriod:      FOCUS_PERIOD,
		Diffs:            wordDiffQueue.Public(),
	}
	gob.Register((wordDiffQueue.Last()).(lib.WordDiff))

	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(d)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(recoveryFileName, buffer.Bytes(), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func restoreFromBackup() {
	longGlobalDiff.Lock()
	defer longGlobalDiff.Unlock()
	file, err := os.Open(recoveryFileName)
	if err != nil {
		log.Println("Could not open recovery file due to: ", err)
		log.Println("Starting without recovery...")
		return
	}

	dummy := lib.NewWordDiff()
	gob.Register(*dummy)
	decoder := gob.NewDecoder(file)
	recovery := &RecoveryPoint{}
	err = decoder.Decode(&recovery)
	if err != nil {
		log.Fatal(err)
	}

	globalTweetCount = recovery.GlobalTweetCount
	longGlobalDiff = recovery.LongDiff
	globalDiff = recovery.FocusDiff
	AGG_SIZE = recovery.AggSize
	FOCUS_PERIOD = recovery.FocusPeriod
	wordDiffQueue.SetQueue(recovery.Diffs)
}

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
			if err == io.EOF {
				break
			} else if errors.Is(err, syscall.ECONNRESET) {
				break
			} else {
				continue
			}
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
	for tweet := range tweets {
		globalTweetCount++
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

		if globalTweetCount%int64(PRUNE_PERIOD) == 0 {
			globalDiff.Prune(0)
			longGlobalDiff.Prune(1)

			// right after pruning, store the backup
			createBackup()
		}

		if globalTweetCount%int64(AGG_SIZE) == 0 {
			if wordDiffQueue.IsFull() {
				obj := wordDiffQueue.Dequeue()
				oldestDiff, ok := obj.(lib.WordDiff)
				if !ok {
					log.Printf("%T %v", obj, obj)
					log.Println(wordDiffQueue.String())
					log.Panic("Could not convert dequeued object to WordDiff.")
				}
				globalDiff.Sub(&oldestDiff)
			}

			wordDiffQueue.Enqueue(*diff)
			diff = lib.NewWordDiff()

			// update the chunkUpdate channel
			select {
			case chunkUpdateChannel <- 0:
			default:
				// message was not recieved, carry on.
			}
		}
	}
}

func tweetsWorker() {
	// this may fail, in which case we just start all of the values from empty (and zero)
	restoreFromBackup()

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
		longCount := int64(longGlobalDiff.GetUnlocked(word))
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
