package main

import (
  "fmt"
  "github.com/gorilla/mux"
  _ "github.com/mattn/go-sqlite3"
  "hodor/blog"
  "hodor/config"
  "hodor/orchard"
  "log"
  "net"
  "net/http"
  "os"
  "os/signal"
  "strconv"
  "syscall"
  "time"
)

const defaultConfigFile = "/etc/hodor/hodor.conf"

func hodor(config *config.Config, pot *orchard.Pot) {
  for {
    tree, err := orchard.NewBerryTree(config)

    if err != nil {
      log.Println("Couldn't instantiate berry tree")
    }

    seed, _ := tree.Pick()

    if seed == nil {
      log.Println("No seed to pick")
      tree.ComeDown()

      time.Sleep(60*time.Second)
      continue
    }

    pot.Put(seed)

    tree.Snap(config, seed.Id())
    tree.ComeDown()

    time.Sleep(60 * time.Second)
  }
}

func main() {

  config := config.NewConfig(defaultConfigFile)

  addr := fmt.Sprintf(":%d", config.ServerConfig.Port())
  server := &http.Server{Addr: addr}
  listener, err := net.Listen("tcp", addr)
  if err != nil {
    log.Fatal(err)
  }

  // signal handler
  go func() {
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGTERM)
    <-stop

    log.Println("SIGTERM: Closing listener\n")
    listener.Close()
    return
  }()

  // Pot !
  pot, err := orchard.NewPot(config)
  if err != nil {
    log.Panic("Failed to make pot")
  }

  // Set Hodor loose and hand him a pot
  go hodor(config, pot)

  // gorilla mux for routing
  r := mux.NewRouter()

  r.HandleFunc("/eungyu/rss", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/xml")
    latest := pot.GetLatest()
    rss := blog.NewRss(latest)
    fmt.Fprintf(w, rss.Out())
  })

  r.HandleFunc("/eungyu/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    id, err := strconv.Atoi(vars["id"])
    if err != nil {
      http.NotFound(w, r)
      return
    }

    blog.Respond(config, w, pot.GetOne(id))
  })

  r.HandleFunc("/eungyu", func(w http.ResponseWriter, r *http.Request) {
    blog.Respond(config, w, pot.GetLatest())
  })

  // delegate control to the gorilla mux
  http.Handle("/", r)

  // Run HTTP Server
  server.Serve(listener)

  log.Println("Shutting down")
}
