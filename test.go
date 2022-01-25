package main

import (
  "context"
  "log"

  "github.com/jackc/pgx/v4"
)

var conn *pgx.Conn
var err error

func main() {
  ctx := context.Background()
  conn, _ = pgx.Connect(ctx, "postgresql://admin:quest@localhost:8812/qdb")
  defer conn.Close(ctx)

  // text-based query
  _, err := conn.Exec(ctx, "INSERT INTO word_diffs_compressed VALUES(systimestamp(), $1)", []byte("blah"))
  if err != nil {
    log.Fatalln(err)
  }
}
