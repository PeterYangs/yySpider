package yySpider

type DetailPage struct {
	fields   map[string]Field //列表页面字段选择器
	y        *YySpider
	callback func(item map[string]string) bool
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
