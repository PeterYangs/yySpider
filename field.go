package yySpider

type Field struct {
	Type              FieldType
	Selector          string   //字段选择器
	PrefixNotSelector []string //前置剔除选择器(意思是先剔除html的节点)
	AfterNotSelector  []string //剔除选择器(后置选择器，意思是先获取该item的doc再剔除节点)
	AttrKey           string   //属性值参数
}
