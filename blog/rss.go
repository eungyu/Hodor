package blog

import (
  "time"
  "encoding/xml"
  "fmt"
  "hodor/orchard"
)

const ns = "http://www.w3.org/2005/Atom"

type Feed struct {
  Title string
  Link  string
}

type entrySummary struct {
  Content string `xml:",chardata"`
  Type    string `xml:"type,attr"`
}

type linkXml struct {
  XMLName xml.Name `xml:"link"`
  Href    string   `xml:"href,attr"`
  Rel     string   `xml:"rel,attr"`
}

type feedXml struct {
  XMLName xml.Name `xml:"feed"`
  Ns      string   `xml:"xmlns,attr"`
  Title   string   `xml:"title"`
  Id      string   `xml:"id"`
  Name    string   `xml:"author>name"`
  Link    *linkXml
  Updated string   `xml:"updated"`
  Entries []*entryXml
}

type entryXml struct {
  XMLName xml.Name `xml:"entry""`
  Title   string   `xml:"title"`
  Link    *linkXml
  Updated string   `xml:"updated"`
  Id      string   `xml:"id"`
  Summary *entrySummary `xml:"summary"`
}

func newEntry(berry *orchard.Berry) *entryXml {
  id := fmt.Sprintf("http://lately.cc/eungyu/%d", berry.Id)

  content := ""
  for _, paragraph := range berry.Body {
    content = content + "<p>" + paragraph + "</p>"
  }

  s := &entrySummary{Content: content, Type: "html"}
  x := &entryXml{
    Title:  berry.Subject,
    Link: &linkXml{Href: id, Rel: "alternate"},
    Summary: s,
    Id: id,
    Updated: berry.Created.Format(time.RFC3339)}
  return x
}

type Rss struct {
  out      string
}

func generateFeed(berries []*orchard.Berry) string {
  feed := &feedXml{
    Ns:       ns,
    Title:    "lately.cc/eungyu",
    Name:     "Eun-Gyu Kim",
    Id:       "http://lately.cc/eungyu",
    Updated:  time.Now().Format(time.RFC3339),
    Link:     &linkXml{Href: "http://lately.cc/eungyu", Rel: "self"}}

  for _, berry := range berries {
    feed.Entries = append(feed.Entries, newEntry(berry))
  }

  data, err := xml.MarshalIndent(feed,"  ", "  ")
  if err != nil {
    return ""
  }

  s := xml.Header[:len(xml.Header)-1] + string(data) + "\n"
  return s  
}

func NewRss(berries []*orchard.Berry) *Rss {
  s := generateFeed(berries)
  r := &Rss{out:s}
  return r
}

func (r *Rss) Out() string {
  return r.out
}

func (r *Rss) Reload(berries []*orchard.Berry) {
  s := generateFeed(berries)
  r.out = s
}
