package yySpider

type DetailPage struct {
	fields map[string]Field //列表页面字段选择器
	y      *YySpider
	//hasNextPage bool
}

func newDetailPage(y *YySpider) *DetailPage {

	return &DetailPage{y: y}
}

func (d *DetailPage) SetFields(f map[string]Field) *DetailPage {

	d.fields = f

	return d
}
