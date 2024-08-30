package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"strings"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	//设置域名
	s.Host("https://www.secretmine.net")

	//打开debug
	s.Debug()

	//第一个页面是列表
	list := s.NewListPage(
		"/tag/page_[PAGE]/", //列表规则,页码用[PAGE]代替
		"body > div.main > div.downlist.boxbg.lazy.clearfix > ul > li", //列表选择器
		1, //起始页码
		1, //采集长度
	)

	//列表采集
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: " div > b > a"}, //采集列表的标题，选择器是相对列表，比如列表选择器是ul li,这里的选择器就是从li开始找
		"size": {Type: yySpider.Text, Selector: " div > i:nth-child(3)", ConversionFunc: func(item string) string {
			return strings.Replace(item, "大小：", "", -1)
		}}, //ConversionFunc是转换器，item是采集到的结果，返回你需要的格式
	}).CallbackWithBreak(func(item map[string]string) bool {
		//数据过滤
		if item["title"] != "pvz2杂交版2.3版本" {
			return true
		}
		return false
	})

	//设置详情页的入口，这里的意思是，列表上的li下的 p > a的a链接是详情页，取href属性
	list.SetNextPageLinkSelector(" p > a", "href")

	//实例化详情页面
	detail := s.NewDetailPage()

	//跟列表配置一样
	detail.SetFields(map[string]yySpider.Field{
		"source": {Type: yySpider.Text, Selector: " #decimal_unm"},
	}).CallbackWithBreak(func(item map[string]string) bool {
		//数据过滤
		if item["source"] != "8.5" {
			return true
		}
		return false
	})

	//设置输出的xlsx文件路径
	s.SetXlsxName("xlsx/" + uuid.NewV4().String())

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
