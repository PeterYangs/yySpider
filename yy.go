package yySpider

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"strconv"
	"strings"
)

type YySpider struct {
	client   *resty.Client
	host     string
	header   map[string]string
	pageList []interface{}
}

func NewYySpider() *YySpider {

	client := resty.New()

	return &YySpider{client: client, header: make(map[string]string)}
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

			for listPage.pageCurrent = listPage.pageStart; listPage.pageCurrent < listPage.pageStart+listPage.pageLength; listPage.pageCurrent++ {

				listLink := y.host + strings.Replace(listPage.channel, "[PAGE]", strconv.Itoa(listPage.pageCurrent), -1)

				fmt.Println(listLink)

			}

		}

	}

}
