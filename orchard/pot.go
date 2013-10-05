package orchard

import (
  "bufio"
  "bytes"
  "database/sql"
  "fmt"
  _ "github.com/mattn/go-sqlite3"
  "hodor/config"
  "image/jpeg"
  "log"
  "os"
  "regexp"
  "time"
)

const (
  SINGLE = iota
  LATEST = iota
)

const numFrontPage = 5

const singleQuery = "SELECT id, title, content, created FROM posts WHERE id = ?"
const latestQuery = "SELECT id, title, content, created FROM posts ORDER BY id DESC LIMIT ?"

const insertQuery = "INSERT INTO posts (title, created) values(?, DATETIME('now'))"
const updateQuery = "UPDATE posts SET content = ? WHERE id = ?"

const imgRegEx = "^cid:[a-zA-Z0-9_]+\\.jpeg$"

type Pick struct {
  mode int
  num  int
  res  chan []*Berry
}

type Pot struct {
  config *config.Config
  berries chan *Berry
  pickch chan Pick
  shutdown chan bool
}

func NewPot(config *config.Config) (*Pot, error) {
  berries := make(chan *Berry)
  pickch := make(chan Pick)
  shutdown := make(chan bool)

  pot := &Pot{
    config: config,
    berries: berries,
    pickch: pickch,
    shutdown: shutdown }

  go potter(config, berries, pickch, shutdown)

  return pot, nil
}

func NewLatestPick(res chan []*Berry) Pick {
  return Pick {
    mode: LATEST,
    num: numFrontPage,
    res: res }  
}

func (p *Pot) Put(berry *Berry) {
  p.berries <- berry
}

func (p *Pot) GetOne(id int) []*Berry {
  resch := make(chan []*Berry)
  pick := Pick {
    mode: SINGLE,
    num: id,
    res: resch } 

  p.pickch <- pick
  berries := <- resch

  return berries
}

func (p *Pot) GetLatest() []*Berry {
  resch := make(chan []*Berry)
  pick := Pick {
    mode: LATEST,
    num: numFrontPage,
    res: resch } 

  p.pickch <- pick
  berries := <- resch

  return berries
}

func (p *Pot) Shutdown() {
  p.shutdown <- true
}

func potter(config *config.Config, berries chan *Berry, pickch chan Pick, shutdown chan bool) {
  mode := config.ServerConfig.Mode()
  basedir := config.ServerConfig.BaseDir()
  dbfile  := fmt.Sprintf("%s/%s", basedir, config.DbConfig.File())
  postConfig := config.PostConfig

  cache := make(map[int64]*Berry)

  var recent []int64
  warmedup := false

  for {
    select {
    case berry := <-berries:
      db, err := sql.Open("sqlite3", dbfile)  

      tx, err := db.Begin()
      if err != nil {
        log.Fatal(err)
      }
      stmt, err := tx.Prepare(insertQuery)
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
      imgformat := "%d-%d.jpg"

      var content bytes.Buffer

      receipts := make(map[string]string)

      // refactor this
      for _, paragraph := range berry.GetBody() {
        isimg, _ := regexp.MatchString(imgRegEx, paragraph)
        if isimg {
          imgname := fmt.Sprintf(imgformat, lastid, imgcount)
          
          imgtagfmt := "<img src=\"http://lately.cc/eungyu%s/%s\">"
          if mode == "dev" {
            imgtagfmt = "<img src=\"%s/%s\">"
          }

          content.WriteString(fmt.Sprintf(imgtagfmt, postConfig.ImgUrl(), imgname))
          receipts[paragraph] = imgname
          imgcount = imgcount + 1
        } else {
          content.WriteString(paragraph)
        }
        content.WriteString("\n")
      }

      ustmt, err := tx.Prepare(updateQuery)
      if err != nil {
        log.Fatal(err)
      }

      ustmt.Exec(content.String(), lastid)

      tx.Commit()

      log.Println("New Berry ", berry.GetSubject())
      cache[lastid] = NewBerryFromContent(int64(lastid), berry.GetSubject(), content.String(), time.Now().UTC())

      if warmedup {
        newindex := make([]int64, 0, len(recent))
        newindex = append(newindex, lastid)
        newindex = newindex[0:len(recent)]
        copy(newindex[1:], recent[0:len(recent)-1])

        recent = newindex
      }

      log.Println(recent)

      for cid, img := range berry.ImgMap() {
        name := fmt.Sprintf("%s%s/%s", basedir, postConfig.ImgUrl(), receipts[cid])

        fo, _ := os.Create(name)
        w := bufio.NewWriter(fo)

        err = jpeg.Encode(w, img, nil)
        if err != nil {
          log.Fatal("Failed to write image")
        }
      }

      stmt.Close()
      ustmt.Close()
      db.Close()

    case pick := <-pickch:
      db, err := sql.Open("sqlite3", dbfile)  
      if err != nil {
        log.Fatal(err)
      }

      log.Println("Pick requested")

      var queryString string

      if pick.mode == SINGLE {
        queryString = singleQuery
      } else {
        queryString = latestQuery
      }

      stmt, err := db.Prepare(queryString)
      if err != nil {
        log.Fatal("Failed to fetch from db")
      }

      rows, err := stmt.Query(pick.num)
      if err != nil {
        log.Fatal("Failed to query")
      }

      i := 0

      size := 1
      if pick.mode == LATEST {
        size = pick.num
      }

      berries := make([]*Berry, size)

      cachehit := false
      if pick.mode == SINGLE {
        if _, ok := cache[int64(pick.num)]; ok {
          cachehit = true
        }
      } else if warmedup {
        cachehit = true
      }

      if cachehit {
        if pick.mode == LATEST {
          for _, id := range recent {
            berry := cache[id]
            berries[i] = berry
            i = i + 1
          }
        } else {
          berries[0] = cache[int64(pick.num)]
        }
      } else {
        if pick.mode == LATEST {
          recent = []int64{}
          warmedup = true
        }
        for rows.Next() {

          var id int64
          var subject string
          var content string
          var created time.Time

          rows.Scan(&id, &subject, &content, &created)

          berry := NewBerryFromContent(id, subject, content, created)
          berries[i] = berry
          cache[id] = berry

          if pick.mode == LATEST {
            recent = append(recent, id)
          }
          i = i + 1
        }
      }

      log.Println(recent)

      pick.res <- berries

      rows.Close()
      stmt.Close()
      db.Close()

    case <-shutdown:
      log.Println("Shutting Down")
      return
    }
  }

}  

