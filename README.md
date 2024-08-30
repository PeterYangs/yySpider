### YySpider是一个简单的采集工具
<hr/>

声明：该爬虫仅供学习使用，如产生任何法律后果，本人概不负责

**安装**

```shell
go get github.com/PeterYangs/yySpider
```

**快速开始**
```go
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
		2, //采集长度
	)

	//列表采集
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: " div > b > a"}, //采集列表的标题，选择器是相对列表，比如列表选择器是ul li,这里的选择器就是从li开始找
		"size": {Type: yySpider.Text, Selector: " div > i:nth-child(3)", ConversionFunc: func(item string) string {
			return strings.Replace(item, "大小：", "", -1)
		}}, //ConversionFunc是转换器，item是采集到的结果，返回你需要的格式
	})

	//设置详情页的入口，这里的意思是，列表上的li下的 p > a的a链接是详情页，取href属性
	list.SetNextPageLinkSelector(" p > a", "href")

	//实例化详情页面
	detail := s.NewDetailPage()

	//跟列表配置一样
	detail.SetFields(map[string]yySpider.Field{
		"img": {Type: yySpider.Image, Selector: "body > div.comment_box.clearfix > div.down_infor_top > div > img"},
	})

	//设置输出的xlsx文件路径
	s.SetXlsxName("xlsx/" + uuid.NewV4().String())

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
```

**多页面采集**
<br/>
采集小说这种的多级页面
```go
package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	"golang.org/x/net/context"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	//设置域名
	s.Host("https://www.yznnw.com")

	//设置headers
	s.Headers(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	})

	//打开debug
	s.Debug()

	//第一个页面是小说列表页
	list := s.NewListPage(
		"/list/1/1.html",
		"ul.list_l2 > li",
		1,
		1,
	)

	//设置选择器
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: "a"},
	})

	//设置下一页入口
	list.SetNextPageLinkSelector("a", "href")

	//小说章节列表页面
	list2 := s.NewListPage(
		"",
		".section-list li",
		1,
		1,
	)

	//设置选择器
	list2.SetFields(map[string]yySpider.Field{
		"zhang_name": {Type: yySpider.Text, Selector: "a"},
	})

	//设置下一页入口
	list2.SetNextPageLinkSelector("a", "href")

	//详情页
	detail := s.NewDetailPage()

	detail.SetFields(map[string]yySpider.Field{
		"detail_title": {Type: yySpider.Text, Selector: ".chapter-title"},
	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
```

**自定义采集结果**
<br/>
结果不会生成到xlsx
```go
package main

import (
	"fmt"
	"github.com/PeterYangs/yySpider"
	"golang.org/x/net/context"
	"strings"
)

func main() {

	s := yySpider.NewYySpider(context.Background())

	//设置域名
	s.Host("https://www.secretmine.net")

	//打开debug
	//s.Debug()

	//第一个页面是列表
	list := s.NewListPage(
		"/tag/page_[PAGE]/", //列表规则,页码用[PAGE]代替
		"body > div.main > div.downlist.boxbg.lazy.clearfix > ul > li", //列表选择器
		1, //起始页码
		2, //采集长度
	)

	//列表采集
	list.SetFields(map[string]yySpider.Field{
		"title": {Type: yySpider.Text, Selector: " div > b > a"}, //采集列表的标题，选择器是相对列表，比如列表选择器是ul li,这里的选择器就是从li开始找
		"size": {Type: yySpider.Text, Selector: " div > i:nth-child(3)", ConversionFunc: func(item string) string {
			return strings.Replace(item, "大小：", "", -1)
		}}, //ConversionFunc是转换器，item是采集到的结果，返回你需要的格式
	})

	//设置详情页的入口，这里的意思是，列表上的li下的 p > a的a链接是详情页，取href属性
	list.SetNextPageLinkSelector(" p > a", "href")

	//实例化详情页面
	detail := s.NewDetailPage()

	//跟列表配置一样
	detail.SetFields(map[string]yySpider.Field{
		"img": {Type: yySpider.Image, Selector: "body > div.comment_box.clearfix > div.down_infor_top > div > img"},
	})

	//自行处理采集结果
	s.ResultCallback(func(item map[string]string) {

		fmt.Println(item)

	})

	err := s.Start()

	if err != nil {

		fmt.Println(err)

	}

}
```

**数据过滤**

```go
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
```

**字段类型**
```go
"title":         {Type: yySpider.Text, Selector: " div > b > a"},               //文本
"img":           {Type: yySpider.Image, Selector: " div > img"},                //单张图片
"imgs":          {Type: yySpider.MultipleImages, Selector: "div > img"},        //多张图片
"html":          {Type: yySpider.OnlyHtml, Selector: ".content"},               //富文本
"HtmlWithImage": {Type: yySpider.HtmlWithImage, Selector: ".content"},          //富文本带图片
"attr":          {Type: yySpider.Attr, Selector: ".content", AttrKey: "href"},  //元素属性
"attrs":         {Type: yySpider.Attrs, Selector: ".content", AttrKey: "href"}, //属性列表，如一个图片列表的所有图片链接
"fixed":         {Type: yySpider.Fixed, Selector: "固定内容"},                      //固定内容，Selector填什么就输出什么

```

