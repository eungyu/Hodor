package orchard

import (
  "bufio"
  "os"
  "regexp"
  "fmt"
  "bytes"
  "strings" 
  "unicode"
  "log"
  "image/jpeg"
)

type Soil struct {
  sprout string
}

func NewSoil(id int64, seed *Seed) *Soil {
  mode := seed.config.ServerConfig.Mode()  
  imgurl := seed.config.PostConfig.ImgUrl()
  basedir := seed.config.ServerConfig.BaseDir()

  imgcount := 0
  imgformat := "%d-%d.jpg"

  var content bytes.Buffer

  receipts := make(map[string]string)
 
  inQuote := false

  // refactor this
  for _, paragraph := range seed.Paragraphs() {
    isimg, _ := regexp.MatchString(imgRegEx, paragraph)
    if isimg {
      imgname := fmt.Sprintf(imgformat, id, imgcount)
      
      imgtagfmt := "<img src=\"http://lately.cc/eungyu%s/%s\">"
      if mode == "dev" {
        imgtagfmt = "<img src=\"%s/%s\">"
      }

      content.WriteString(enclose(fmt.Sprintf(imgtagfmt, imgurl, imgname)))
      receipts[paragraph] = imgname
      imgcount = imgcount + 1
    } else {
      if len(paragraph) < 1 {
        continue
      }

      if strings.HasPrefix(paragraph, blockquote) {
        paragraph = strings.TrimFunc(paragraph[1:], unicode.IsSpace)
        if !inQuote {
          content.WriteString("<blockquote>")
          inQuote = true
        }
      } else if inQuote {
        content.WriteString("</blockquote>")
        inQuote = false
      }
      content.WriteString(enclose(paragraph))
    }
    content.WriteString("\n")
  }

  // find a cleaner way
  if inQuote {
    content.WriteString("</blockquote>")
  }

  for cid, img := range seed.ImgMap() {
    name := fmt.Sprintf("%s%s/%s", basedir, imgurl, receipts[cid])
    log.Println(name)
    
    fo, _ := os.Create(name)
    w := bufio.NewWriter(fo)

    err := jpeg.Encode(w, img, nil)
    if err != nil {
      log.Fatal("Failed to write image")
    }
  }

  return &Soil{sprout:content.String()}
}

func (s *Soil) Sprout() string {
  return s.sprout
}
