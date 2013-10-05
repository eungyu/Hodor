package config

import (
  "code.google.com/p/goconf/conf"
)

type imapConfig struct {
  user string
  pass string
  server string
}

func NewImapConfig(conf *conf.ConfigFile)  *imapConfig {
  user, _   := conf.GetString("imap", "user")
  pass, _   := conf.GetString("imap", "pass")
  server, _ := conf.GetString("imap", "server")

  imap := &imapConfig {
    user:   user,
    pass:   pass,
    server: server }

  return imap
}

func (i *imapConfig) User() string {
  return i.user
}

func (i *imapConfig) Pass() string {
  return i.pass
}

func (i *imapConfig) Server() string {
  return i.server
}


