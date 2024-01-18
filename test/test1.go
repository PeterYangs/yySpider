package main

import "github.com/PeterYangs/yySpider"

func main() {

	s := yySpider.NewYySpider()

	s.Host("https://www.925g.com")

	s.NewListPage("/gonglue/list_[PAGE].html", "body > div.ny-container.uk-background-default > div.wrap > div > div.commonLeftDiv.uk-float-left > div > div.bdDiv > div > ul > li", "a", 1, 20)

	s.Start()

}
