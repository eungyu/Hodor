package main

import (
  "fmt"
  "log"
  "time"
  "os"
  "regexp"
  "os/signal"
  "syscall"
  "database/sql"
  "bytes"
  "hodor/config"
  "hodor/orchard"
  _ "github.com/mattn/go-sqlite3"
)

func stroll(config *config.Config, berries chan *orchard.Berry) {
  for {
    tree, err := orchard.NewBerryTree(config)
  
    if err != nil {
      log.Println("Couldn't instantiate berry tree")
    }

    berry, err := tree.Pick()
    if err != nil {
      log.Println("Error while picking")
    }
  
    berries <- berry
    time.Sleep(30 * time.Second)
  }
}

func nap(berries chan *orchard.Berry) {
  db, err := sql.Open("sqlite3", "./Hodor.db")
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  rows, err := db.Query("SELECT id FROM posts ORDER BY id DESC LIMIT 5")
  if err != nil {
    log.Fatal(err)
  }
  defer rows.Close()

  for rows.Next() {
    var id int
    rows.Scan(&id)
    fmt.Println(id)
  }
  rows.Close()

  berry := <- berries

  tx, err := db.Begin()
  if err != nil {
    log.Fatal(err)
  }
  stmt, err := tx.Prepare("INSERT INTO posts (subject, created) values(?, DATETIME('now'))")
  if err != nil {
    log.Fatal(err)
  }

  result, err := stmt.Exec(berry.GetSubject())
  if err != nil {
    log.Fatal(err)
  }

  lastid, err := result.LastInsertId()
  if err != nil {
    log.Fatal(err)
  }

  imgcount := 0
  imgloc := "/Users/eungyu/Desktop/static/img/upload/%d-%d.jpg" 

  var content bytes.Buffer

  for _, paragraph := range berry.GetBody() {
    isimg, _ := regexp.MatchString("^cid:[a-zA-Z0-9_]+\\.jpeg$", paragraph)
    if isimg {
      content.WriteString(fmt.Sprintf(imgloc, lastid, imgcount))
    } else {
      content.WriteString(paragraph)
    }
    content.WriteString("\n")
  }

  ustmt, err := tx.Prepare("UPDATE posts SET content = ? WHERE id = ?")
  if err != nil {
    log.Fatal(err)
  }

  ustmt.Exec(content.String(), lastid)

  tx.Commit()

  stmt.Close()
  ustmt.Close()

}

func main() {
  config := config.NewConfig("hodor.conf")


  berries := make(chan *orchard.Berry)
  go stroll(config, berries)
  go nap(berries)

  shutdown := make(chan os.Signal, 1)
  signal.Notify(shutdown, syscall.SIGTERM)

  <-shutdown
  log.Println("Shutting down")
}