# YySpider 新手快速上手

## 这是什么

`yySpider` 是一个偏“配置式”的 Go 采集库。

你不用自己从头写一整套 HTTP 请求、HTML 解析、翻页、详情页跟进、图片下载、Excel 导出这些流程，而是把：

- 站点域名
- 列表页规则
- 详情页规则
- 每个字段的选择器

配好之后，最后调用一个 `Start()`，库就会按你定义的流程往下跑。

如果你是第一次接触这个项目，可以先把它理解成一句话：

**你负责告诉它“去哪抓、抓什么、下一步去哪”，它负责帮你把流程串起来。**

---

## 这个项目能做什么

目前这个库已经支持下面这些能力：

- 抓普通网页 HTML
- 抓列表页
- 跟进详情页
- 多级页面串联采集
- 分页采集
- 采文本、属性、HTML、图片、文件、正则结果
- 自动下载图片
- 抓富文本并把里面的图片一起下载
- 抓浏览器渲染后的页面
- 浏览器里点击按钮、等待元素出现
- 捕获浏览器触发的下载链接
- 自定义结果回调
- 默认导出到 Excel
- 支持代理
- 支持自定义请求头
- 支持图片目录和图片下载规则配置

如果你只是要做“列表页 -> 详情页 -> 导出 Excel”，这个库已经够用了。

---

## 安装

```bash
go get github.com/PeterYangs/yySpider
```

建议 Go 版本尽量新一点。

---

## 先记住这 5 个核心概念

在开始写代码前，先把下面 5 个概念记住，后面会轻松很多。

### 1. `YySpider`

它是整个采集任务的总控制器。

你一般会先创建它：

```go
s := yySpider.NewYySpider(context.Background())
```

后面像设置域名、代理、浏览器模式、结果输出，都是挂在它上面的。

### 2. `ListPage`

表示“列表页”。

比如：

- 商品列表
- 文章列表
- 分类页
- 小说章节列表

你需要告诉它：

- 列表页 URL 规则
- 每一条列表项的选择器
- 当前列表项里要抓哪些字段
- 当前列表项里，详情页链接在哪里

### 3. `DetailPage`

表示“详情页”。

比如：

- 商品详情
- 文章详情
- 下载页

你需要告诉它：

- 详情页里要抓哪些字段

### 4. `Field`

一个字段就是一条你想抓的数据。

比如：

- 标题
- 价格
- 正文
- 图片
- 下载地址

每个字段至少要指定：

- 字段类型 `Type`
- 选择器 `Selector`

### 5. `Start()`

所有规则都配好后，真正开始跑任务的入口。

```go
err := s.Start()
```

---

## 最小可运行示例

如果你是第一次用，建议先从这个最小例子开始理解：

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/PeterYangs/yySpider"
	uuid "github.com/satori/go.uuid"
)

func main() {
	s := yySpider.NewYySpider(context.Background())

	// 1. 设置站点域名
	s.Host("https://www.secretmine.net")

	// 2. 打开调试输出，方便排查问题
	s.Debug()

	// 3. 定义列表页
	list := s.NewListPage(
		"/tag/page_[PAGE]/",
		"body > div.main > div.downlist.boxbg.lazy.clearfix > ul > li",
		1,
		2,
	)

	// 4. 定义列表页要抓的字段
	list.SetFields(map[string]yySpider.Field{
		"title": {
			Type:     yySpider.Text,
			Selector: "div > b > a",
		},
		"size": {
			Type:     yySpider.Text,
			Selector: "div > i:nth-child(3)",
			ConversionFunc: func(item string) string {
				return strings.ReplaceAll(item, "大小：", "")
			},
		},
	})

	// 5. 告诉库：详情页链接在哪
	list.SetNextPageLinkSelector("p > a", "href")

	// 6. 定义详情页
	detail := s.NewDetailPage()
	detail.SetFields(map[string]yySpider.Field{
		"img": {
			Type:     yySpider.Image,
			Selector: "body > div.comment_box.clearfix > div.down_infor_top > div > img",
		},
	})

	// 7. 设置输出 Excel 名称
	s.SetXlsxName("xlsx/" + uuid.NewV4().String())

	// 8. 启动
	if err := s.Start(); err != nil {
		fmt.Println(err)
	}
}
```

这个例子做了什么？

1. 打开一个站点。
2. 访问列表页。
3. 在列表页中找到每个 `li`。
4. 抓每个 `li` 里的标题和大小。
5. 再跟进这个 `li` 里的详情链接。
6. 在详情页抓图片。
7. 最后把结果输出到 Excel。

---

## 一次完整采集是怎么跑的

你可以把整个流程理解成下面这样：

1. 创建 `YySpider`
2. 设置基础参数，比如域名、请求头、代理
3. 添加第一个页面，通常是 `ListPage`
4. 给这个页面设置字段规则
5. 如果这个页面还能进入下一个页面，就设置“下一个页面链接”
6. 继续添加第二个页面、第三个页面
7. 最后调用 `Start()`
8. 结果进入 Excel，或者进入你自定义的回调函数

换句话说，`yySpider` 的核心思路不是“随便请求一个页面拿点数据”，而是：

**按页面链路，一层一层往下走。**

比如：

- 列表页 -> 详情页
- 分类页 -> 商品列表页 -> 商品详情页
- 小说目录页 -> 章节列表页 -> 章节正文页

---

## 最常用的 API，一次讲清楚

## 一、`YySpider` 级别配置

### `NewYySpider(context.Context)`

创建采集器。

```go
s := yySpider.NewYySpider(context.Background())
```

一般直接传 `context.Background()` 就行。

如果你想让任务支持超时、取消，也可以传带超时的 `context`。

---

### `Host(host string)`

设置站点域名。

```go
s.Host("https://example.com")
```

注意：

- 这里设置的是主域名
- 后面的 `NewListPage()` 一般只写相对路径
- 库内部会自动拼成完整地址

---

### `Headers(map[string]string)`

设置请求头。

```go
s.Headers(map[string]string{
	"user-agent": "Mozilla/5.0 ...",
	"referer":    "https://example.com",
})
```

常见用途：

- 设置 `User-Agent`
- 带 `Referer`
- 模拟浏览器请求
- 应对一些简单反爬

---

### `SetProxy(proxyUrl string)`

设置代理。

```go
s.SetProxy("http://127.0.0.1:7897")
```

适合：

- 需要科学上网
- 站点有地域限制
- 想把请求走到抓包代理里观察

---

### `Debug()`

打开调试模式。

```go
s.Debug()
```

打开后会输出一些运行信息，比如：

- 当前请求地址
- 选择器没找到
- 图片下载失败
- 页面等待信息

如果你是新手，我建议一开始先一直开着。

---

### `UseBrowserMode()`

启用浏览器模式，也就是用 `chromedp` 打开页面。

```go
s.UseBrowserMode()
```

什么时候要开？

- 页面是前端渲染的
- 直接请求拿不到真实内容
- 需要点击按钮后内容才出现
- 需要等待 JS 执行
- 需要捕获浏览器中的真实下载链接

什么时候不用开？

- 普通静态页面
- 直接 HTTP 请求就能拿到数据

能不用浏览器，就尽量先不用。因为浏览器模式更慢，也更重。

---

### `SetDeviceType(d DeviceType)`

设置浏览器模拟设备。

支持：

- `yySpider.DevicePC`
- `yySpider.DeviceAndroid`
- `yySpider.DeviceIPhone`

示例：

```go
s.UseBrowserMode()
s.SetDeviceType(yySpider.DeviceAndroid)
```

适合：

- 某些站点移动端更简单
- 移动端页面更容易拿到内容
- 某些下载站手机端逻辑不一样

---

### `SetXlsxName(name string)`

设置 Excel 输出文件名。

```go
s.SetXlsxName("xlsx/result")
```

最终会生成：

```text
xlsx/result.xlsx
```

如果你不设置，它会自动生成一个随机文件名。

---

### `ResultCallback(func(item map[string]string))`

自定义处理结果。

```go
s.ResultCallback(func(item map[string]string) {
	fmt.Println(item)
})
```

这个非常重要。

一旦你设置了 `ResultCallback`，结果就会走你的回调，**不会再默认写 Excel**。

适合：

- 你想自己存数据库
- 你想自己写 JSON
- 你只想打印看看结果

---

### `SetImageDir(path string)`

设置图片目录名。

```go
s.SetImageDir("images")
```

默认是：

```text
image
```

---

### `SetSavePath(path string)`

设置图片保存的根目录。

```go
s.SetSavePath("./data")
```

比如：

- `SetSavePath("./data")`
- `SetImageDir("images")`

那图片大致会落到：

```text
./data/images/...
```

---

### `SetLazyImageAttrName(name string)`

设置全局懒加载图片属性名。

```go
s.SetLazyImageAttrName("data-original")
```

有些站点图片不放在 `src`，而是放在：

- `data-src`
- `data-original`
- `data-lazy`

这时就需要设置它。

---

### `DisableAutoCoding()`

关闭自动转码。

默认情况下，库会尝试根据响应头自动处理编码。

如果你碰到某些站点转码反而出问题，可以试试：

```go
s.DisableAutoCoding()
```

---

### `SetDisableImageExtensionCheck(bool)`

关闭图片后缀检查。

```go
s.SetDisableImageExtensionCheck(true)
```

有些图片链接没有标准后缀，或者后缀不可信，这个时候可以用。

---

### `SetAllowImageExtension([]string)`

限制允许下载的图片后缀。

```go
s.SetAllowImageExtension([]string{"jpg", "jpeg", "png", "webp"})
```

适合你想明确控制图片格式时使用。

---

### `SetCustomDownloadFun(func(imgUrl, imgPath string) error)`

自定义图片下载逻辑。

```go
s.SetCustomDownloadFun(func(imgUrl string, imgPath string) error {
	// 你自己实现下载逻辑
	return nil
})
```

什么时候需要自己接管？

- 站点图片下载规则特殊
- 需要带鉴权
- 需要走自己的下载器
- 需要对接对象存储

---

### `GetRedirectUrl(u string)`

获取重定向后的最终地址。

```go
finalUrl, err := s.GetRedirectUrl(shortUrl)
```

这个很适合：

- 短链接跳转
- 跳转下载地址
- 电商外链跳转

---

### `If(condition, trueVal, falseVal)`

一个简单的三元表达式辅助函数。

```go
v := s.If(price != "", price, "暂无价格")
```

这个不是核心功能，只是一个小工具。

---

## 二、`ListPage` 怎么用

### `NewListPage(channel, listSelector, pageStart, pageLength)`

创建列表页。

```go
list := s.NewListPage(
	"/news/page_[PAGE].html",
	".news-list li",
	1,
	5,
)
```

4 个参数分别是：

1. 列表页地址规则
2. 列表项选择器
3. 起始页码
4. 抓多少页

其中 `[PAGE]` 是页码占位符。

比如：

```text
/news/page_[PAGE].html
```

实际运行时会变成：

- `/news/page_1.html`
- `/news/page_2.html`
- `/news/page_3.html`

---

### `SetFields(map[string]Field)`

给列表页定义要抓的字段。

```go
list.SetFields(map[string]yySpider.Field{
	"title": {
		Type:     yySpider.Text,
		Selector: "a.title",
	},
	"link": {
		Type:     yySpider.Attr,
		Selector: "a.title",
		AttrKey:  "href",
	},
})
```

这里有个很关键的点：

**列表页字段的选择器，是相对于每个列表项来的。**

也就是说，如果你的列表项是：

```css
.news-list li
```

那字段里的：

```css
a.title
```

会从当前这个 `li` 里面继续找，而不是从整页 HTML 顶部开始找。

---

### `SetNextPageLinkSelector(selector, attr)`

告诉列表页：下一层页面入口在哪。

```go
list.SetNextPageLinkSelector("a.title", "href")
```

这个“下一层页面”通常就是详情页链接。

如果不设置它，列表页抓完字段后就不会继续往下走。

---

### `Callback(func(item map[string]string))`

列表页每一项处理时的回调。

```go
list.Callback(func(item map[string]string) {
	fmt.Println(item)
})
```

适合：

- 调试当前抓到了什么
- 临时打印
- 加一点节奏控制

---

### `CallbackWithBreak(func(item map[string]string) bool)`

带布尔返回值的回调。

```go
list.CallbackWithBreak(func(item map[string]string) bool {
	if item["title"] == "" {
		return false
	}
	return true
})
```

返回值含义：

- `true`：继续处理
- `false`：当前这条数据跳过，不继续往下走

这个很适合做数据过滤。

---

### `RequestListPrefixCallback(func(listUrl string, currentIndex int))`

列表请求前回调。

```go
list.RequestListPrefixCallback(func(listUrl string, currentIndex int) {
	fmt.Println("准备请求：", listUrl)
})
```

适合：

- 打印日志
- 做请求前统计
- 动态观察当前分页

---

### `SetPreviousLinkCallback(func(listUrl string) string, startPage, maxPage int)`

这个名字有点绕，但作用可以理解为：

**在已有链接基础上，继续拼出分页规则。**

比如你先拿到一个分类页链接：

```text
/category/phone
```

然后你想把它扩展成：

```text
/category/phone?page=1
/category/phone?page=2
/category/phone?page=3
```

就可以这么写：

```go
list2.SetPreviousLinkCallback(func(listUrl string) string {
	return listUrl + "?page=[PAGE]"
}, 1, 3)
```

---

### `SetHtmlCallback(func(htmlStr string, httpCode int, url string))`

拿到原始 HTML 后的回调。

```go
list.SetHtmlCallback(func(htmlStr string, httpCode int, url string) {
	fmt.Println(url, len(htmlStr))
})
```

适合：

- 调试页面原始内容
- 保存 HTML 样本
- 排查选择器问题

---

### `SetWaitElement(selector, timeout)`

只在浏览器模式下生效。

```go
list.SetWaitElement(".news-list", 20*time.Second)
```

作用是：

- 打开页面后先等某个元素出现
- 元素出现后再继续抓

如果你不设置，当前版本会默认等待 `body`，默认超时 `15s`。

如果页面是 JS 渲染的，这个配置很常用。

---

### `SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error)`

只在浏览器模式下生效。

在抓 HTML 前，可以先操作页面。

比如：

- 点击弹窗关闭按钮
- 点击“展开”
- 点击下载按钮
- 先等待某个区域加载出来

示例：

```go
list.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {
	return chromedp.Click(".open-more", chromedp.ByQuery).Do(ctx)
})
```

---

### `SetDownload(downloadKey string)`

只在浏览器模式下有意义。

作用是捕获浏览器触发的下载地址，并把结果写进某个字段。

```go
detail.SetDownload("download_url")
```

最终结果里会多一个：

```text
download_url
```

---

## 三、`DetailPage` 怎么用

`DetailPage` 比 `ListPage` 简单很多。

它最常用的能力就是：

- `SetFields`
- `Callback`
- `CallbackWithBreak`
- `SetHtmlCallback`
- `SetWaitElement`
- `SetChromedpBeforeCallback`
- `SetDownload`

用法和 `ListPage` 基本一致，只是它没有分页和“下一页入口”的概念。

最常见写法：

```go
detail := s.NewDetailPage()

detail.SetFields(map[string]yySpider.Field{
	"title": {
		Type:     yySpider.Text,
		Selector: "h1",
	},
	"content": {
		Type:     yySpider.OnlyHtml,
		Selector: ".content",
	},
})
```

---

## `Field` 字段类型大全

这一部分很重要。很多人刚开始不会用，其实就是没把字段类型搞清楚。

## 1. `Text`

抓文本。

```go
"title": {
	Type:     yySpider.Text,
	Selector: "h1",
}
```

适合：

- 标题
- 价格
- 作者
- 时间

---

## 2. `Attr`

抓单个标签属性。

```go
"link": {
	Type:     yySpider.Attr,
	Selector: "a",
	AttrKey:  "href",
}
```

适合：

- 链接
- 图片地址
- 下载地址

---

## 3. `Attrs`

抓多个标签的属性列表，最后会拼成一个字符串。

```go
"imgs": {
	Type:     yySpider.Attrs,
	Selector: ".gallery img",
	AttrKey:  "src",
}
```

适合：

- 多张图片链接
- 多个下载地址

---

## 4. `Image`

抓单张图片，并下载到本地。

```go
"cover": {
	Type:     yySpider.Image,
	Selector: ".cover img",
}
```

结果保存的是本地图片路径，不是原始图片 URL。

---

## 5. `MultipleImages`

抓多张图片，并下载到本地。

```go
"gallery": {
	Type:     yySpider.MultipleImages,
	Selector: ".content img",
}
```

结果会是多个本地图片路径拼成的字符串。

---

## 6. `OnlyHtml`

抓 HTML，但不处理里面的图片下载。

```go
"content": {
	Type:     yySpider.OnlyHtml,
	Selector: ".article-content",
}
```

适合：

- 你只想保留 HTML 结构
- 不需要把图片下载到本地

---

## 7. `HtmlWithImage`

抓 HTML，同时把里面的图片下载到本地，并替换 HTML 里的图片地址。

```go
"content": {
	Type:     yySpider.HtmlWithImage,
	Selector: ".article-content",
}
```

这个很适合做文章采集、富文本迁移。

---

## 8. `Fixed`

固定值。

```go
"source": {
	Type:     yySpider.Fixed,
	Selector: "yySpider",
}
```

注意：这里的 `Selector` 不是选择器，而是你想返回的固定内容。

适合：

- 打标来源站点
- 写默认分类
- 填常量字段

---

## 9. `Regular`

用正则匹配。

```go
"price": {
	Type:         yySpider.Regular,
	Selector:     `"price":"([^"]+)"`,
	RegularIndex: 1,
}
```

适合：

- 页面中有一段 JSON
- 选择器不好拿
- 目标值藏在脚本里

---

## 10. `File`

下载文件。

```go
"pdf": {
	Type:     yySpider.File,
	Selector: ".download-btn",
	AttrKey:  "href",
}
```

适合：

- PDF
- 压缩包
- 安装包

---

## `Field` 里的高级配置

除了 `Type` 和 `Selector`，还有一些你后面会用到的配置。

### `AttrKey`

给 `Attr`、`Attrs`、`File` 这类字段用。

比如：

- `href`
- `src`
- `data-src`

---

### `ConversionFunc`

字段转换函数。

```go
"size": {
	Type:     yySpider.Text,
	Selector: ".size",
	ConversionFunc: func(item string) string {
		return strings.TrimSpace(strings.ReplaceAll(item, "大小：", ""))
	},
}
```

适合：

- 去掉前缀
- 去掉空格
- 转换单位
- 格式化文本

---

### `PrefixNotSelector`

前置剔除节点。

意思是：在解析字段前，先把某些节点从 HTML 里删掉。

适合：

- 删除广告块
- 删除无关区域

---

### `AfterNotSelector`

后置剔除节点。

意思是：先定位到当前字段对应的节点，再把节点里的某些子元素删掉。

适合：

- 只保留正文，不要“相关推荐”
- 抓文本前去掉按钮、脚注、广告

---

### `LazyImageAttrName`

字段级别的懒加载图片属性。

如果某个字段里的图片特殊，可以单独设置，而不是全局设置。

---

### `ImageDir`

给当前图片字段单独指定子目录。

支持的变量有：

- `[date:Y-m-d]`
- `[random:1-100]`
- `[singleField:title]`

这很适合按日期、随机数、某个字段值分类图片。

---

### `ImagePrefix`

设置图片路径前缀。

适合你想让最终返回的图片路径带一个自定义前缀时使用。

---

### `RegularIndex`

正则分组索引。

默认一般取第 1 组。

---

## 最常见的 5 种使用场景

## 场景 1：列表页 -> 详情页

最常见。

比如：

- 新闻列表 -> 新闻详情
- 商品列表 -> 商品详情

思路：

1. 先建 `ListPage`
2. 在列表里抓摘要信息
3. 设置详情链接
4. 再建 `DetailPage`
5. 在详情页抓正文、图片等

---

## 场景 2：多级页面

比如小说站：

- 小说列表
- 章节列表
- 正文章节

做法是连续添加多个页面：

```go
list1 := s.NewListPage(...)
list2 := s.NewListPage(...)
detail := s.NewDetailPage()
```

库会按添加顺序往后串。

---

## 场景 3：只想拿结果，不想导 Excel

用 `ResultCallback`。

```go
s.ResultCallback(func(item map[string]string) {
	fmt.Println(item)
})
```

---

## 场景 4：页面是 JS 渲染的

用浏览器模式：

```go
s.UseBrowserMode()
list.SetWaitElement(".list", 20*time.Second)
```

如果还要点击按钮：

```go
detail.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {
	return chromedp.Click(".load-more", chromedp.ByQuery).Do(ctx)
})
```

---

## 场景 5：要抓点击后的真实下载地址

用浏览器模式 + `SetDownload()`。

```go
detail.SetDownload("download_url")
```

然后在浏览器里触发下载动作，结果里就会出现这个字段。

---

## 浏览器模式怎么理解

很多新手第一次会被这里卡住。

简单说：

- 普通模式：直接发 HTTP 请求拿页面源码
- 浏览器模式：真的打开一个浏览器，再读取浏览器渲染后的 HTML

浏览器模式适合处理：

- 前端渲染页面
- 点击后才出现内容
- 需要滚动、等待、点击
- 需要捕获下载链接

### 浏览器模式常用组合

```go
s.UseBrowserMode()
s.SetDeviceType(yySpider.DeviceAndroid)
list.SetWaitElement(".list", 20*time.Second)
detail.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {
	return chromedp.Click(".btn", chromedp.ByQuery).Do(ctx)
})
```

### 浏览器模式下的等待规则

当前版本中：

- 如果你设置了 `SetWaitElement()`，就等待这个元素出现
- 如果你没设置，就默认等待 `body`
- 默认等待超时是 `15s`

所以建议：

- 关键页面尽量显式写 `SetWaitElement()`
- 不要完全依赖默认等待

---

## 结果是怎么输出的

默认情况下，结果会写到 Excel。

每条最终数据会合并成一个 `map[string]string`，然后写入一行。

### 默认输出

- 输出文件：`xxx.xlsx`
- 第一行是字段名
- 后面每一行是一条结果

### 自定义输出

如果你设置了：

```go
s.ResultCallback(func(item map[string]string) {
	fmt.Println(item)
})
```

那结果就由你自己处理，库不再自动写 Excel。

---

## 一个推荐的新手写法模板

如果你每次都不知道怎么起手，可以直接照这个模板改：

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PeterYangs/yySpider"
)

func main() {
	s := yySpider.NewYySpider(context.Background())

	s.Host("https://example.com")
	s.Debug()
	// s.Headers(map[string]string{"user-agent": "Mozilla/5.0 ..."})
	// s.SetProxy("http://127.0.0.1:7897")
	// s.UseBrowserMode()

	list := s.NewListPage(
		"/list_[PAGE].html",
		".list-item",
		1,
		2,
	)

	// 如果是浏览器渲染页，建议加等待
	list.SetWaitElement(".list-item", 15*time.Second)

	list.SetFields(map[string]yySpider.Field{
		"title": {
			Type:     yySpider.Text,
			Selector: ".title",
		},
	})

	list.SetNextPageLinkSelector("a", "href")

	detail := s.NewDetailPage()
	detail.SetFields(map[string]yySpider.Field{
		"title": {
			Type:     yySpider.Text,
			Selector: "h1",
		},
		"content": {
			Type:     yySpider.OnlyHtml,
			Selector: ".content",
		},
	})

	s.ResultCallback(func(item map[string]string) {
		fmt.Println(item)
	})

	if err := s.Start(); err != nil {
		fmt.Println("采集失败：", err)
	}
}
```

---

## 新手最容易踩的坑

## 1. `Host()` 没设置

这样相对路径没法正确拼接。

---

## 2. 选择器写成了整页选择器，但当前场景其实是列表项内部选择器

列表字段是在“每个列表项内部”查找的，不是在整页查找。

这个坑非常常见。

---

## 3. 页面是前端渲染的，却还在用普通模式

表现一般是：

- HTML 拿到了，但没有你想要的内容
- 选择器永远找不到

这时一般就要切到 `UseBrowserMode()`。

---

## 4. 浏览器模式没有设置等待元素

页面还没渲染完就开始抓，自然抓不到。

建议给关键页面加：

```go
SetWaitElement(selector, timeout)
```

---

## 5. 已经设置了 `ResultCallback`，却还在找 Excel 文件

只要你设置了 `ResultCallback`，结果就不会自动写 Excel。

---

## 6. 图片懒加载属性没处理

有些图片真实地址不在 `src`，而是在：

- `data-src`
- `data-original`

这时记得设置：

```go
s.SetLazyImageAttrName("data-original")
```

或者在字段里单独设置 `LazyImageAttrName`。

---

## 7. 站点链接是相对地址

这个库内部会帮你做一定的链接补全，但前提是你的 `Host()` 要正确。

---

## 调试建议

如果你抓不到数据，不要上来就怀疑库有问题，先按这个顺序排查：

1. 先开 `s.Debug()`
2. 看请求地址对不对
3. 用 `SetHtmlCallback()` 打印或保存 HTML
4. 在浏览器开发者工具里重新验证选择器
5. 判断页面是不是前端渲染
6. 如果是浏览器模式，加 `SetWaitElement()`
7. 如果要点击后才出现内容，加 `SetChromedpBeforeCallback()`

这是最快的排查路线。

---

## 什么时候该看 `test/` 目录

项目里的 `test/` 目录更像“示例集合”。

如果你想参考真实使用方式，建议看这些文件：

- `test/goods.go`
- `test/slickdeals.go`
- `test/test-289.go`
- `test/test-cssmoban.go`

它们分别覆盖了：

- 普通采集
- 多级列表
- 浏览器模式
- 捕获下载链接

---

## 最后给新手的建议

如果你第一次上手，不要一上来就做很复杂的站点。

建议按这个顺序学习：

1. 先跑一个普通静态列表页
2. 再加一个详情页
3. 再试 `ResultCallback`
4. 再试图片下载
5. 最后再试浏览器模式

这样最不容易乱。

如果你现在就想开始写自己的第一个站点采集，我建议你复制本文的“推荐模板”，先把：

- `Host()`
- `NewListPage()`
- `SetFields()`
- `SetNextPageLinkSelector()`
- `NewDetailPage()`
- `detail.SetFields()`
- `Start()`

这 7 步跑通。

只要这 7 步能通，后面再慢慢加代理、浏览器、下载链接、图片目录这些能力就行。
