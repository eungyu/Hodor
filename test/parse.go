package main

import (
  "log"
  "strings"
  "io/ioutil"
  "bytes"
  "code.google.com/p/go.net/html"
  "code.google.com/p/go.net/html/atom"    
)

func main() {
  rawbody, err := ioutil.ReadFile("mail-06.txt")
  if err != nil {
    log.Println("Failed to open file")
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
  },
  func(token *html.Token) bool {
    if token.DataAtom == atom.A {
      return true
    }

    return false
  })

  log.Println(paragraphs)
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

func washUp(body []byte, startCond func(*html.Token) bool, paragraphCond func(*html.Token, int) bool, allowed func(*html.Token) bool)  []string {  
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

      if allowed(&token) {
        buf.WriteString(token.String())
      }

    case html.EndTagToken:
      if paragraphCond(&token, taglevel) {
        if buf.Len() > 0 {
          paragraphs = append(paragraphs, buf.String())
          buf.Reset()
        }
      }

      if allowed(&token) {
        buf.WriteString(token.String())
      }

      if taglevel == 0 {
        return paragraphs
      }

      taglevel = taglevel - 1

    case html.TextToken:
      log.Println(token.Data)
      content := token.Data
      if len(strings.TrimSpace(content)) < 1 {
        continue
      }
      buf.WriteString(strings.Trim(content, "\n"))
    }
  }

  return paragraphs
}
