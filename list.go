package yySpider

import "strings"

type ListPage struct {
	channel      string
	listSelector string
	hrefSelector string
	pageStart    int
	pageLength   int
	pageCurrent  int //当前分页
}

func NewListPage(channel string, listSelector string, hrefSelector string, pageStart int, pageLength int) *ListPage {

	l := &ListPage{}

	l.SetChannel(channel)

	l.SetListSelector(listSelector)

	l.SetHrefSelector(hrefSelector)

	l.SetPageStart(pageStart)

	l.SetPageLength(pageLength)

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

func (l *ListPage) SetHrefSelector(hrefSelector string) *ListPage {

	l.hrefSelector = hrefSelector

	return l

}

func (l *ListPage) SetPageStart(pageStart int) *ListPage {

	l.pageStart = pageStart

	return l
}

func (l *ListPage) SetPageLength(pageLength int) *ListPage {

	l.pageLength = pageLength

	return l
}
