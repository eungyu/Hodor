package orchard

import (
  "log"
  "strings"
  "bytes"
  "net/mail"
  "errors"
  "time"
  "hodor/config"
  "code.google.com/p/go-imap/go1/imap"
)

type berryTree struct {
  config *config.Config
  client *imap.Client
}

func NewBerryTree(config *config.Config) (*berryTree, error) {  
  imapConfig := config.ImapConfig

  c, err := imap.DialTLS(imapConfig.Server(),nil)
  if err != nil {
    return nil, err
  }

  c.Data = nil

  // Enable encryption, if supported by the server
  if c.Caps["STARTTLS"] {
    c.StartTLS(nil)
  }

  // Authenticate
  if c.State() == imap.Login {
    _, err := c.Login(imapConfig.User(), imapConfig.Pass())
    if err != nil {
      return nil, err
    }
  }

  _, err = c.Select("INBOX", false)
  if err != nil {
    return nil, err
  }

  b := &berryTree {
    config: config,
    client: c }

  return b, nil
}

func (b *berryTree) ComeDown() {
  b.client.Logout(1 * time.Second)
}

func (b *berryTree) Snap(config *config.Config, uid int64) error {
  c := b.client

  msgSeq, _ := imap.NewSeqSet("")
  msgSeq.AddNum(uint32(uid))

  _, err := c.Store(msgSeq, "+FLAGS", "(\\Seen)")
  if err != nil {
    log.Println(err)
    return err
  }

  return nil
}

func (b *berryTree) Pick() (*Berry, error) {

  c := b.client
 
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
  if len(messages) < 1 {
    return nil, errors.New("No new messages")
  }

  uid := messages[0]
  msgSeq.AddNum(uid)

  if len(messages) > 0 {
    cmd, _ = c.Fetch(msgSeq, "RFC822")
    for cmd.InProgress() {
      c.Recv(-1)
      for _, rsp := range cmd.Data {
        header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822"])
        if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {          
          recipient := msg.Header.Get("To")
          if strings.Contains(recipient, "+dev") && !(b.config.ServerConfig.Mode() == "dev") {            
            log.Println("Skipping dev entry")
            continue;
          }
          return NewBerry(int64(uid), msg)
        }
      }

      cmd.Data = nil
      c.Data = nil
    }
  }

  return nil, errors.New("No new messages")
}