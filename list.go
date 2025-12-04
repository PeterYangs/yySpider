package yySpider

import "strings"

type ListPage struct {
	channel                   string
	listSelector              string
	hrefSelector              string
	pageStart                 int
	pageLength                int
	pageCurrent               int //当前分页
	hrefSelectorAttr          string
	fields                    map[string]Field //列表页面字段选择器
	hasNextPage               bool
	callback                  func(item map[string]string) bool
	y                         *YySpider
	requestListPrefixCallback func(listUrl string, currentIndex int)
	previousLinkCallback      func(listUrl string) string //下一页链接的回调
	previousStartPage         int
	previousMaxPage           int
}

func newListPage(y *YySpider, channel string, listSelector string, pageStart int, pageLength int) *ListPage {

	l := &ListPage{}

	l.SetChannel(channel)

	l.SetListSelector(listSelector)

	//l.SetHrefSelector(hrefSelector)

	l.SetPageStart(pageStart)

	l.SetPageLength(pageLength)

	//l.SetHrefSelectorAttr("")

	l.y = y

	return l
}

func (l *ListPage) SetChannel(channel string) *ListPage {

	l.channel = "/" + strings.TrimLeft(channel, "/")

	return l

}

func (l *ListPage) SetListSelector(listSelector string) *ListPage {

	l.listSelector = listSelector

	return l
}

//func (l *ListPage) SetHrefSelector(hrefSelector string) *ListPage {
//
//	l.hrefSelector = hrefSelector
//
//	return l
//
//}

func (l *ListPage) SetPageStart(pageStart int) *ListPage {

	l.pageStart = pageStart

	return l
}

func (l *ListPage) SetPageLength(pageLength int) *ListPage {

	l.pageLength = pageLength

	return l
}

//func (l *ListPage) SetHrefSelectorAttr(hrefSelector string) *ListPage {
//
//	if strings.TrimSpace(hrefSelector) == "" {
//
//		l.hrefSelector = "href"
//
//		return l
//	}
//
//	l.hrefSelector = hrefSelector
//
//	return l
//}

func (l *ListPage) SetFields(f map[string]Field) *ListPage {

	l.fields = f

	return l
}

func (l *ListPage) GetFields() map[string]Field {

	return l.fields
}

// SetNextPageLinkSelector 设置下一个page的入口
func (l *ListPage) SetNextPageLinkSelector(hrefSelector string, hrefSelectorAttr string) *ListPage {

	l.hasNextPage = true

	l.hrefSelector = hrefSelector

	l.hrefSelectorAttr = hrefSelectorAttr

	return l
}

// Callback 列表上的每个结果的回调
func (l *ListPage) Callback(callback func(item map[string]string)) *ListPage {

	l.callback = func(i map[string]string) bool {

		callback(i)

		return true

	}

	return l
}

// RequestListPrefixCallback 请求列表链接前的回调函数，listUrl是列表链接，currentIndex是计数器
func (l *ListPage) RequestListPrefixCallback(callback func(listUrl string, currentIndex int)) *ListPage {

	l.requestListPrefixCallback = callback

	return l
}

func (l *ListPage) CallbackWithBreak(callback func(item map[string]string) bool) *ListPage {

	l.callback = callback

	return l
}

func (l *ListPage) SetPreviousLinkCallback(callback func(listUrl string) string, startPage int, maxPage int) {
	l.previousLinkCallback = callback
	l.previousMaxPage = maxPage
	l.previousStartPage = startPage
}
