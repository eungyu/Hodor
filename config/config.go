package config

import (
  "fmt"
  "os"
  "code.google.com/p/goconf/conf"
)

type Config struct {
  imap *imapConfig
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

  imap := NewImapConfig(conf)
  c := &Config{ imap: imap }
  return c
}

func (c *Config) GetImapConfig() *imapConfig {
  return c.imap
}

