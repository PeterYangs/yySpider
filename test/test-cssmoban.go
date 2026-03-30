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

	s.SetDeviceType(yySpider.DeviceAndroid)

	//设置域名
	s.Host("https://m.cssmoban.com")

	//打开debug（无头浏览器模式下会打开网页）
	s.Debug()

	//列表
	list := s.NewListPage(
		"/app/",
		".app_soft_bd li",
		1,
		1,
	)

	//等待元素出现（只在无头浏览器模式下生效）
	list.SetWaitElement("#app_hot_list_view", 30*time.Second)

	//设置选择器
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: "h3"}, //分类名称
	})

	//设置下一个page的入口
	list.SetNextPageLinkSelector("a", "href")

	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		//"download_url": {Type: yySpider.Attr, Selector: ".address_like > a", AttrKey: "href"},
		"title2": {Type: yySpider.Text, Selector: "h1"}, //分类名称
	})

	//等待元素出现（只在无头浏览器模式下生效）
	detail.SetWaitElement("h1", 10*time.Second)

	//捕获下载链接（结果会出现在结果中）
	detail.SetDownload("download_url")

	//获取html前操作浏览器（只在无头浏览器模式下生效）
	detail.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {

		ctx2, _ := context.WithTimeout(ctx, 5*time.Second)

		eee := chromedp.WaitVisible("#downlinkaddress", chromedp.ByQuery).Do(ctx2)

		if eee != nil {

			fmt.Println(eee)

			return eee
		}

		time.Sleep(3 * time.Second)

		err := chromedp.Click("#downlinkaddress", chromedp.ByQuery).Do(ctx)

		if err != nil {

			fmt.Println(err)
			return err
		}

		return nil
	})

	detail.SetHtmlCallback(func(htmlStr string, httpCode int, url string) {

	})

	s.ResultCallback(func(item map[string]string) {
		time.Sleep(1200 * time.Millisecond)
		fmt.Println(item)

	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
