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
		"#ctbar-ctbarw > div.uk-background-default > div > div > div > div.commonLeftDiv.uk-float-left > div > div.bdDiv > div > ul > li",
		1,
		1,
	)

	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: "a > div > span"},
	})

	//列表上每个结果回调
	list.Callback(func(item map[string]string) {

		fmt.Println(item)
	})

	//列表抓取前回调
	list.RequestListPrefixCallback(func(listUrl string, currentIndex int) {

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
