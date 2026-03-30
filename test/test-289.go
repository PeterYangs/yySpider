package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/yySpider"
	"github.com/chromedp/chromedp"
	"time"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	//无头浏览器模式
	s.UseBrowserMode()

	//设置域名
	s.Host("https://www.289.com")

	//打开debug（无头浏览器模式下会打开网页）
	s.Debug()

	//列表
	list := s.NewListPage(
		"/azrj/list-161-[PAGE].html",
		".m-downlistul li",
		1,
		1,
	)

	//等待元素出现（只在无头浏览器模式下生效）
	list.SetWaitElement(".m-downlistul", 30*time.Second)

	//设置选择器
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: ".m-tit"}, //分类名称
	})

	//设置下一个page的入口
	list.SetNextPageLinkSelector("p.m-tit > a", "href")

	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		"download_url": {Type: yySpider.Attr, Selector: ".address_like > a", AttrKey: "href"},
	})

	//等待元素出现（只在无头浏览器模式下生效）
	detail.SetWaitElement(".m-goabtn", 10*time.Second)

	//获取html前操作浏览器（只在无头浏览器模式下生效）
	detail.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {

		err := chromedp.Click(".m-goabtn", chromedp.ByQuery).Do(ctx)

		if err != nil {
			return err
		}

		ctx2, _ := context.WithTimeout(ctx, 5*time.Second)

		return chromedp.WaitVisible(".address_like", chromedp.ByQuery).Do(ctx2)

	})

	detail.SetHtmlCallback(func(htmlStr string, httpCode int, url string) {

		time.Sleep(800 * time.Millisecond)
	})

	s.ResultCallback(func(item map[string]string) {

		fmt.Println(item)

	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
