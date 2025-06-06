package yySpider

import (
	"fmt"
	"github.com/PeterYangs/tools"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/phpisfirstofworld/image"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type YySpider struct {
	client                     *resty.Client
	host                       string
	header                     map[string]string
	pageList                   []interface{}
	cxt                        context.Context
	disableAutoCoding          bool
	debug                      bool
	lazyImageAttrName          string                                    //懒加载图片属性，默认为data-original(全局设置，field里面有一个局部设置)
	savePath                   string                                    //图片保存文件夹，不会出现在图片路径中，为空则为当前运行路径
	imageDir                   string                                    //图片文件夹
	disableImageExtensionCheck bool                                      //禁用图片拓展名检查，禁用后所有图片拓展名强制为png
	allowImageExtension        []string                                  //允许下载的图片拓展名
	customDownloadFun          func(imgUrl string, imgPath string) error //自实现图片下载
	imageResizePercent         int                                       //图片缩放百分比
	imageResizeByte            int64                                     //图片超过此设置的大小就执行图片缩放，单位字节
	lock                       sync.Mutex
	hasGetExcelHeader          bool
	excelKeyArray              []string
	excelSheetIndex            int
	excelIndex                 int
	excelFile                  *excelize.File
	xlsxName                   string                       //自定义xlsx文件名
	resultCallback             func(item map[string]string) //自定义获取采集结果回调
	proxyUrl                   string
}

func NewYySpider(cxt context.Context) *YySpider {

	//client := resty.New()

	y := &YySpider{header: make(map[string]string), cxt: cxt, imageDir: "image", lock: sync.Mutex{}}

	y.client = y.httpInit()

	return y
}

func (y *YySpider) SetProxy(proxyUrl string) *YySpider {

	y.proxyUrl = proxyUrl

	return y
}

func (y *YySpider) Host(host string) *YySpider {

	y.host = strings.TrimRight(host, "/")

	return y
}

func (y *YySpider) Headers(headers map[string]string) *YySpider {

	y.header = headers

	y.client.SetHeaders(headers)

	return y

}

func (y *YySpider) Debug() *YySpider {

	y.debug = true

	return y
}

// SetImageDir 设置图片文件夹
func (y *YySpider) SetImageDir(path string) *YySpider {

	y.imageDir = path

	return y

}

func (y *YySpider) SetLazyImageAttrName(lazyImageAttrName string) *YySpider {

	y.lazyImageAttrName = lazyImageAttrName

	return y
}

// SetSavePath 图片保存文件夹，不会出现在图片路径中，为空则为当前运行路径
func (y *YySpider) SetSavePath(path string) *YySpider {

	y.savePath = path

	return y
}

func (y *YySpider) DisableAutoCoding() *YySpider {

	y.disableAutoCoding = true

	return y

}

func (y *YySpider) SetDisableImageExtensionCheck(b bool) *YySpider {

	y.disableImageExtensionCheck = b

	return y
}

func (y *YySpider) SetAllowImageExtension(allow []string) *YySpider {

	y.allowImageExtension = allow

	return y
}

func (y *YySpider) SetCustomDownloadFun(f func(imgUrl string, imgPath string) error) *YySpider {

	y.customDownloadFun = f

	return y
}

func (y *YySpider) NewListPage(channel string, listSelector string, pageStart int, pageLength int) *ListPage {

	list := newListPage(y, channel, listSelector, pageStart, pageLength)

	y.pageList = append(y.pageList, list)

	return list
}

func (y *YySpider) NewDetailPage() *DetailPage {

	detail := newDetailPage(y)

	y.pageList = append(y.pageList, detail)

	return detail

}

func (y *YySpider) SetXlsxName(name string) *YySpider {

	y.xlsxName = strings.TrimRight(name, ".xlsx")

	return y

}

func (y *YySpider) ResultCallback(f func(item map[string]string)) *YySpider {

	y.resultCallback = f

	return y
}

func (y *YySpider) Start() error {

	y.excelInit()

	res := make(map[string]string)

	if len(y.pageList) <= 0 {

		return errors.New("page数量为0")
	}

	y.dealPage("", 0, res)

	if y.resultCallback == nil {

		xlsxName := y.generateXlsxName()

		dir := filepath.Dir(xlsxName)

		os.MkdirAll(dir, 0755)

		err := y.excelFile.SaveAs(xlsxName)

		if err != nil {

			return err
		}

		log.Println(xlsxName)

	}

	return nil

}

func (y *YySpider) generateXlsxName() string {

	xlsxName := y.xlsxName

	if xlsxName == "" {

		xlsxName = uuid.NewV4().String()
	}

	return xlsxName + ".xlsx"

}

func (y *YySpider) dealRes(res map[string]string) {

	defer func() {

		y.excelIndex++
	}()

	if y.resultCallback != nil {

		y.resultCallback(res)

		return
	}

	sheetName := y.excelFile.GetSheetName(y.excelSheetIndex)

	if !y.hasGetExcelHeader {

		y.lock.Lock()

		var keyArr []string

		for s, _ := range res {

			keyArr = append(keyArr, s)
		}

		y.excelKeyArray = keyArr

		y.hasGetExcelHeader = true

		//设置表头
		for i, s := range y.excelKeyArray {

			y.excelFile.SetCellValue(sheetName, y.getColumnNameByIndex(i+1)+"1", s)
		}

		y.excelIndex++

		y.lock.Unlock()

	}

	for s, s2 := range res {

		y.excelFile.SetCellValue(sheetName, y.getCell(s)+strconv.Itoa(y.excelIndex+1), s2)

	}

	y.debugMsg(res, "", "")

}

func (y *YySpider) getCell(key string) string {

	for i, s := range y.excelKeyArray {

		if s == key {

			return y.getColumnNameByIndex(i + 1)
		}

	}

	return "A"

}

func (y *YySpider) getColumnNameByIndex(index int) string {
	if index <= 0 {
		return ""
	}
	var columnName string
	for index > 0 {
		index--
		columnName = string(rune((index%26)+'A')) + columnName
		index /= 26
	}
	return columnName
}

func (y *YySpider) excelInit() {

	f := excelize.NewFile()

	// Create a new sheet.
	index, err := f.NewSheet("Sheet1")
	if err != nil {

		y.debugMsg(err.Error(), "", "")

		return
	}

	y.excelSheetIndex = index

	y.excelFile = f

}

func (y *YySpider) httpInit() *resty.Client {
	client := resty.New()

	client.SetTimeout(60 * time.Second)

	if y.proxyUrl != "" {

		client.SetProxy(y.proxyUrl)

	}

	return client

}

func (y *YySpider) dealPage(link string, currentIndex int, res map[string]string) error {

	//超出下标，开始处理结果
	if currentIndex+1 > len(y.pageList) {

		y.dealRes(res)

		return nil
	}

	page := y.pageList[currentIndex]

	switch page.(type) {

	case *ListPage:

		listPage := page.(*ListPage)

		if link != "" {

			err := y.getList(link, listPage, res, currentIndex)

			if err != nil {

				y.debugMsg(err.Error(), link, "")

			}

		} else {

		FOR:
			for listPage.pageCurrent = listPage.pageStart; listPage.pageCurrent < listPage.pageStart+listPage.pageLength; listPage.pageCurrent++ {

				select {

				case <-y.cxt.Done():

					break FOR

				default:

				}

				listLink := y.host + strings.Replace(listPage.channel, "[PAGE]", strconv.Itoa(listPage.pageCurrent), -1)

				err := y.getList(listLink, listPage, res, currentIndex)

				if err != nil {

					y.debugMsg(err.Error(), listLink, "")

				}

			}
		}

	case *DetailPage:

		detailPage := page.(*DetailPage)

		y.getDetail(link, detailPage, res, currentIndex)

	}

	return nil
}

func (y *YySpider) mergeRes(res1 map[string]string, res2 map[string]string) map[string]string {

	for s, s2 := range res1 {

		res2[s] = s2
	}

	return res2

}

// currentIndex是计数器
func (y *YySpider) getList(listUrl string, listPage *ListPage, res map[string]string, currentIndex int) error {

	if listPage.requestListPrefixCallback != nil {
		//执行前置回调
		listPage.requestListPrefixCallback(listUrl, currentIndex)
	}

	html, err := y.requestHtml(listUrl)

	if err != nil {

		return err

	}

	if y.debug {

		fmt.Println(listUrl)
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	listSize := doc.Find(listPage.listSelector).Size()

	if listSize == 0 {

		y.debugMsg("列表选择器未找到", listUrl, listPage.listSelector)
	}

	doc.Find(listPage.listSelector).EachWithBreak(func(i int, selection *goquery.Selection) bool {

		href := ""

		isFind := false

		if listPage.hasNextPage {

			if strings.TrimSpace(listPage.hrefSelector) == "" {

				href, isFind = selection.Attr(listPage.hrefSelectorAttr)

			} else {

				href, isFind = selection.Find(listPage.hrefSelector).Attr(listPage.hrefSelectorAttr)

			}

			if !isFind {

				y.debugMsg("下一页选择器未找到", listUrl, listPage.listSelector+" "+listPage.hrefSelectorAttr)

			} else {

				href = y.getHref(href)
			}

		}

		if len(listPage.GetFields()) > 0 {

			listItem, listItemErr := goquery.OuterHtml(selection)

			if listItemErr != nil {

				y.debugMsg(listItemErr.Error(), listUrl, listPage.listSelector)

				return false
			}

			res2, resErr := y.resolveSelector(listItem, listPage.fields, listUrl)

			if resErr != nil {

				return false

			}

			if listPage.callback != nil {

				isContinue := listPage.callback(res2)

				if !isContinue {

					y.debugMsg("数据过滤", listUrl, listPage.listSelector)

					return true
				}

			}

			res3 := y.mergeRes(res2, res)

			y.dealPage(href, currentIndex+1, res3)

		} else {

			y.dealPage(href, currentIndex+1, res)

		}

		return true
	})

	return nil
}

func (y *YySpider) getDetail(detailUrl string, detailPage *DetailPage, res map[string]string, currentIndex int) error {

	html, err := y.requestHtml(detailUrl)

	if err != nil {

		return err

	}

	res2, resErr := y.resolveSelector(html, detailPage.fields, detailUrl)

	if resErr != nil {

		return resErr
	}

	if detailPage.callback != nil {

		isContinue := detailPage.callback(res2)

		if !isContinue {

			y.debugMsg("数据过滤", detailUrl, "")

			return nil
		}

	}

	res3 := y.mergeRes(res2, res)

	y.dealPage("", currentIndex+1, res3)

	return nil

}

func (y *YySpider) requestHtml(htmlUrl string) (string, error) {

	rsp, err := y.client.R().Get(htmlUrl)

	if err != nil {

		return "", errors.New(err.Error() + ",url:" + htmlUrl)
		//return "", NewSpiderError(HtmlRequestError, err.Error(), htmlUrl)

	}

	h := rsp.String()

	//var ee *SpiderError

	if y.disableAutoCoding == false {

		html, e := y.dealCoding(rsp.String(), rsp.Header())

		if e != nil {

			//ee = NewSpiderError(HtmlCodeError, "html转码失败", htmlUrl)

			return "", errors.New(e.Error() + ",url:" + htmlUrl)

		}

		h = html

	}

	return h, nil

}

// dealCoding 解决编码问题
func (y *YySpider) dealCoding(html string, header http.Header) (string, error) {

	headerContentType_ := header["Content-Type"]

	if len(headerContentType_) > 0 {

		headerContentType := headerContentType_[0]

		charset := y.getCharsetByContentType(headerContentType)

		charset = strings.ToLower(charset)

		switch charset {

		case "gbk":

			return tools.Convert(html, "gbk", "utf8")

		case "gb2312":

			return tools.Convert(html, "gbk", "utf8")

		case "utf-8":

			return html, nil

		case "utf8":

			return html, nil

		case "euc-jp":

			return tools.Convert(html, "euc-jp", "utf8")

		case "":

			break

		default:
			return tools.Convert(html, charset, "utf8")

		}

	}

	code, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {

		return html, err
	}

	contentType, _ := code.Find("meta[charset]").Attr("charset")

	//转小写
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	switch contentType {

	case "gbk":

		return tools.Convert(html, "gbk", "utf8")

	case "gb2312":

		return tools.Convert(html, "gbk", "utf8")

	case "utf-8":

		return html, nil

	case "utf8":

		return html, nil

	case "euc-jp":

		return tools.Convert(html, "euc-jp", "utf8")

	case "":

		break
	default:
		return tools.Convert(html, contentType, "utf8")

	}

	contentType, _ = code.Find("meta[http-equiv=\"Content-Type\"]").Attr("content")

	charset := y.getCharsetByContentType(contentType)

	switch charset {

	case "utf-8":

		return html, nil

	case "utf8":

		return html, nil

	case "gbk":

		return tools.Convert(html, "gbk", "utf8")

	case "gb2312":

		return tools.Convert(html, "gbk", "utf8")

	case "euc-jp":

		return tools.Convert(html, "euc-jp", "utf8")

	case "":

		break

	default:
		return tools.Convert(html, charset, "utf8")

	}

	return html, nil
}

// getCharsetByContentType 从contentType中获取编码
func (y *YySpider) getCharsetByContentType(contentType string) string {

	contentType = strings.TrimSpace(strings.ToLower(contentType))

	//捕获编码
	r, _ := regexp.Compile(`charset=([^;]+)`)

	re := r.FindAllStringSubmatch(contentType, 1)

	if len(re) > 0 {

		c := re[0][1]

		return c

	}

	return ""
}

// debugMsg debug信息输出
func (y *YySpider) debugMsg(msg any, link string, selector string) {

	if y.debug {

		str := fmt.Sprintln(msg) + " "

		if link != "" {

			str += "链接：" + link + " "
		}

		if selector != "" {

			str += "选择器：" + selector + " "
		}

		fmt.Println(str)

	}

}

// resolveSelector 解析选择器
func (y *YySpider) resolveSelector(html string, selector map[string]Field, originUrl string) (map[string]string, error) {

	//存储结果
	var res = &sync.Map{}

	var wait = &sync.WaitGroup{}

	var globalErr error = nil

	//goquery加载html
	htmlDoc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {

		return nil, err

	}

	//解析详情页面选择器
	for fieldT, itemT := range selector {

		doc := htmlDoc

		//前置剔除选择器
		for _, s := range itemT.PrefixNotSelector {

			doc.Find(s).Remove()
		}

		field := fieldT

		item := itemT

		resKey := field

		resValue := ""

		switch item.Type {

		//单个文字字段
		case Text:

			selectors := doc.Find(item.Selector)

			//排除选择器
			for _, s := range item.AfterNotSelector {

				selectors.Find(s).Remove()

			}

			v := strings.TrimSpace(selectors.Text())

			//res.Store(field, v)

			resKey = field

			resValue = v

			break

		//单个元素属性
		case Attr:

			v := ""

			if strings.TrimSpace(item.Selector) == "" {

				v, _ = doc.Attr(item.AttrKey)

			} else {

				v, _ = doc.Find(item.Selector).Attr(item.AttrKey)
			}

			//res.Store(field, strings.TrimSpace(v))

			resKey = field

			resValue = strings.TrimSpace(v)

			break

		//多个元素属性
		case Attrs:

			var v []string

			doc.Find(item.Selector).Each(func(i int, selection *goquery.Selection) {

				ss, ok := selection.Attr(item.AttrKey)

				if ok {

					v = append(v, ss)
				}

			})

			//res.Store(field, tools.Join(",", v))

			resKey = field

			resValue = tools.Join(",", v)

			break

		//只爬html（不包括图片）
		case OnlyHtml:

			selectors := doc.Find(item.Selector)

			//排除选择器
			for _, s := range item.AfterNotSelector {

				selectors.Find(s).Remove()

			}

			v, sErr := selectors.Html()

			if sErr != nil {

				res.Store(field, "")

				y.debugMsg("获取onlyHtml失败："+err.Error(), originUrl, item.Selector)

				globalErr = err

				break

			}

			//res.Store(field, v)

			resKey = field

			resValue = v

			break

		//爬取html，包括图片
		case HtmlWithImage:

			wait.Add(1)

			go func(_item Field, field string) {

				defer wait.Done()

				selectors := doc.Find(_item.Selector)

				//排除选择器
				for _, s := range item.AfterNotSelector {

					selectors.Find(s).Remove()

				}

				html_, sErr := selectors.Html()

				if sErr != nil {

					y.debugMsg(sErr.Error(), originUrl, _item.Selector)

					globalErr = sErr

					return
				}

				htmlImg, htmlImgErr := goquery.NewDocumentFromReader(strings.NewReader(html_))

				if htmlImgErr != nil {

					//f.s.notice.Error(err.Error() + ",源链接：" + originUrl)

					globalErr = err

					y.debugMsg("获取HtmlWithImage失败:"+htmlImgErr.Error(), originUrl, _item.Selector)

					return

				}

				var waitImg sync.WaitGroup

				var imgList = sync.Map{}

				htmlImg.Find("img").Each(func(i int, selection *goquery.Selection) {

					img, imgErr := y.getImageLink(selection, _item, originUrl)

					if imgErr != nil {

						//f.s.notice.Error(err.Error()+",源链接："+originUrl, ",富文本内容")

						globalErr = imgErr

						y.debugMsg(imgErr.Error(), originUrl, _item.Selector+" img")

						return
					}

					waitImg.Add(1)

					go func(waitImg *sync.WaitGroup, imgList *sync.Map, __item Field) {

						defer waitImg.Done()

						imgName, e := y.downImg(img, __item, res)

						if e != nil {

							//f.s.notice.Error(e.Error()+",源链接："+originUrl, ",富文本图片下载失败", "图片地址", img)

							y.debugMsg("富文本图片下载失败:"+"图片地址 "+img, originUrl, "")

						}

						globalErr = e

						imgList.Store(imgName, img)

					}(&waitImg, &imgList, _item)

				})

				waitImg.Wait()

				html_, _ = htmlImg.Html()

				imgList.Range(func(key, value interface{}) bool {

					html_ = strings.Replace(html_, value.(string), key.(string), -1)

					return true
				})

				//res.Store(field, html_)

				resKey = field

				resValue = html_

			}(item, field)

		//单个图片
		case Image:

			wait.Add(1)

			go func(_item Field, field string) {

				defer wait.Done()

				imgUrl, imgUrlErr := y.getImageLink(doc.Find(_item.Selector), _item, originUrl)

				if imgUrlErr != nil {

					//f.s.notice.Error(err.Error()+",源链接："+originUrl, ",选择器：", _item.Selector)

					y.debugMsg("获取单个图片选择器失败", originUrl, _item.Selector)

					globalErr = err

					return
				}

				imgName, e := y.downImg(imgUrl, _item, res)

				globalErr = e

				if e != nil {

					y.debugMsg("下载单个图片失败:"+imgUrl, originUrl, _item.Selector)

				} else {
					//res.Store(field, imgName)

					resKey = field

					resValue = imgName
				}

			}(item, field)

			break

		//单个文件
		case File:

			selectors := doc.Find(item.Selector)

			v, ok := selectors.Attr(item.AttrKey)

			if !ok {

				break
			}

			imgName, e := y.downImg(v, item, res)

			globalErr = e

			if e != nil {

				y.debugMsg("文件下班失败", v, item.Selector)

			} else {

				//res.Store(field, imgName)

				resKey = field

				resValue = imgName
			}

		//多个图片
		case MultipleImages:

			wait.Add(1)

			go func(_item Field, field string) {

				defer wait.Done()

				var waitImg sync.WaitGroup

				var imgList = sync.Map{}

				doc.Find(_item.Selector).Each(func(i int, selection *goquery.Selection) {

					imgUrl, imgUrlErr := y.getImageLink(selection, _item, originUrl)

					if imgUrlErr != nil {

						y.debugMsg(err.Error(), originUrl, _item.Selector)

						globalErr = err

						return
					}

					waitImg.Add(1)

					go func(waitImg *sync.WaitGroup, imgList *sync.Map, __item Field) {

						defer waitImg.Done()

						imgName, e := y.downImg(imgUrl, __item, res)

						if e != nil {

							//f.s.notice.Error(e.Error()+",源链接："+originUrl, ",选择器：", _item.Selector, "图片地址", imgUrl)

							y.debugMsg("图片下载失败:"+imgUrl, originUrl, _item.Selector)
						}

						globalErr = e

						imgList.Store(imgName, "")

					}(&waitImg, &imgList, _item)

				})

				waitImg.Wait()

				var strArray []string

				imgList.Range(func(key, value interface{}) bool {

					strArray = append(strArray, key.(string))

					return true
				})

				array := tools.Join(",", strArray)

				//res.Store(field, array)

				resKey = field

				resValue = array

			}(item, field)

		//固定数据
		case Fixed:

			//res.Store(field, item.Selector)

			resKey = field

			resValue = item.Selector

		//正则
		case Regular:

			reg := regexp.MustCompile(item.Selector).FindStringSubmatch(html)

			if len(reg) > 0 {

				index := 1

				if item.RegularIndex != 0 {

					index = item.RegularIndex
				}

				//res.Store(field, reg[index])

				resKey = field

				resValue = reg[index]

			} else {

				globalErr = errors.New("正则匹配未找到")

				//f.s.notice.Error("正则匹配未找到")

				y.debugMsg("正则匹配未找到", originUrl, item.Selector)
			}

		}

		wait.Wait()

		if item.ConversionFunc != nil {

			resValue = item.ConversionFunc(resValue)
		}

		res.Store(resKey, resValue)

	}

	arr := make(map[string]string)

	res.Range(func(key, value interface{}) bool {

		arr[key.(string)] = value.(string)

		return true

	})
	//
	//r := NewRows(arr)
	//
	//r.err = globalErr

	return arr, globalErr

}

// 获取图片链接
func (y *YySpider) getImageLink(imageDoc *goquery.Selection, item Field, originUrl string) (string, error) {

	//懒加载图片处理
	if item.LazyImageAttrName != "" {

		//Field里面的懒加载属性
		imgUrl, imgBool := imageDoc.Attr(item.LazyImageAttrName)

		if imgBool && imgUrl != "" {

			//填充图片src，防止图片无法显示
			imageDoc.RemoveAttr(item.LazyImageAttrName)

			imageDoc.SetAttr("src", imgUrl)

			return imgUrl, nil
		}

	}

	//懒加载图片处理
	if y.lazyImageAttrName != "" {

		//form里面的懒加载属性
		imgUrl, imgBool := imageDoc.Attr(y.lazyImageAttrName)

		if imgBool && imgUrl != "" {

			//填充图片src，防止图片无法显示
			imageDoc.RemoveAttr(y.lazyImageAttrName)

			imageDoc.SetAttr("src", imgUrl)

			return imgUrl, nil
		}

	}

	imgUrl, imgBool := imageDoc.Attr("src")

	if imgBool == false || imgUrl == "" {

		return "", errors.New("未找到图片链接，请检查是否存在懒加载")
	}

	return imgUrl, nil
}

// downImg 下载图片（包括生成文件夹）
func (y *YySpider) downImg(url string, item Field, res *sync.Map) (string, error) {

	url = strings.Replace(url, "\n", "", -1)

	//获取完整链接
	imgUrl := y.getHref(url)

	//生成随机名称
	uuidString := uuid.NewV4().String()

	uuidString = strings.Replace(uuidString, "-", "", -1)

	dir := ""

	//获取图片文件夹
	dir = y.getDir(item.ImageDir, res)

	//设置文件夹,图片保存路径+图片默认前缀路径+生成路径
	err := os.MkdirAll(y.completePath(y.savePath)+y.completePath(y.imageDir)+dir, 0755)

	if err != nil {

		//f.s.notice.Error(err.Error())

		y.debugMsg("设置文件夹失败:"+err.Error(), "", "")

		return "", err

	}

	ex, err := tools.GetExtensionName(imgUrl)

	if err != nil {

		ex = "png"

		//return "", err
	}

	//禁用拓展名检查
	if y.disableImageExtensionCheck {

		ex = "png"

	} else {

		allowImage := []string{"png", "jpg", "jpeg", "gif", "jfif"}

		//自定义允许下载的图片拓展名
		if len(y.allowImageExtension) > 0 {

			allowImage = y.allowImageExtension
		}

		if !tools.In_array(allowImage, strings.ToLower(ex)) {

			//f.s.notice.Error("图片拓展名异常:" + imgUrl)

			y.debugMsg("图片拓展名异常:"+imgUrl, "", "")

			////获取默认图片(这块有问题，先注释)
			//if y.DefaultImg != nil {
			//
			//	return f.DefaultImg(f, item), errors.New("图片拓展名异常,使用默认图片")
			//}

			return "", errors.New("图片拓展名异常")
		}

	}

	imgName := (y.If(dir == "", "", dir+"/")).(string) + uuidString + "." + ex

	prefix := ""

	if item.ImagePrefix != nil {

		prefix = item.ImagePrefix(imgName)

	}

	//自动添加斜杠
	prefix = strings.TrimRight(prefix, "/") + "/"

	var imgErr error

	if y.customDownloadFun != nil {

		imgErr = y.customDownloadFun(imgUrl, imgName)

	} else {

		//imgErr = f.s.client.R().Download(imgUrl, f.completePath(f.s.savePath)+f.completePath(f.s.imageDir)+imgName)

		_, imgErr = y.client.R().SetOutput(y.completePath(y.savePath) + y.completePath(y.imageDir) + imgName).Get(imgUrl)

	}

	if imgErr != nil {

		//msg := imgErr.Error()

		//f.s.notice.Error(msg)

		//y.debugMsg("图片下载失败:"+imgErr.Error(), "", "")

		////获取默认图片
		//if y.DefaultImg != nil {
		//
		//	return f.DefaultImg(f, item), errors.New("图片下载异常,使用默认图片：" + imgErr.Error())
		//}

		return "", errors.New("图片下载异常:" + imgUrl + " " + imgErr.Error())

	}

	//图片压缩
	if y.imageResizePercent != 0 {

		imgDeal := image.NewImage()

		imgRes, errRes := imgDeal.LoadImage(y.completePath(y.savePath) + y.completePath(y.imageDir) + imgName)

		if errRes != nil {

			return "", errors.New("图片压缩加载错误:" + errRes.Error())

		}

		ee := imgRes.ResizePercent(y.imageResizePercent).OverSave()

		if ee != nil {

			return "", errors.New("图片压缩错误:" + ee.Error())
		}

	}

	return (y.If(item.ImagePrefix == nil, "", prefix)).(string) + imgName, nil

}

// getHref 获取完整a链接
func (y *YySpider) getHref(href string) string {

	case1, _ := regexp.MatchString("^/[a-zA-Z0-9_]+.*", href)

	case2, _ := regexp.MatchString("^//[a-zA-Z0-9_]+.*", href)

	case3, _ := regexp.MatchString("^(http|https).*", href)

	switch true {

	case case1:

		href = y.host + href

		break

	case case2:

		//获取当前网址的协议
		res := regexp.MustCompile("^(https|http).*").FindStringSubmatch(y.host)

		href = res[1] + ":" + href

		break

	case case3:

		break

	default:

		href = y.host + "/" + href
	}

	return href

}

func (y *YySpider) getDir(path string, res *sync.Map) string {

	//替换时间格式
	r1, _ := regexp.Compile(`\[date:(.*?)]`)

	date := r1.FindAllStringSubmatch(path, -1)

	for _, v := range date {

		path = strings.Replace(path, v[0], tools.Date(v[1], time.Now().Unix()), -1)

	}

	//替换随机格式
	r2, _ := regexp.Compile(`\[random:([0-9]+-[0-9]+)]`)

	random := r2.FindAllStringSubmatch(path, -1)

	for _, v := range random {

		min, _ := strconv.Atoi(tools.Explode("-", v[1])[0])

		max, _ := strconv.Atoi(tools.Explode("-", v[1])[1])

		path = strings.Replace(path, v[0], strconv.FormatInt(tools.Mt_rand(int64(min), int64(max)), 10), -1)

	}

	//根据爬取文件给文件夹命名
	r3, _ := regexp.Compile(`\[singleField:(.*?)]`)

	singleField := r3.FindAllStringSubmatch(path, -1)

	for i, v := range singleField {

		field := ""

		//ok:=false

		if i == 0 {

			times := 0

			for {

				field_, ok := res.Load(v[1])

				if !ok {

					time.Sleep(200 * time.Millisecond)

					times++

					if times >= 5 {

						field = "timeout"

						break
					}

				} else {

					field = field_.(string)

					//处理为空的情况
					if field == "" {

						field = "unknown"
					}

					break

				}

			}

		}

		path = strings.Replace(path, v[0], field, -1)

	}

	return path

}

func (y *YySpider) completePath(path string) string {

	if path == "" {

		return path
	}

	m, _ := regexp.MatchString(`.*/$`, path)

	if m {

		return path
	}

	return path + "/"
}

// If 伪三元运算
func (y *YySpider) If(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}
