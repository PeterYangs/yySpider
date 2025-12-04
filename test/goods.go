package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	"golang.org/x/net/context"
	"time"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	s.SetProxy("http://127.0.0.1:7897")

	//设置域名
	s.Host("https://slickdeals.net")

	//设置headers
	s.Headers(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	})

	//打开debug
	s.Debug()

	//第一个页面是小说列表页
	list := s.NewListPage(
		"/deal-categories/",
		".featured-deals > li",
		1,
		1,
	)

	//设置选择器
	list.SetFields(map[string]yySpider.Field{
		"category_name": {Type: yySpider.Text, Selector: "a"},
	})

	////设置下一个page的入口
	list.SetNextPageLinkSelector("a", "href")

	//小说章节列表页面
	list2 := s.NewListPage(
		"",
		".bp-p-filterGrid_items li.bp-c-card",
		1,
		1,
	)

	//设置选择器
	list2.SetFields(map[string]yySpider.Field{
		//"title": {Type: yySpider.Text, Selector: ".bp-c-card_title"},
	})

	list2.Callback(func(item map[string]string) {

		time.Sleep(200 * time.Millisecond)

	})

	//下一页链接
	list2.SetPreviousLinkCallback(func(listUrl string) string {

		return listUrl + "?page=[PAGE]"
	}, 1, 12)

	//下一个Page选择器
	list2.SetNextPageLinkSelector("a.bp-c-link", "href")

	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		"title":          {Type: yySpider.Text, Selector: "h1.dealDetailsMainBlock__dealTitle"},
		"original_price": {Type: yySpider.Text, Selector: "h3.dealDetailsMainBlock__listPrice"},
		"price":          {Type: yySpider.Text, Selector: "h2.dealDetailsMainBlock__finalPrice"},
		"screenshots":    {Type: yySpider.Attrs, Selector: ".dealDetailsMainBlock__dealImageGalleryContainer .carousel__viewport > .carousel__track img", AttrKey: "src"},
		"content":        {Type: yySpider.OnlyHtml, Selector: ".dealDetailsRawHtml"},
		"store_link": {Type: yySpider.Attr, Selector: ".dealDetailsOutclickButton", AttrKey: "href", ConversionFunc: func(item string) string {

			sss, err := s.GetRedirectUrl(item)

			if err != nil {

				fmt.Println(err)
			}

			return sss
		}},
	})

	detail.Callback(func(item map[string]string) {

		time.Sleep(100 * time.Millisecond)
	})

	s.ResultCallback(func(item map[string]string) {

		fmt.Println(item["store_link"])

		//if  {
		//
		//}

	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
