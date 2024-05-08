package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	"golang.org/x/net/context"
)

func main() {

	//cxt, cancel := context.WithCancel(context.Background())

	//每个page需要一个入口
	//cancel()
	//_ = cancel
	s := yySpider.NewYySpider(context.Background())

	s.Host("https://www.925g.com")

	s.Debug()

	list := s.NewListPage(
		"/gonglue/list_[PAGE].html",
		"body > div.ny-container.uk-background-default > div.wrap > div > div.commonLeftDiv.uk-float-left > div > div.bdDiv > div > ul > li",
		1,
		20,
	)

	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: "a > div > span"},
	})

	list.SetNextPageLinkSelector("a", "href")

	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		"title2": {Type: yySpider.Text, Selector: "h1"},
	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
