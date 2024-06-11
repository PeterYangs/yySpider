package yySpider

type Field struct {
	Type              FieldType
	Selector          string                        //字段选择器
	PrefixNotSelector []string                      //前置剔除选择器(意思是先剔除html的节点)
	AfterNotSelector  []string                      //剔除选择器(后置选择器，意思是先获取该item的doc再剔除节点)
	AttrKey           string                        //属性值参数
	LazyImageAttrName string                        //懒加载图片属性，默认为data-original
	ImageDir          string                        //图片子文件夹，支持变量 1.[date:Y-m-d] 2.[random:1-100] 3.[singleField:title]
	ImagePrefix       func(imageName string) string //图片路径前缀,会添加到图片路径前缀，但不会生成文件夹
	RegularIndex      int                           //正则匹配中的反向引用的下标，默认是1
	ConversionFunc    func(item string) string      //转换格式函数,第一个参数是该字段数据
}
