package config

import (
  "code.google.com/p/goconf/conf"
)

const section = "imap"

type imapConfig struct {
  user string
  pass string
  server string
}

func NewImapConfig(conf *conf.ConfigFile)  *imapConfig {

  user,   _ := conf.GetString(section, "user")
  passwd, _ := conf.GetString(section, "pass")
  server, _ := conf.GetString(section, "server")

  imap := &imapConfig {
    user: user,
    pass: passwd,
    server: server }

  return imap
}

func (i *imapConfig) GetUser() string {
  return i.user
}

func (i *imapConfig) GetPassword() string {
  return i.pass
}

func (i *imapConfig) GetServer() string {
  return i.server
}