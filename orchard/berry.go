package orchard

import (
  "time"
)

type Berry struct {
  Id int64
  Subject string
  Content string
  Created time.Time
}

func NewBerry(id int64, subject string, content string, created time.Time) (*Berry, error) {
  // do some processing here
  return &Berry{ Id:id, Subject: subject, Content: content, Created: created }, nil
}

