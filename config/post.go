package config

import (
  "code.google.com/p/goconf/conf"
)

type postConfig struct {
  imgurl string
  template string
}

func NewPostConfig(conf *conf.ConfigFile)  *postConfig {
  imgurl, _  := conf.GetString("post", "imgurl")
  template, _ := conf.GetString("post", "template")

  post := &postConfig {
    imgurl: imgurl,
    template: template }

  return post
}

func (p* postConfig) ImgUrl() string {
  return p.imgurl 
}

func (p *postConfig) Template() string {
  return p.template
}