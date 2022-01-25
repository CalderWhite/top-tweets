package main

import (
	"bufio"
	"io/ioutil"
	"log"
    "context"
	"net/http"
    "encoding/gob"

    "github.com/jackc/pgx/v4"


    "github.com/CalderWhite/top-tweets/lib"
)

var chunkUpdateChannel = make(chan int)
const (
    host     = "localhost"
    port     = 8812
    user     = "admin"
    password = "quest"
    dbname   = "qdb"
)

var conn *pgx.Conn
var err error

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

func dbWorker() {
    ctx := context.Background()
    conn, _ = pgx.Connect(ctx, "postgresql://admin:quest@localhost:8812/qdb")
    defer conn.Close(ctx)

    _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS word_diffs_compressed(
        ts TIMESTAMP,
        data BINARY
    ) timestamp(ts)`)
    if err != nil {
        log.Fatal("Failed to create schema ", err)
    }

    for {
        <-chunkUpdateChannel

        resp, err := http.Get("http://localhost:8080/api/chunks/last")
        defer resp.Body.Close()

        decoder := gob.NewDecoder(resp.Body)
        diff := lib.NewWordDiff()
        err = decoder.Decode(&diff)
        if err != nil {
            //log.Fatal(err)
            log.Println(err)
            continue
        }
        resp.Body.Close()

        _, err = conn.Exec(ctx, "INSERT INTO word_diffs_compressed VALUES(systimestamp(), $1)", diff.Serialize())
        if err != nil {
            log.Fatal(err)
        }

    }
}

func main() {
    go dbWorker()
    for {
        subscribeToAPI()
    }
}
