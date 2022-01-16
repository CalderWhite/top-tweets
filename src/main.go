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
)

type StreamDataSchema struct {
    Data struct {
        Text      string    `json:"text"`
        ID        string    `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        AuthorID  string    `json:"author_id"`
    } `json:"data"`
}

var wordDiffQueue *CircularQueue = NewCircularQueue(300)
var diff *WordDiff = NewWordDiff()

func streamTweets(tweets chan<- StreamDataSchema) {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", "https://api.twitter.com/2/tweets/sample/stream", nil)
    req.Header.Set("Authorization", "Bearer " + os.Getenv("TWITTER_BEARER"))
    resp, err := client.Do(req)

    if err != nil {
        log.Println("Println performing request to twitter stream:", err)
        return
    }

    defer resp.Body.Close()
    reader := bufio.NewReader(resp.Body)
    for {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            log.Println("Got error while reading bytes:", err)
            // exit out so we can restart the stream
            return
        }

        data := StreamDataSchema{}
        if err := json.Unmarshal(line, &data); err != nil {
            log.Println("failed to unmarshal bytes:", err)
            return
        }

        tweets <- data
    }
}

func processTweets(tweets <-chan StreamDataSchema) {
    //diff := NewWordDiff()
    urlRule := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[\w]*))?)`)
    delimRule := regexp.MustCompile(` |"|\.|\,|\!|\?|\:|、`)
    for tweet := range tweets {
        // this is inefficient. If our process is slowing down, make this is a custom parser.
        sanatizedText := urlRule.ReplaceAllString(tweet.Data.Text, "")
        tokens := delimRule.Split(sanatizedText, -1)
        for _, token := range tokens {
            token = strings.ToLower(token)
            //log.Println(token)
            val, ok := diff.trie.Get(token).(int)
            if !ok {
                val = 0
            }
            diff.trie.Put(token, val + 1)
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

func main() {
    go tweetsWorker()

    for {
        log.Println("Table:")
        diff.trie.Walk(func(key string, value interface{}) error {
            val, ok := value.(int)
            if ok {
                if val > 10 {
                    log.Println(key, value)
                }
            }

            return nil
        })
        time.Sleep(1 * time.Second)
    }
}
