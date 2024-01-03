package yySpider

import "github.com/go-resty/resty/v2"

type YySpider struct {
	client *resty.Client
	host   string
	header map[string]string
}

func NewYySpider() *YySpider {

	client := resty.New()

	return &YySpider{client: client, header: make(map[string]string)}
}

func (y *YySpider) Host(host string) *YySpider {

	y.host = host

	return y
}

func (y *YySpider) Header(headers map[string]string) *YySpider {

	y.header = headers

	y.client.SetHeaders(headers)

	return y

}

func (y *YySpider) NewPage() {

}
