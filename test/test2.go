package main

import (
	"fmt"
	"gitee.com/mryy1996/yySpider"
	"golang.org/x/net/context"
)

func main() {

	//cxt, cancel := context.WithCancel(context.Background())

	//每个page需要一个入口
	//cancel()
	//_ = cancel
	s := yySpider.NewYySpider(context.Background())

	s.Host("https://www.yznnw.com")

	s.Header(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	})

	s.Debug()

	list := s.NewListPage(
		"/list/1/1.html",
		"ul.list_l2 > li",
		1,
		1,
	)

	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: "a"},
	})

	list.SetNextPageLinkSelector("", "href")

	//多级列表没有channel，是从上层page获取的链接
	list2 := s.NewListPage(
		"/list/1/1.html",
		"ul.list_l2 > li",
		1,
		1,
	)

	//detail := s.NewDetailPage()
	//
	//detail.SetFields(map[string]yySpider.Field{
	//	"title2": {Type: yySpider.Text, Selector: "h1"},
	//})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
