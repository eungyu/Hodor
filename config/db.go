package config

import (
  "code.google.com/p/goconf/conf"
)

type dbConfig struct {
  file string
}

func NewDbConfig(conf *conf.ConfigFile)  *dbConfig {
  file, _   := conf.GetString("db", "file")

  db := &dbConfig { file: file }
  return db
}

func (db *dbConfig) File() string {
  return db.file
}
