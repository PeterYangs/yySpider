package yySpider

type Page interface {
	GetHtmlCallback() func(htmlStr string, httpCode int, urls string)
	SetHtmlCallback(callback func(htmlStr string, httpCode int, urls string))
}
