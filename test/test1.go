package main

import "github.com/PeterYangs/yySpider"

func main() {

	s := yySpider.NewYySpider()

	s.Host("https://www.925g.com")

	page := s.NewPage()

	_ = page

}
