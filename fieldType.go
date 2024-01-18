package yySpider

type FieldType int

const (
	Text           FieldType = 0x00000 //单个字段
	Image          FieldType = 0x00002 //单个图片
	OnlyHtml       FieldType = 0x00003 //普通html(不包括图片)
	HtmlWithImage  FieldType = 0x00004 //html包括图片
	MultipleImages FieldType = 0x00005 //多图
	Attr           FieldType = 0x00006 //标签属性选择器
	Fixed          FieldType = 0x00007 //固定数据，填什么返回什么,选择器就是返回的数据
	Regular        FieldType = 0x00008 //正则（FindStringSubmatch,返回一个结果）
	File           FieldType = 0x00009 //文件类型
	Attrs          FieldType = 0x00010 //属性列表，如一个图片列表的所有图片链接
)
