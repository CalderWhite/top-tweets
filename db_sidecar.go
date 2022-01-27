package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/CalderWhite/top-tweets/lib"
)

/**
 * A quick note on the schema: I would prefer the SYMBOL be stored with a 64-bit integer instead of 32-bit,
 * however, I cannot change QuestDB's internals. At least, not without effort.
 * If there were 1M new words per day, we would run out of indicies in the symbol table after 11 years.
 * I am content with that. This essentially puts a maximum on the number of novel words per day we can encounter.
 * TBD: How many new words we see per day.
 * Note: If we upgraded our API to a 10% firehose from twitter, we would likely want to get into QuestDB's internals
 *       and switch the index to 64-bit, OR do the symbol table ourselves.
 */

// when run outside of docker-compose, these can both be set to "localhost"
const (
	questDbHost   = "questdb"
	topTweetsHost = "top_tweets"
)

var chunkUpdateChannel = make(chan int)
var conn *pgx.Conn
var production = os.Getenv("TOP_TWEETS_MODE") == "PRODUCTION"
var apiUrl = getApiUrl()

func getApiUrl() string {
	if production {
		return "https://toptweets.calderwhite.com:8080"
	} else {
		return "http://" + topTweetsHost + ":8080"
	}
}

func subscribeToAPI() {
	client := &http.Client{}
	req_url := fmt.Sprintf("%s/api/chunks/stream", apiUrl)
	req, err := http.NewRequest("GET", req_url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	resp, err := client.Do(req)

	if err != nil {
		log.Println("Error performing request to api stream", err)
		return
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("Did not get 200 OK response from api stream.", string(body))
		return
	}

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	for {
		_, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("Got error while reading bytes:", err)
			return
		}

		chunkUpdateChannel <- 0
	}
}

func insertRows(ctx context.Context, diff *lib.WordDiff) {
	// Prepared statement given the name 'ps1'
	_, err := conn.Prepare(ctx, "ps1", "INSERT INTO word_counts VALUES($1, $2, $3)")
	if err != nil {
		log.Println(err)
		return
	}
	// Insert all rows in a single commit
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Println(err)
	}

	ts := time.Now()
	diff.Walk(func(word string, count int) {
		_, err = conn.Exec(ctx, "ps1", ts, word, int16(count))
		if err != nil {
			log.Println(err)
			return
		}
	})

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		log.Println(err)
		return
	}
}

func dbWorker() {
	ctx := context.Background()
	var err error
	conn_url := fmt.Sprintf("postgresql://admin:quest@%s:8812/qdb", questDbHost)
	conn, err = pgx.Connect(ctx, conn_url)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS word_counts(
        ts TIMESTAMP,
        word SYMBOL,
        count SHORT
    ) timestamp(ts)`)
	if err != nil {
		log.Println("Failed to create schema ", err)
		return
	}

	for {
		<-chunkUpdateChannel

		req_url := fmt.Sprintf("%s/api/chunks/last", apiUrl)
		resp, err := http.Get(req_url)
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()

		// this is fine for small packets (like chunks)
		// but for the long-term diff it is inefficnet.
		decoder := gob.NewDecoder(resp.Body)
		diff := lib.NewWordDiff()
		err = decoder.Decode(&diff)
		if err != nil {
			//log.Fatal(err)
			log.Println(err)
			continue
		}
		resp.Body.Close()

		insertRows(ctx, diff)
	}
}

func main() {
	go func() {
		for {
			dbWorker()
			time.Sleep(1 * time.Second)
		}
	}()
	for {
		subscribeToAPI()
		time.Sleep(1 * time.Second)
	}
}
