package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
    "encoding/gob"
    "time"
    "context"

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

var chunkUpdateChannel = make(chan int)
var conn *pgx.Conn

func subscribeToAPI() {
	client := &http.Client{}
    req, _ := http.NewRequest("GET", "http://localhost:8080/api/chunks/stream", nil)
	resp, err := client.Do(req)

	if err != nil {
		log.Println("Error performing request to api stream", err)
		return
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Fatal("Did not get 200 OK response from api stream.", string(body))
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
    _, err := conn.Prepare(ctx, "ps1", "INSERT INTO word_counts VALUES(now(), $1, $2)")
    if err != nil {
        log.Fatalln(err)
    }
    // Insert all rows in a single commit
    tx, err := conn.Begin(ctx)
    if err != nil {
        log.Fatalln(err)
    }

    diff.Walk(func (word string, count int) {
        _, err = conn.Exec(ctx, "ps1", word, int16(count))
        if err != nil {
            log.Fatal(err)
        }
    })

    // Commit the transaction
    err = tx.Commit(ctx)
    if err != nil {
        log.Fatalln(err)
    }
}

func dbWorker() {
    ctx := context.Background()
    var err error
    conn, err = pgx.Connect(ctx, "postgresql://admin:quest@localhost:8812/qdb")
    defer conn.Close(ctx)
    if err != nil {
        log.Fatal(err)
    }

    _, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS word_counts(
        ts TIMESTAMP,
        word SYMBOL,
        count SHORT
    ) timestamp(ts)`)
    if err != nil {
        log.Fatal("Failed to create schema ", err)
    }

    for {
        <-chunkUpdateChannel

        resp, err := http.Get("http://localhost:8080/api/chunks/last")
        defer resp.Body.Close()

        // this is really inefficient for our memory usage. Implementing an actual streaming protocol
        // with just plaintext would be slower but require far less memory.
        // It appears in AWS, RAM is more expensive than compute. At least for our purposes & t4g instances.
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
    go dbWorker()

    // barebones code to stop this part of the code from DOSing the server when it goes offline
    retryMaxWindow := 5 * 5000
    maxRetries := 5
    lastDisconnect := time.Now().UnixMilli()
    disconnectCount := 0
    for {
        subscribeToAPI()
        disconnectCount++

        if disconnectCount % maxRetries  == 0 {
            t2 := time.Now().UnixMilli()
            if t2 - lastDisconnect < int64(retryMaxWindow) {
                time.Sleep(3 * time.Second)
            }
        }
    }
}
