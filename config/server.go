package config

import (
  "code.google.com/p/goconf/conf"
)

type serverConfig struct {
  mode string
  port int
  basedir string
}

func NewServerConfig(conf *conf.ConfigFile)  *serverConfig {
  port, _   := conf.GetInt("server", "port")
  mode, _   := conf.GetString("server", "mode")
  basedir, _ := conf.GetString("server", "basedir")

  server := &serverConfig { port: port, mode: mode, basedir: basedir }
  return server
}

func (server *serverConfig) Port() int {
  return server.port
}

func (server *serverConfig) Mode() string {
  return server.mode
}

func (server *serverConfig) BaseDir() string {
  return server.basedir
}
