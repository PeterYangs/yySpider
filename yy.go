package yySpider

import (
	"github.com/PeterYangs/tools"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"golang.org/x/net/context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type YySpider struct {
	client            *resty.Client
	host              string
	header            map[string]string
	pageList          []interface{}
	cxt               context.Context
	disableAutoCoding bool
}

func NewYySpider(cxt context.Context) *YySpider {

	client := resty.New()

	return &YySpider{client: client, header: make(map[string]string), cxt: cxt}
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

	list := NewListPage(channel, listSelector, hrefSelector, pageStart, pageLength)

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

				y.getList(listLink)

			}

		}

	}

}

func (y *YySpider) getList(listUrl string) {

}

func (y *YySpider) getHtml(htmlUrl string) (string, *SpiderError) {

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
