package main

import (
    "log"
    "net/http"
    "os"
    "bufio"
    "time"
    "encoding/json"
    "regexp"
    "strings"
    "io/ioutil"
    mtrie "github.com/dghubble/trie"
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

 /**
  * New idea: just continuously add to the longDiff and track how many periods it includes.
  *           This will be an extremely long term average.
  * Note: log_2( 40 tweets/second * 60 * 60 * 24 * 365) = 30.23
  * This means that if every tweet contained 1 common word, it would take a year for us to get to 31 bits.
  * After 1 year we would approach 31.23 bits
  * After 100 years we would get to 36 bits
  * After 100 000 years we would require 47 bits.
  * With this I can assume that a 64 bit integer is good enough to contain the long-term counts.
  * Note: If every single word in the tweet was the same, it would still take 100 000 years to get to 54 bits.
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
const FOCUS_PERIOD int = 100

type StreamDataSchema struct {
    Data struct {
        Text      string    `json:"text"`
        ID        string    `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        AuthorID  string    `json:"author_id"`
    } `json:"data"`
}

type WordPair struct {
    Word string `json:"word"`
    Count int64 `json:"count"`
}

var stopWords *mtrie.RuneTrie = NewStopWordsTrie()
var wordDiffQueue *CircularQueue = NewCircularQueue(FOCUS_PERIOD)
var globalDiff *WordDiff = NewWordDiff()
var longGlobalDiff *WordDiff = NewWordDiff()

var globalTweetCount int64;

func streamTweets(tweets chan<- StreamDataSchema) {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", "https://api.twitter.com/2/tweets/sample/stream", nil)
    req.Header.Set("Authorization", "Bearer " + os.Getenv("TWITTER_BEARER"))
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
    stopWord := stopWords.Get(word)
    if stopWord != nil {
        return false
    }

    // primitive filter to get rid of uninteresting words.
    // more sophisticated algorithm idea at the top.
    if len(word) < 3 {
        return false
    }

    return true
}

func processTweets(tweets <-chan StreamDataSchema) {
    diff := NewWordDiff()
    urlRule := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[\w]*))?)`)
    delimRule := regexp.MustCompile(` |"|\.|\,|\!|\?|\:|、|\n`)

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
            if (validWord) {
                diff.IncWord(word)
                globalDiff.IncWord(word)
                longGlobalDiff.IncWord(word)
            }
        }

        if tweetCount == 0 {
            if (wordDiffQueue.IsFull()) {
                oldestDiff, ok := wordDiffQueue.Dequeue().(*trie.SlimTrie)
                if !ok {
                    log.Panic("Could not convert dequeued object to WordDiff.")
                }
                globalDiff.Sub(oldestDiff)
            }

            strie := diff.GetStrie()
            wordDiffQueue.Enqueue(strie)

            diff = NewWordDiff()
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
    if globalTweetCount / int64(FOCUS_PERIOD * AGG_SIZE) == 0 {
        return make([]WordPair, 0)
    }

    foundNonZero := false

    globalDiff.Lock.Lock()
    defer globalDiff.Lock.Unlock()
    longGlobalDiff.Lock.Lock()
    defer longGlobalDiff.Lock.Unlock()
    globalDiff.trie.Walk(func(word string, _count interface{}) error {
        count, ok := _count.(int64)
        if !ok {
            return nil
        }

        longCount, ok := longGlobalDiff.trie.Get(word).(int64);
        if !ok {
          // why is this happening???
          log.Println("Got not okay from longGlobalDiff on word in globalDiff. Word:", word)
        }

        // divide the long-term count by the globalTweetCount to get a word's average usage.
        // subtract the average usage from the count to get the above average usage.
        count -= longCount / (globalTweetCount / (int64(FOCUS_PERIOD * AGG_SIZE)))

        if count > 0 && count > top[0].Count {
            foundNonZero = true
            for i := 0; i < len(top); i++ {
                if count <= top[i].Count {
                    // subtract one since the previous index is the one we are greater than
                    i -= 1;

                    // shift all those less than <count> back 1
                    copy(top[:i], top[1:i+1])
                    // overwrite the current element
                    top[i] = WordPair{Word: word, Count: count}
                    return nil
                } else if i == len(top) - 1 {
                    // shift all those less than <count> back 1
                    copy(top[:i], top[1:i+1])
                    // overwrite the current element
                    top[i] = WordPair{Word: word, Count: count}
                    return nil
                }
            }
        }

        return nil
    })

    if foundNonZero {
        return top
    } else {
        log.Println("Stuff")
        return make([]WordPair, 0)
    }
}
