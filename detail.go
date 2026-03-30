package yySpider

import (
	"context"
	"time"
)

type DetailPage struct {
	fields                 map[string]Field //列表页面字段选择器
	y                      *YySpider
	callback               func(item map[string]string) bool
	htmlCallback           func(htmlStr string, httpCode int, url string)
	chromedpWaitSelector   string
	chromedpWaitTimeout    time.Duration
	chromedpBeforeCallback func(ctx context.Context, htmlUrl string) error
	downloadKey            string
}

func newDetailPage(y *YySpider) *DetailPage {

	return &DetailPage{y: y}
}

func (d *DetailPage) SetFields(f map[string]Field) *DetailPage {

	d.fields = f

	return d
}

func (d *DetailPage) Callback(callback func(item map[string]string)) *DetailPage {

	d.callback = func(i map[string]string) bool {

		callback(i)

		return true

	}

	return d
}

func (d *DetailPage) CallbackWithBreak(callback func(item map[string]string) bool) *DetailPage {

	d.callback = callback

	return d
}

func (d *DetailPage) GetHtmlCallback() func(htmlStr string, httpCode int, url string) {

	return d.htmlCallback
}

// SetHtmlCallback 原生html的回调
func (d *DetailPage) SetHtmlCallback(callback func(htmlStr string, httpCode int, url string)) {

	d.htmlCallback = callback
}

func (d *DetailPage) SetWaitElement(selector string, timeout time.Duration) {

	d.chromedpWaitSelector = selector
	d.chromedpWaitTimeout = timeout
}

func (d *DetailPage) GetWaitElement() (string, time.Duration) {

	return d.chromedpWaitSelector, d.chromedpWaitTimeout
}

// SetChromedpBeforeCallback Chromedp前置操作（如点击弹窗之类的）
func (d *DetailPage) SetChromedpBeforeCallback(callback func(ctx context.Context, htmlUrl string) error) {

	d.chromedpBeforeCallback = callback
}

func (d *DetailPage) GetChromedpBeforeCallback() func(ctx context.Context, htmlUrl string) error {

	return d.chromedpBeforeCallback
}

func (d *DetailPage) SetDownload(downloadKey string) {

	d.downloadKey = downloadKey
}

func (d *DetailPage) GetDownloadKey() string {

	return d.downloadKey
}
