package orchard

import (
  "time"
  "bufio"
  "os"
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
  "hodor/config"
  "regexp"
  "fmt"
)

type Seed struct {
  id uint32
  subject string
  paragraphs []string
  imgmap  map[string]image.Image
  config *config.Config
}

func NewSeed(id uint32, subject string, paragraphs []string, imgmap map[string]image.Image, config *config.Config) (*Seed, error) {
  return &Seed{ id: id, subject: subject, paragraphs: paragraphs, imgmap: imgmap, config: config }, nil
}

func (s *Seed) Id() uint32 {
  return s.id
}

func (s *Seed) Subject() string {
  return s.subject
}

func (s *Seed) Paragraphs() []string {
  return s.paragraphs
}

func (s *Seed) ImgMap() map[string]image.Image {
  return s.imgmap
}

func (s *Seed) Config() *config.Config {
  return s.config
}

func enclose(paragraph string) string {
  return "<p>" + paragraph + "</p>"
}

// Grow processes Seed and produce a fully grown Berry
func (s *Seed) Grow(lastid int64) (*Berry, error) {
  mode := s.config.ServerConfig.Mode()  
  imgurl := s.config.PostConfig.ImgUrl()
  basedir := s.config.ServerConfig.BaseDir()

  imgcount := 0
  imgformat := "%d-%d.jpg"

  var content bytes.Buffer

  receipts := make(map[string]string)

  // refactor this
  for _, paragraph := range s.Paragraphs() {
    isimg, _ := regexp.MatchString(imgRegEx, paragraph)
    if isimg {
      imgname := fmt.Sprintf(imgformat, lastid, imgcount)
      
      imgtagfmt := "<img src=\"http://lately.cc/eungyu%s/%s\">"
      if mode == "dev" {
        imgtagfmt = "<img src=\"%s/%s\">"
      }

      content.WriteString(enclose(fmt.Sprintf(imgtagfmt, imgurl, imgname)))
      receipts[paragraph] = imgname
      imgcount = imgcount + 1
    } else {
      content.WriteString(enclose(paragraph))
    }
    content.WriteString("\n")
  }

  for cid, img := range s.ImgMap() {
    name := fmt.Sprintf("%s%s/%s", basedir, imgurl, receipts[cid])
    log.Println(name)
    fo, _ := os.Create(name)
    w := bufio.NewWriter(fo)

    err := jpeg.Encode(w, img, nil)
    if err != nil {
      log.Fatal("Failed to write image")
    }
  }  

  return NewBerry(lastid, s.subject, content.String(), time.Now().UTC() )
}

func ExtractSeedFromMail(id uint32, m *mail.Message, config *config.Config) (*Seed, error) {

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
    
    for _, attr := range token.Attr {
      if attr.Key == "class" && attr.Val == "ennote" {
        return true
      }
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

  return NewSeed(id, subject, paragraphs, imgmap, config)
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