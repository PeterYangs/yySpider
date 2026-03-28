package yySpider

import (
	"golang.org/x/net/context"
	"time"
)

type Page interface {
	GetHtmlCallback() func(htmlStr string, httpCode int, urls string)
	SetHtmlCallback(callback func(htmlStr string, httpCode int, urls string))
	SetWaitElement(selector string, timeout time.Duration)
	GetWaitElement() (string, time.Duration)
	SetChromedpBeforeCallback(callback func(ctx context.Context, htmlUrl string) error)
	GetChromedpBeforeCallback() func(ctx context.Context, htmlUrl string) error
	SetDownload(downloadKey string)
	GetDownloadKey() string
}
