package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	"golang.org/x/net/context"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	//设置域名
	s.Host("http://www.anfensi.com")

	//设置headers
	s.Headers(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	})

	//打开debug
	s.Debug()

	//第一个页面是小说列表页
	list := s.NewListPage(
		"/game/r_1_[PAGE].html",
		"#listCont dd p",
		1,
		50,
	)

	//设置选择器
	list.SetFields(map[string]yySpider.Field{
		//"pp": {Type: yySpider.Text, Selector: "a"},
	})

	//设置下一页入口
	list.SetNextPageLinkSelector("a", "href")

	//小说章节列表页面
	list2 := s.NewListPage(
		"",
		".klist",
		1,
		1,
	)

	//设置选择器
	list2.SetFields(map[string]yySpider.Field{
		//"zhang_name": {Type: yySpider.Text, Selector: "a"},
	})

	//设置下一页入口
	list2.SetNextPageLinkSelector("a", "href")

	//详情页
	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: ".topics-top .soft-title"},
		"image": {Type: yySpider.Attr, Selector: ".topics-top .fl-lf img", AttrKey: "src"},
		"desc":  {Type: yySpider.Text, Selector: ".soft .introduction"},
	})

	//s.SetSavePath("collect/anfensi-game.xlsx")

	s.SetXlsxName("collect/anfensi-game.xlsx")

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
