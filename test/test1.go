package main

import (
	"gitee.com/mryy1996/yySpider"
	"golang.org/x/net/context"
)

func main() {

	//cxt, cancel := context.WithCancel(context.Background())

	//每个page需要一个入口
	//cancel()
	//_ = cancel
	s := yySpider.NewYySpider(context.Background())

	s.Host("https://www.925g.com")

	s.NewListPage("/gonglue/list_[PAGE].html", "body > div.ny-container.uk-background-default > div.wrap > div > div.commonLeftDiv.uk-float-left > div > div.bdDiv > div > ul > li", "a", 1, 20)

	s.Start()

}
