package orchard

import (
  "time"
  "bytes"
  "net/mail"
  "errors"
  "hodor/config"
  "code.google.com/p/go-imap/go1/imap"
)

type berryTree struct {
  config *config.Config
  client *imap.Client
}

func NewBerryTree(config *config.Config) (*berryTree, error) {  
  c, err := imap.DialTLS("imap.zoho.com",nil)
  if err != nil {
    return nil, err
  }

  c.Data = nil

  // Enable encryption, if supported by the server
  if c.Caps["STARTTLS"] {
    c.StartTLS(nil)
  }

  imapConfig := config.GetImapConfig()

  // Authenticate
  if c.State() == imap.Login {
    _, err := c.Login(imapConfig.GetUser(), imapConfig.GetPassword())
    if err != nil {
      return nil, err
    }
  }

  _, err = c.Select("INBOX", true)
  if err != nil {
    return nil, err
  }

  b := &berryTree {
    config: config,
    client: c }

  return b, nil
}

func (b *berryTree) Pick() (*Berry, error) {

  c := b.client
  defer c.Logout(10 * time.Second)

  cmd, err := c.Send("SEARCH", "UNSEEN")
  if err != nil {
    return nil, err
  }

  messages := []uint32{}

  for cmd.InProgress() {
    err = c.Recv(-1)

    for _, rsp := range cmd.Data {
      result := rsp.SearchResults()
      messages = append(messages, result...)
    }

    cmd.Data = nil
    c.Data = nil
  }

  if _, err := cmd.Result(imap.OK); err != nil {
    if err == imap.ErrAborted {
      return nil, err
    } else {
      return nil, err
    }
  }

  msgSeq, _ := imap.NewSeqSet("")
  for _, uid := range messages {
    msgSeq.AddNum(uid)
  }

  if len(messages) > 0 {
    cmd, _ = c.Fetch(msgSeq, "RFC822")
    for cmd.InProgress() {
      c.Recv(-1)
      for _, rsp := range cmd.Data {
        header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822"])
        if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {          
          return NewBerry(msg)
        }
      }

      cmd.Data = nil
      c.Data = nil
    }
  }

  return nil, errors.New("No new messages")
}