package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	"golang.org/x/net/context"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	//设置域名
	s.Host("https://www.weidown.com")

	//设置headers
	s.Headers(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	})

	//打开debug
	s.Debug()

	//第一个页面是小说列表页
	list := s.NewListPage(
		"/android/list_[PAGE].html",
		" div.articleWrapper > ul",
		1,
		2,
	)

	//设置选择器
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: "h2"},
	})

	//设置下一页入口
	list.SetNextPageLinkSelector("a", "href")

	//详情页
	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		"detail_title": {Type: yySpider.Text, Selector: "h1"},
	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
