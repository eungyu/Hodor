package orchard

import (
  "log"
  "net/mail"
  "hodor/util"
  "mime/multipart"
  "mime"
  "io"
  "io/ioutil"
  "bytes"
  "code.google.com/p/go.net/html"
  "code.google.com/p/go.net/html/atom"  
  "strings"
  "unicode"
  "image"
  "image/jpeg"
  "encoding/base64"
)

type Berry struct {
  message *mail.Message
  subject string
  body    []string
  imgmap  map[string]image.Image
}

func NewBerry(m *mail.Message) (*Berry, error) {

  printable := util.QuotedPrintable([]byte(m.Header.Get("Subject")))
  subject, err := printable.Decode()

  if err != nil {
    log.Println("Failed to identify subject")
    return nil, err
  }

  rawbody, imgmap, err := getParts(m)
  if err != nil {
    log.Println("Failed to identify body part")
    return nil, err
  }

  paragraphs := washUp(rawbody, func(token *html.Token) bool {
    if token.DataAtom != atom.Div {
      return false
    }
    
    if len(token.Attr) == 0 {
      return false
    }
    
    if token.Attr[0].Key == "class" && token.Attr[0].Val == "ennote" {
      return true
    }

    return false
  },
  func(token *html.Token, level int) bool {
    if token.DataAtom != atom.Div {
      return false
    }
    if level > 1 {
      return false
    }
    return true
  })

  return &Berry{ message : m, subject: subject, body: paragraphs, imgmap: imgmap }, nil
}

func (b * Berry) GetSubject() string {
  return b.subject
}

func (b *Berry) GetBody() []string {
  return b.body
}

func getParts(msg *mail.Message) ([]byte, map[string]image.Image, error) {
  _, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
  if err != nil {
    return nil, nil, err
  }

  boundary := params["boundary"]
  reader := multipart.NewReader(msg.Body, boundary)

  var body []byte
  imgmap := make(map[string]image.Image)

  for {
    part, err := reader.NextPart()    
    if err == io.EOF {
      break
    }

    if err != nil {
      return nil, nil, err
    }

    header := mail.Header(part.Header)
    mediatype, _, err := mime.ParseMediaType(header.Get("Content-Type"))
    if err != nil {
      continue
    }

    if mediatype == "text/html" {
      body, _ = ioutil.ReadAll(part)
    } else if mediatype == "image/jpeg" {
      img, err := jpeg.Decode(base64.NewDecoder(base64.StdEncoding, part))
      if err != nil {
        log.Println("Not a valid image")
        return nil, nil, err
      }

      contentid := part.Header.Get("Content-Id")
      cid := "cid:" + contentid[1:len(contentid)-1]

      imgmap[cid] = img
    }
  }

  return body, imgmap, nil
}

func getSrc(token *html.Token) string {
  if len(token.Attr) == 0 {
    return ""
  }

  for _, attr := range token.Attr {
    if attr.Key == "src" {
      return attr.Val
    }
  }

  return ""
}

func washUp(body []byte, startCond func(*html.Token) bool, paragraphCond func(*html.Token, int) bool)  []string {
  t := html.NewTokenizer(bytes.NewReader(body))

  effective := false
  taglevel := 0

  var buf bytes.Buffer

  paragraphs := []string{}

  for {
    tt := t.Next()
    if tt == html.ErrorToken {
      return paragraphs
    }

    token := t.Token()

    if !effective {
      if startCond(&token) {
        effective = true
      }
      continue
    }

    switch tt {

    case html.SelfClosingTagToken:
      if token.DataAtom == atom.Img {
        src := getSrc(&token)
        paragraphs = append(paragraphs, src)
      }

    case html.StartTagToken:
      taglevel = taglevel + 1

      if paragraphCond(&token, taglevel) {
        if buf.Len() > 0 {
          paragraphs = append(paragraphs, buf.String())
          buf.Reset()
        }
      }

    case html.EndTagToken:
      if paragraphCond(&token, taglevel) {
        if buf.Len() > 0 {
          paragraphs = append(paragraphs, buf.String())
          buf.Reset()
        }
      }

      if taglevel == 0 {
        return paragraphs
      }

      taglevel = taglevel - 1

    case html.TextToken:
      content := strings.TrimFunc(token.Data, unicode.IsSpace)
      if len(content) > 0 {
        if buf.Len() > 0 {
          buf.WriteString(" ")
        }
        buf.WriteString(content)
      }
    }
  }

  return paragraphs
}