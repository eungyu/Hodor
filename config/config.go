package config

import (
  "fmt"
  "os"
  "code.google.com/p/goconf/conf"
)

type Config struct {
  ImapConfig *imapConfig
  PostConfig *postConfig
  DbConfig   *dbConfig
  ServerConfig *serverConfig
}

func NewConfig(path string) *Config {
  _, err := os.Stat(path)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Path %s does not exist\n", path)
    return nil
  }

  conf, err := conf.ReadConfigFile(path)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Failed to read %s\n", path)
    return nil
  }

  imap   := NewImapConfig(conf)
  post   := NewPostConfig(conf)
  db     := NewDbConfig(conf)
  server := NewServerConfig(conf)

  c := &Config{ ImapConfig: imap, PostConfig: post, DbConfig: db, ServerConfig: server }
  return c
}


