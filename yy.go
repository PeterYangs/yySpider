package yySpider

import (
	"fmt"
	"github.com/PeterYangs/tools"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"golang.org/x/net/context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type YySpider struct {
	client            *resty.Client
	host              string
	header            map[string]string
	pageList          []interface{}
	cxt               context.Context
	disableAutoCoding bool
	debug             bool
	res               *sync.Map
}

func NewYySpider(cxt context.Context) *YySpider {

	client := resty.New()

	return &YySpider{client: client, header: make(map[string]string), cxt: cxt, res: &sync.Map{}}
}

func (y *YySpider) Host(host string) *YySpider {

	y.host = strings.TrimRight(host, "/")

	return y
}

func (y *YySpider) Header(headers map[string]string) *YySpider {

	y.header = headers

	y.client.SetHeaders(headers)

	return y

}

func (y *YySpider) DisableAutoCoding() *YySpider {

	y.disableAutoCoding = true

	return y

}

func (y *YySpider) NewListPage(channel string, listSelector string, hrefSelector string, pageStart int, pageLength int) *ListPage {

	list := NewListPage(y, channel, listSelector, hrefSelector, pageStart, pageLength)

	y.pageList = append(y.pageList, list)

	return list
}

func (y *YySpider) Start() {

	for _, item := range y.pageList {

		switch item.(type) {

		case *ListPage:

			listPage := item.(*ListPage)

		FOR:
			for listPage.pageCurrent = listPage.pageStart; listPage.pageCurrent < listPage.pageStart+listPage.pageLength; listPage.pageCurrent++ {

				select {

				case <-y.cxt.Done():

					break FOR

				default:

				}

				listLink := y.host + strings.Replace(listPage.channel, "[PAGE]", strconv.Itoa(listPage.pageCurrent), -1)

				//fmt.Println(listLink)

				y.getList(listLink, listPage)

			}

		}

	}

}

func (y *YySpider) getList(listUrl string, listPage *ListPage) {

	html, err := y.requestHtml(listUrl)

	if err != nil {

		fmt.Println(err.Message)

		return

	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find(listPage.listSelector).EachWithBreak(func(i int, selection *goquery.Selection) bool {

		href := ""

		isFind := false

		if strings.TrimSpace(listPage.hrefSelector) == "" {

			href, isFind = selection.Attr(listPage.hrefSelectorAttr)

		} else {

			href, isFind = selection.Find(listPage.hrefSelector).Attr(listPage.hrefSelectorAttr)

		}

		if len(listPage.GetFields()) > 0 {

			listItem, listItemErr := goquery.OuterHtml(selection)

			if listItemErr != nil {

				y.Debug(listItemErr.Error(), listUrl, listPage.listSelector)

				return false
			}

		}

		return true
	})

}

func (y *YySpider) requestHtml(htmlUrl string) (string, *SpiderError) {

	rsp, err := y.client.R().Get(htmlUrl)

	if err != nil {

		return "", NewSpiderError(HtmlRequestError, err.Error(), htmlUrl)

	}

	h := rsp.String()

	var ee *SpiderError

	if y.disableAutoCoding == false {

		html, e := y.DealCoding(rsp.String(), rsp.Header())

		if e != nil {

			ee = NewSpiderError(HtmlCodeError, "html转码失败", htmlUrl)

			return "", ee

		}

		h = html

	}

	return h, nil

}

// DealCoding 解决编码问题
func (y *YySpider) DealCoding(html string, header http.Header) (string, error) {

	headerContentType_ := header["Content-Type"]

	if len(headerContentType_) > 0 {

		headerContentType := headerContentType_[0]

		charset := y.GetCharsetByContentType(headerContentType)

		charset = strings.ToLower(charset)

		switch charset {

		case "gbk":

			return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

		case "gb2312":

			return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

		case "utf-8":

			return html, nil

		case "utf8":

			return html, nil

		case "euc-jp":

			return string(tools.ConvertToByte(html, "euc-jp", "utf8")), nil

		case "":

			break

		default:
			return string(tools.ConvertToByte(html, charset, "utf8")), nil

		}

	}

	code, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {

		return html, err
	}

	contentType, _ := code.Find("meta[charset]").Attr("charset")

	//转小写
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	switch contentType {

	case "gbk":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "gb2312":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "utf-8":

		return html, nil

	case "utf8":

		return html, nil

	case "euc-jp":

		return string(tools.ConvertToByte(html, "euc-jp", "utf8")), nil

	case "":

		break
	default:
		return string(tools.ConvertToByte(html, contentType, "utf8")), nil

	}

	contentType, _ = code.Find("meta[http-equiv=\"Content-Type\"]").Attr("content")

	charset := y.GetCharsetByContentType(contentType)

	switch charset {

	case "utf-8":

		return html, nil

	case "utf8":

		return html, nil

	case "gbk":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "gb2312":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "euc-jp":

		return string(tools.ConvertToByte(html, "euc-jp", "utf8")), nil

	case "":

		break

	default:
		return string(tools.ConvertToByte(html, charset, "utf8")), nil

	}

	return html, nil
}

// GetCharsetByContentType 从contentType中获取编码
func (y *YySpider) GetCharsetByContentType(contentType string) string {

	contentType = strings.TrimSpace(strings.ToLower(contentType))

	//捕获编码
	r, _ := regexp.Compile(`charset=([^;]+)`)

	re := r.FindAllStringSubmatch(contentType, 1)

	if len(re) > 0 {

		c := re[0][1]

		return c

	}

	return ""
}

// Debug debug信息输出
func (y *YySpider) Debug(msg string, link string, selector string) {

	if y.debug {

		str := msg + " "

		if link != "" {

			str += "链接：" + link + " "
		}

		if selector != "" {

			str += "选择器：" + selector + " "
		}

		fmt.Println(str)

	}

}

// ResolveSelector 解析选择器
func (y *YySpider) ResolveSelector(html string, selector map[string]Field, originUrl string) (*sync.Map, error) {

	//存储结果
	var res = &sync.Map{}

	var wait = &sync.WaitGroup{}

	var globalErr error = nil

	//goquery加载html
	htmlDoc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {

		return nil, err

	}

	//解析详情页面选择器
	for fieldT, itemT := range selector {

		doc := htmlDoc

		//前置剔除选择器
		for _, s := range itemT.PrefixNotSelector {

			doc.Find(s).Remove()
		}

		field := fieldT

		item := itemT

		switch item.Type {

		//单个文字字段
		case Text:

			selectors := doc.Find(item.Selector)

			//排除选择器
			for _, s := range item.AfterNotSelector {

				selectors.Find(s).Remove()

			}

			v := strings.TrimSpace(selectors.Text())

			res.Store(field, v)

			break

		//单个元素属性
		case Attr:

			v := ""

			if strings.TrimSpace(item.Selector) == "" {

				v, _ = doc.Attr(item.AttrKey)

			} else {

				v, _ = doc.Find(item.Selector).Attr(item.AttrKey)
			}

			res.Store(field, strings.TrimSpace(v))

			break

		//多个元素属性
		case Attrs:

			var v []string

			doc.Find(item.Selector).Each(func(i int, selection *goquery.Selection) {

				ss, ok := selection.Attr(item.AttrKey)

				if ok {

					v = append(v, ss)
				}

			})

			res.Store(field, tools.Join(",", v))

			break

		//只爬html（不包括图片）
		case OnlyHtml:

			selectors := doc.Find(item.Selector)

			//排除选择器
			for _, s := range item.AfterNotSelector {

				selectors.Find(s).Remove()

			}

			v, sErr := selectors.Html()

			if sErr != nil {

				res.Store(field, "")

				y.Debug("获取onlyHtml失败："+err.Error(), originUrl, item.Selector)

				break

			}

			res.Store(field, v)

			break

		//爬取html，包括图片
		case HtmlWithImage:

			wait.Add(1)

			go func(_item Field, field string) {

				defer wait.Done()

				selectors := doc.Find(_item.Selector)

				//排除选择器
				for _, s := range item.AfterNotSelector {

					selectors.Find(s).Remove()

				}

				html_, sErr := selectors.Html()

				if sErr != nil {

					//f.s.notice.Error(sErr.Error()+",源链接："+originUrl, ",选择器：", _item.Selector)
					//
					//globalErr = sErr

					return

				}

				htmlImg, err := goquery.NewDocumentFromReader(strings.NewReader(html_))

				if err != nil {

					f.s.notice.Error(err.Error() + ",源链接：" + originUrl)

					globalErr = err

					return

				}

				var waitImg sync.WaitGroup

				var imgList = sync.Map{}

				htmlImg.Find("img").Each(func(i int, selection *goquery.Selection) {

					img, err := f.getImageLink(selection, _item, originUrl)

					if err != nil {

						f.s.notice.Error(err.Error()+",源链接："+originUrl, ",富文本内容")

						globalErr = err

						return
					}

					waitImg.Add(1)

					go func(waitImg *sync.WaitGroup, imgList *sync.Map, __item Field) {

						defer waitImg.Done()

						imgName, e := f.DownImg(img, __item, res)

						if e != nil {

							f.s.notice.Error(e.Error()+",源链接："+originUrl, ",富文本图片下载失败", "图片地址", img)

						}

						globalErr = e

						imgList.Store(imgName, img)

					}(&waitImg, &imgList, _item)

				})

				waitImg.Wait()

				html_, _ = htmlImg.Html()

				imgList.Range(func(key, value interface{}) bool {

					html_ = strings.Replace(html_, value.(string), key.(string), -1)

					return true
				})

				res.Store(field, html_)

			}(item, field)

		//单个图片
		case Image:

			wait.Add(1)

			go func(_item Field, field string) {

				defer wait.Done()

				imgUrl, err := f.getImageLink(doc.Find(_item.Selector), _item, originUrl)

				if err != nil {

					f.s.notice.Error(err.Error()+",源链接："+originUrl, ",选择器：", _item.Selector)

					globalErr = err

					return
				}

				imgName, e := f.DownImg(imgUrl, _item, res)

				globalErr = e

				if e != nil {

					f.s.notice.Error(e.Error()+",源链接："+originUrl, ",选择器：", _item.Selector, "图片地址", imgUrl)
				}

				res.Store(field, imgName)

			}(item, field)

			break

		//单个文件
		case File:

			selectors := doc.Find(item.Selector)

			v, ok := selectors.Attr(item.AttrKey)

			if !ok {

				break
			}

			imgName, e := f.DownImg(v, item, res)

			globalErr = e

			res.Store(field, imgName)

			//res.Store(field, v)

		//多个图片
		case MultipleImages:

			wait.Add(1)

			go func(_item Field, field string) {

				defer wait.Done()

				var waitImg sync.WaitGroup

				var imgList = sync.Map{}

				doc.Find(_item.Selector).Each(func(i int, selection *goquery.Selection) {

					imgUrl, err := f.getImageLink(selection, _item, originUrl)

					if err != nil {

						f.s.notice.Error(err.Error()+",源链接："+originUrl, ",选择器：", _item.Selector)

						globalErr = err

						return
					}

					waitImg.Add(1)

					go func(waitImg *sync.WaitGroup, imgList *sync.Map, __item Field) {

						defer waitImg.Done()

						imgName, e := f.DownImg(imgUrl, __item, res)

						if e != nil {

							f.s.notice.Error(e.Error()+",源链接："+originUrl, ",选择器：", _item.Selector, "图片地址", imgUrl)

						}

						globalErr = e

						imgList.Store(imgName, "")

					}(&waitImg, &imgList, _item)

				})

				waitImg.Wait()

				var strArray []string

				imgList.Range(func(key, value interface{}) bool {

					strArray = append(strArray, key.(string))

					return true
				})

				array := tools.Join(",", strArray)

				res.Store(field, array)

			}(item, field)

		//固定数据
		case Fixed:

			res.Store(field, item.Selector)

		//正则
		case Regular:

			reg := regexp.MustCompile(item.Selector).FindStringSubmatch(html)

			if len(reg) > 0 {

				index := 1

				if item.RegularIndex != 0 {

					index = item.RegularIndex
				}

				res.Store(field, reg[index])

			}

			globalErr = errors.New("正则匹配未找到")

			f.s.notice.Error("正则匹配未找到")

		}

	}

	wait.Wait()

	arr := make(map[string]string)

	res.Range(func(key, value interface{}) bool {

		arr[key.(string)] = value.(string)

		return true

	})

	r := NewRows(arr)

	r.err = globalErr

	return r, nil

}
