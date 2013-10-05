package blog

import (
  "fmt"
  "log"
  "net/http"
  "hodor/orchard"
  "hodor/config"
  "text/template"
)

func Respond(config *config.Config, w http.ResponseWriter, berries []*orchard.Berry) {
  basedir := config.ServerConfig.BaseDir()
  templateFile := fmt.Sprintf("%s/%s", basedir, config.PostConfig.Template())
  log.Println(templateFile)
  
  t, err := template.ParseFiles(templateFile)
  if err != nil {
    log.Println("Failed to parse index template")
  }

  capture := &Capture{}
  t.Execute(capture, berries)

  // todo cache this
  w.Header().Set("Content-Type", "text/html")
  fmt.Fprintf(w, "%s", capture.Render())
}