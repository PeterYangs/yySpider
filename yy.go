package yySpider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/PeterYangs/tools"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/go-resty/resty/v2"
	"github.com/phpisfirstofworld/image"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/xuri/excelize/v2"
	"log"
	"math/rand"
	"net/http"
	"net/url"
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
	pageList                   []Page
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
	browserMode                bool
	chromedpCtx                context.Context
	chromedpCancel             context.CancelFunc
	chromedpTimeout            time.Duration
	Device                     DeviceType
	downloadCh                 DownloadCh
}

type DownloadMessage struct {
	PageId      string
	DownloadUrl string
}

type DownloadCh chan DownloadMessage

type DeviceType string

const (
	DevicePC      DeviceType = "pc"
	DeviceAndroid DeviceType = "android"
	DeviceIPhone  DeviceType = "iphone"
)

const defaultBrowserWaitTimeout = 15 * time.Second

type DeviceProfile struct {
	Type             DeviceType
	Name             string
	UA               string
	Platform         string
	AcceptLanguage   string
	Width            int64
	Height           int64
	DeviceScaleRatio float64
	Mobile           bool
	Touch            bool
	MaxTouchPoints   int64
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewYySpider(cxt context.Context) *YySpider {

	//client := resty.New()

	y := &YySpider{
		header:     make(map[string]string),
		cxt:        cxt,
		imageDir:   "image",
		lock:       sync.Mutex{},
		Device:     DevicePC,
		downloadCh: make(DownloadCh, 20),
	}

	y.client = y.httpInit()

	return y
}

// SetDeviceType 设置设备（支持pc、安卓和ios）
func (y *YySpider) SetDeviceType(d DeviceType) *YySpider {

	y.Device = d

	return y
}

func (y *YySpider) UseBrowserMode() *YySpider {

	y.browserMode = true

	return y
}

func (y *YySpider) initChrome(device DeviceType) (context.Context, context.CancelFunc, error) {

	profile := y.getRandomDeviceProfile(device)

	var opts []chromedp.ExecAllocatorOption

	// 1. 基础配置：显式禁用导致报错的 Flag，并设置初始化参数
	opts = append(opts, chromedp.DefaultExecAllocatorOptions[:]...)

	// 2. 核心伪装：针对 Cloudflare 2026 检测逻辑的配置
	opts = append(opts,
		// 必须：禁用控制特征，防止 window.navigator.webdriver = true
		chromedp.Flag("disable-blink-features", "AutomationControlled"),

		// 必须：防止 Chrome 弹出“正受到自动化软件控制”的提示
		chromedp.Flag("excludeSwitches", "enable-automation"),
		chromedp.Flag("enable-automation", false),

		// 增强：禁用一些暴露 CDP 特征的选项
		chromedp.Flag("use-mock-keychain", true),        // 避免尝试访问系统钥匙串
		chromedp.Flag("no-default-browser-check", true), // 禁用默认浏览器检查

		// 硬件伪装：设置真实的窗口和显卡加速模式
		chromedp.WindowSize(int(profile.Width), int(profile.Height)),
		chromedp.UserAgent(profile.UA),
		chromedp.Flag("disable-gpu", false), // 开启 GPU，让 Canvas 渲染指纹看起来更像真人
	)

	// 3. 区分 Debug 模式
	if y.debug {
		opts = append(opts,
			chromedp.Flag("headless", false),
			// 使用一个独立的临时目录，而不是真实用户数据，确保每次运行环境干净
			//chromedp.Flag("user-data-dir", "/tmp/chromedp_test_profile"),
		)
	} else {
		opts = append(opts,
			chromedp.Flag("headless", "new"), // 生产环境建议开启
		)
	}

	// 4. 动态参数：User-Agent 和 Proxy
	//ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
	//opts = append(opts, chromedp.UserAgent(ua))

	if y.proxyUrl != "" {
		opts = append(opts, chromedp.ProxyServer(y.proxyUrl))
	}

	// 5. 启动 Allocator
	timeout := y.chromedpTimeout
	if timeout == 0 {
		timeout = 3 * time.Hour
	}
	parentCtx := y.cxt
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	baseCtx, baseCancel := context.WithTimeout(parentCtx, timeout)
	allocCtx, allocCancel := chromedp.NewExecAllocator(baseCtx, opts...)

	// 6. 创建 Context 并注入 JS (这是最后一道防线)
	ctx, ctxCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(func(s string, i ...interface{}) {}),
	)

	// 先执行伪装和下载配置，再打开页面
	if err := chromedp.Run(ctx, y.buildEmulationActions(profile, "", true)...); err != nil {
		ctxCancel()
		allocCancel()
		baseCancel()
		return nil, nil, err
	}

	//监听下载
	//y.ListenBrowserDownloadURL(ctx)

	// 关键：在页面加载前强制覆盖指纹
	y.chromedpCtx = ctx
	y.chromedpCancel = func() {
		ctxCancel()
		allocCancel()
		baseCancel()
	}

	return y.chromedpCtx, y.chromedpCancel, nil
}

//// ListenBrowserDownloadURL 下载监听
//func (y *YySpider) ListenBrowserDownloadURL(ctx context.Context) {
//	seen := make(map[string]bool)
//
//	chromedp.ListenTarget(ctx, func(ev interface{}) {
//		switch e := ev.(type) {
//		case *browser.EventDownloadWillBegin:
//			// 按 guid 去重，避免同一个下载打印两次
//			if seen[e.GUID] {
//				return
//			}
//			seen[e.GUID] = true
//
//			fmt.Println("最终下载地址:", e.URL)
//			if e.SuggestedFilename != "" {
//				fmt.Println("建议文件名:", e.SuggestedFilename)
//			}
//		}
//	})
//}

// ListenBrowserDownloadURL 下载监听（推荐版）
func (y *YySpider) ListenBrowserDownloadURL(ctx context.Context, pageId string) {
	seen := make(map[string]time.Time)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {

		case *browser.EventDownloadWillBegin:
			now := time.Now()

			// 清理过期 GUID（防止 map 无限增长）
			for k, t := range seen {
				if now.Sub(t) > 5*time.Minute {
					delete(seen, k)
				}
			}

			// 去重
			if _, ok := seen[e.GUID]; ok {
				return
			}
			seen[e.GUID] = now

			//fmt.Println("最终下载地址:", e.URL, pageId)

			y.downloadCh <- DownloadMessage{PageId: pageId, DownloadUrl: e.URL}

			if e.SuggestedFilename != "" {
				//fmt.Println("建议文件名:", e.SuggestedFilename)
			}
		}
	})
}

func (y *YySpider) buildEmulationActions(profile DeviceProfile, downloadDir string, denyDownload bool) []chromedp.Action {
	var downloadAction chromedp.Action

	if denyDownload {
		downloadAction = chromedp.ActionFunc(func(ctx context.Context) error {
			return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorDeny).
				WithEventsEnabled(true).
				Do(ctx)
		})
	} else {
		downloadAction = chromedp.ActionFunc(func(ctx context.Context) error {
			return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
				WithDownloadPath(downloadDir).
				WithEventsEnabled(true).
				Do(ctx)
		})
	}

	return []chromedp.Action{
		downloadAction,
		network.Enable(),

		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetUserAgentOverride(profile.UA).
				WithAcceptLanguage(profile.AcceptLanguage).
				WithPlatform(profile.Platform).
				Do(ctx)
		}),
		y.buildViewportAction(profile),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetTouchEmulationEnabled(profile.Touch).
				WithMaxTouchPoints(profile.MaxTouchPoints).
				Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(y.buildStealthJS(profile)).Do(ctx)
			return err
		}),
	}
}

func (y *YySpider) buildStealthJS(profile DeviceProfile) string {
	orientation := "landscape-primary"
	if profile.Mobile {
		orientation = "portrait-primary"
	}

	mobileJS := "false"
	if profile.Mobile {
		mobileJS = "true"
	}

	touchJS := "false"
	if profile.Touch {
		touchJS = "true"
	}

	return fmt.Sprintf(`
(() => {
	const overrideGetter = (obj, prop, value) => {
		try {
			Object.defineProperty(obj, prop, {
				get: () => value,
				configurable: true
			});
		} catch (e) {}
	};

	const overrideValue = (obj, prop, value) => {
		try {
			Object.defineProperty(obj, prop, {
				value: value,
				configurable: true,
				writable: false
			});
		} catch (e) {}
	};

	overrideGetter(Navigator.prototype, 'webdriver', undefined);
	overrideGetter(Navigator.prototype, 'platform', %q);
	overrideGetter(Navigator.prototype, 'language', 'zh-CN');
	overrideGetter(Navigator.prototype, 'languages', ['zh-CN', 'zh', 'en']);
	overrideGetter(Navigator.prototype, 'maxTouchPoints', %d);
	overrideGetter(Navigator.prototype, 'hardwareConcurrency', 8);
	overrideGetter(Navigator.prototype, 'deviceMemory', 8);
	overrideGetter(Navigator.prototype, 'vendor', 'Google Inc.');
	overrideGetter(Navigator.prototype, 'doNotTrack', null);

	if (!window.chrome) {
		overrideValue(window, 'chrome', {});
	}
	if (!window.chrome.runtime) {
		overrideValue(window.chrome, 'runtime', {});
	}

	if (window.navigator.permissions && window.navigator.permissions.query) {
		const originalQuery = window.navigator.permissions.query.bind(window.navigator.permissions);
		window.navigator.permissions.query = (parameters) => {
			if (parameters && parameters.name === 'notifications') {
				return Promise.resolve({ state: Notification.permission });
			}
			return originalQuery(parameters);
		};
	}

	const fakePlugins = [
		{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
		{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
		{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
	];
	overrideGetter(Navigator.prototype, 'plugins', fakePlugins);
	overrideGetter(Navigator.prototype, 'mimeTypes', [
		{ type: 'application/pdf', suffixes: 'pdf', description: '', enabledPlugin: fakePlugins[0] }
	]);

	if (!navigator.userAgentData) {
		overrideGetter(Navigator.prototype, 'userAgentData', {
			mobile: %s,
			brands: [
				{ brand: 'Chromium', version: '137' },
				{ brand: 'Google Chrome', version: '137' },
				{ brand: 'Not/A)Brand', version: '24' }
			],
			getHighEntropyValues: async function() {
				return {
					architecture: 'arm',
					bitness: '64',
					mobile: %s,
					model: '',
					platform: %q,
					platformVersion: '0.0.0',
					uaFullVersion: '137.0.0.0'
				};
			}
		});
	}

	try {
		if (screen.orientation) {
			overrideGetter(screen.orientation, 'type', %q);
			overrideGetter(screen.orientation, 'angle', 0);
		}
	} catch (e) {}

	if (%s) {
		if (!('ontouchstart' in window)) {
			overrideValue(window, 'ontouchstart', null);
		}
	}

	try {
		overrideGetter(screen, 'availWidth', %d);
		overrideGetter(screen, 'availHeight', %d);
		overrideGetter(screen, 'width', %d);
		overrideGetter(screen, 'height', %d);
		overrideGetter(screen, 'colorDepth', 24);
		overrideGetter(screen, 'pixelDepth', 24);
	} catch (e) {}

	try {
		overrideGetter(window, 'devicePixelRatio', %f);
	} catch (e) {}
})();
`,
		profile.Platform,
		profile.MaxTouchPoints,
		mobileJS,
		mobileJS,
		profile.Platform,
		orientation,
		touchJS,
		profile.Width,
		profile.Height,
		profile.Width,
		profile.Height,
		profile.DeviceScaleRatio,
	)
}

func (y *YySpider) buildViewportAction(profile DeviceProfile) chromedp.Action {
	opts := []chromedp.EmulateViewportOption{
		chromedp.EmulateScale(profile.DeviceScaleRatio),
	}

	if profile.Mobile {
		opts = append(opts,
			chromedp.EmulateMobile,
			chromedp.EmulateTouch,
			chromedp.EmulatePortrait,
		)
	} else {
		opts = append(opts,
			chromedp.EmulateLandscape,
		)
	}

	return chromedp.EmulateViewport(profile.Width, profile.Height, opts...)
}

func (y *YySpider) getRandomDeviceProfile(deviceType DeviceType) DeviceProfile {
	pcProfiles := []DeviceProfile{
		{
			Type:             DevicePC,
			Name:             "Windows Chrome Desktop",
			UA:               "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
			Platform:         "Win32",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            1920,
			Height:           1080,
			DeviceScaleRatio: 1,
			Mobile:           false,
			Touch:            false,
			MaxTouchPoints:   0,
		},
		{
			Type:             DevicePC,
			Name:             "Mac Chrome Desktop",
			UA:               "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
			Platform:         "MacIntel",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            1440,
			Height:           900,
			DeviceScaleRatio: 2,
			Mobile:           false,
			Touch:            false,
			MaxTouchPoints:   0,
		},
		{
			Type:             DevicePC,
			Name:             "Windows Chrome Desktop Small",
			UA:               "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
			Platform:         "Win32",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            1536,
			Height:           864,
			DeviceScaleRatio: 1,
			Mobile:           false,
			Touch:            false,
			MaxTouchPoints:   0,
		},
	}

	androidProfiles := []DeviceProfile{
		{
			Type:             DeviceAndroid,
			Name:             "Android Pixel 7",
			UA:               "Mozilla/5.0 (Linux; Android 13; Pixel 7 Build/TQ3A.230805.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Mobile Safari/537.36",
			Platform:         "Linux armv8l",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            412,
			Height:           915,
			DeviceScaleRatio: 2.625,
			Mobile:           true,
			Touch:            true,
			MaxTouchPoints:   5,
		},
		{
			Type:             DeviceAndroid,
			Name:             "Android Xiaomi",
			UA:               "Mozilla/5.0 (Linux; Android 12; M2012K11AC Build/SKQ1.211006.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Mobile Safari/537.36",
			Platform:         "Linux armv8l",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            393,
			Height:           873,
			DeviceScaleRatio: 2.75,
			Mobile:           true,
			Touch:            true,
			MaxTouchPoints:   5,
		},
		{
			Type:             DeviceAndroid,
			Name:             "Android Samsung",
			UA:               "Mozilla/5.0 (Linux; Android 14; SM-S9280 Build/UP1A.231005.007) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Mobile Safari/537.36",
			Platform:         "Linux armv8l",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            360,
			Height:           800,
			DeviceScaleRatio: 3,
			Mobile:           true,
			Touch:            true,
			MaxTouchPoints:   5,
		},
	}

	iphoneProfiles := []DeviceProfile{
		{
			Type:             DeviceIPhone,
			Name:             "iPhone Safari iOS 17",
			UA:               "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
			Platform:         "iPhone",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            393,
			Height:           852,
			DeviceScaleRatio: 3,
			Mobile:           true,
			Touch:            true,
			MaxTouchPoints:   5,
		},
		{
			Type:             DeviceIPhone,
			Name:             "iPhone Chrome iOS 17",
			UA:               "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/137.0.0.0 Mobile/15E148 Safari/604.1",
			Platform:         "iPhone",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            430,
			Height:           932,
			DeviceScaleRatio: 3,
			Mobile:           true,
			Touch:            true,
			MaxTouchPoints:   5,
		},
		{
			Type:             DeviceIPhone,
			Name:             "iPhone Safari iOS 16",
			UA:               "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7_10 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
			Platform:         "iPhone",
			AcceptLanguage:   "zh-CN,zh;q=0.9,en;q=0.8",
			Width:            390,
			Height:           844,
			DeviceScaleRatio: 3,
			Mobile:           true,
			Touch:            true,
			MaxTouchPoints:   5,
		},
	}

	switch deviceType {
	case DevicePC:
		return y.randomFrom(pcProfiles)
	case DeviceAndroid:
		return y.randomFrom(androidProfiles)
	case DeviceIPhone:
		return y.randomFrom(iphoneProfiles)
	default:
		return y.randomFrom(pcProfiles)
	}
}

func (y *YySpider) randomFrom(list []DeviceProfile) DeviceProfile {
	return list[rand.Intn(len(list))]
}

func (y *YySpider) chromeProfileDir() string {
	host := strings.TrimPrefix(strings.TrimPrefix(y.host, "https://"), "http://")
	host = strings.TrimRight(host, "/")

	key := host + "|" + y.proxyUrl
	sum := md5.Sum([]byte(key))
	name := hex.EncodeToString(sum[:8])

	// 放到你项目目录或可写目录下
	dir := filepath.Join(".", "chrome_profiles", name)
	_ = os.MkdirAll(dir, 0755)
	return dir
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

	defer func() {

		close(y.downloadCh)

	}()

	y.excelInit()

	res := make(map[string]string)

	if len(y.pageList) <= 0 {

		return errors.New("page数量为0")
	}

	//浏览器模式（浏览器的模式打开浏览器）
	if y.browserMode {

		_, chromeCancel, err := y.initChrome(y.Device)
		if err != nil {
			return err
		}
		if chromeCancel != nil {
			defer chromeCancel()
		}
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

	////获取下载地址
	//if page.GetDownloadKey() != "" && y.browserMode == true {
	//
	//}

	switch page.(type) {

	case *ListPage:

		listPage := page.(*ListPage)

		if link != "" && listPage.previousLinkCallback == nil {

			//多重列表
			err := y.getList(link, listPage, res, currentIndex)

			if err != nil {

				y.debugMsg(err.Error(), link, "")

			}

		} else if link != "" && listPage.previousLinkCallback != nil {

		FOR2:

			for i := listPage.previousStartPage; i < listPage.previousStartPage+listPage.previousMaxPage; i++ {

				select {

				case <-y.cxt.Done():

					break FOR2

				default:

				}

				u, uErr := url.Parse(link)

				if uErr != nil {
					y.debugMsg(uErr.Error(), link, "")
					return uErr
				}

				channelLink := listPage.previousLinkCallback(y.getFullPath(u))

				listLink := y.host + strings.Replace(channelLink, "[PAGE]", strconv.Itoa(i), -1)

				err := y.getList(listLink, listPage, res, currentIndex)

				if err != nil {

					y.debugMsg(err.Error(), listLink, "")

				}

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

	pageId := uuid.NewV4().String()

	html, err := y.requestHtml(listUrl, listPage, pageId)

	if err != nil {

		return err

	}

	if y.debug {

		fmt.Println(listUrl)
	}

	if listPage.GetHtmlCallback() != nil {

		callback := listPage.GetHtmlCallback()

		callback(html, 200, listUrl)
	}

	//获取下载地址
	if listPage.GetDownloadKey() != "" && y.browserMode == true {

		select {
		case v := <-y.downloadCh:

			if v.PageId == pageId {

				res[listPage.GetDownloadKey()] = v.DownloadUrl
			}

		case <-time.After(15 * time.Second):

		}

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

	pageId := uuid.NewV4().String()

	html, err := y.requestHtml(detailUrl, detailPage, pageId)

	if err != nil {

		return err

	}

	if detailPage.GetHtmlCallback() != nil {

		callback := detailPage.GetHtmlCallback()

		callback(html, 200, detailUrl)
	}

	//获取下载地址
	if detailPage.GetDownloadKey() != "" && y.browserMode == true {

		select {
		case v := <-y.downloadCh:

			if v.PageId == pageId {

				res[detailPage.GetDownloadKey()] = v.DownloadUrl
			}

		case <-time.After(15 * time.Second):

		}

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

func (y *YySpider) requestHtml(htmlUrl string, page Page, pageId string) (string, error) {

	if y.browserMode {

		var html string

		if page.GetDownloadKey() != "" {

			listenCtx, listenCancel := context.WithCancel(y.chromedpCtx)

			defer listenCancel()

			//监听下载
			y.ListenBrowserDownloadURL(listenCtx, pageId)

		}

		err := chromedp.Run(
			y.chromedpCtx,
			// 注入底层抹除脚本
			chromedp.Navigate(htmlUrl),
			chromedp.ActionFunc(func(ctx context.Context) error {
				return y.waitForBrowserPage(ctx, page, htmlUrl)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {

				if page.GetChromedpBeforeCallback() != nil {

					callback := page.GetChromedpBeforeCallback()

					return callback(ctx, htmlUrl)
				}

				return nil
			}),
			chromedp.OuterHTML("html", &html, chromedp.ByQuery),
		)

		if err != nil {
			return "", err
		}

		return html, nil
	}

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

func (y *YySpider) waitForBrowserPage(ctx context.Context, page Page, htmlUrl string) error {

	selector, timeout := page.GetWaitElement()
	selector = strings.TrimSpace(selector)
	if selector == "" {
		selector = "body"
	}
	if timeout <= 0 {
		timeout = defaultBrowserWaitTimeout
	}

	y.debugMsg(fmt.Sprintf("等待页面元素就绪，超时：%s", timeout), htmlUrl, selector)

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return chromedp.WaitVisible(selector, chromedp.ByQuery).Do(waitCtx)
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

	var globalErr error
	var globalErrLock sync.Mutex

	setGlobalErr := func(err error) {
		if err == nil {
			return
		}
		globalErrLock.Lock()
		if globalErr == nil {
			globalErr = err
		}
		globalErrLock.Unlock()
	}

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

				y.debugMsg("获取onlyHtml失败："+sErr.Error(), originUrl, item.Selector)

				setGlobalErr(sErr)

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

					setGlobalErr(sErr)

					return
				}

				htmlImg, htmlImgErr := goquery.NewDocumentFromReader(strings.NewReader(html_))

				if htmlImgErr != nil {

					//f.s.notice.Error(err.Error() + ",源链接：" + originUrl)

					setGlobalErr(htmlImgErr)

					y.debugMsg("获取HtmlWithImage失败:"+htmlImgErr.Error(), originUrl, _item.Selector)

					return

				}

				var waitImg sync.WaitGroup

				var imgList = sync.Map{}

				htmlImg.Find("img").Each(func(i int, selection *goquery.Selection) {

					img, imgErr := y.getImageLink(selection, _item, originUrl)

					if imgErr != nil {

						//f.s.notice.Error(err.Error()+",源链接："+originUrl, ",富文本内容")

						setGlobalErr(imgErr)

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

						setGlobalErr(e)

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

					setGlobalErr(imgUrlErr)

					return
				}

				imgName, e := y.downImg(imgUrl, _item, res)

				setGlobalErr(e)

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

			setGlobalErr(e)

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

						y.debugMsg(imgUrlErr.Error(), originUrl, _item.Selector)

						setGlobalErr(imgUrlErr)

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

						setGlobalErr(e)

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

				setGlobalErr(errors.New("正则匹配未找到"))

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

func (y *YySpider) getFullPath(u *url.URL) string {
	if u.RawQuery == "" {
		return u.Path
	}
	return u.Path + "?" + u.RawQuery
}

func (y *YySpider) GetRedirectUrl(u string) (string, error) {

	rsp, err := y.client.R().SetDoNotParseResponse(true).Get(u)

	if err != nil {

		return "", err
	}

	defer rsp.RawResponse.Body.Close()

	return rsp.RawResponse.Request.URL.String(), nil
}
