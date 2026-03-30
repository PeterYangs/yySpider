# YySpider Quick Start

## What is this

`yySpider` is a configuration-oriented web scraping library for Go.

You do not have to build the whole pipeline from scratch by yourself, such as:

- sending HTTP requests
- parsing HTML
- following pagination
- entering detail pages
- downloading images
- exporting data to Excel

Instead, you mainly define:

- the website host
- list page rules
- detail page rules
- field selectors

Then you call `Start()`, and the library runs the scraping flow for you.

If this is your first time looking at the project, here is the simplest way to understand it:

**You tell it where to go, what to extract, and where to go next. It takes care of wiring the whole flow together.**

---

## What this project can do

At the moment, this library supports the following features:

- fetch normal HTML pages
- scrape list pages
- follow detail pages
- chain multiple page levels
- scrape paginated data
- extract text, attributes, HTML, images, files, and regex matches
- download images automatically
- extract rich HTML content and download images inside it
- scrape browser-rendered pages
- click buttons and wait for elements in browser mode
- capture download links triggered in the browser
- process results with custom callbacks
- export to Excel by default
- use proxies
- set custom request headers
- configure image folders and image download behavior

If your goal is something like "list page -> detail page -> export to Excel", this library is already quite capable.

---

## Installation

```bash
go get github.com/PeterYangs/yySpider
```

It is recommended to use a relatively recent Go version.

---

## 5 core concepts to remember first

Before writing code, it helps a lot if you understand these 5 concepts first.

### 1. `YySpider`

This is the main controller of the entire scraping task.

You usually create it first:

```go
s := yySpider.NewYySpider(context.Background())
```

Things like host, proxy, browser mode, and result output are configured on it.

### 2. `ListPage`

This represents a list page.

Examples:

- product list
- article list
- category page
- chapter list

You use it to define:

- the list page URL rule
- the selector for each list item
- which fields to extract from each item
- where the next page or detail link is

### 3. `DetailPage`

This represents a detail page.

Examples:

- product detail page
- article detail page
- download page

You use it to define:

- which fields should be extracted from the detail page

### 4. `Field`

A field is a single piece of data you want to extract.

Examples:

- title
- price
- content
- image
- download URL

Each field usually needs at least:

- a `Type`
- a `Selector`

### 5. `Start()`

This is the method that actually starts the scraping job after everything is configured.

```go
err := s.Start()
```

---

## Minimal runnable example

If this is your first time using the project, start with a very small example like this:

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/PeterYangs/yySpider"
	uuid "github.com/satori/go.uuid"
)

func main() {
	s := yySpider.NewYySpider(context.Background())

	// 1. Set the website host
	s.Host("https://www.secretmine.net")

	// 2. Turn on debug output
	s.Debug()

	// 3. Define the list page
	list := s.NewListPage(
		"/tag/page_[PAGE]/",
		"body > div.main > div.downlist.boxbg.lazy.clearfix > ul > li",
		1,
		2,
	)

	// 4. Define the fields to extract from the list page
	list.SetFields(map[string]yySpider.Field{
		"title": {
			Type:     yySpider.Text,
			Selector: "div > b > a",
		},
		"size": {
			Type:     yySpider.Text,
			Selector: "div > i:nth-child(3)",
			ConversionFunc: func(item string) string {
				return strings.ReplaceAll(item, "大小：", "")
			},
		},
	})

	// 5. Tell the scraper where the detail page link is
	list.SetNextPageLinkSelector("p > a", "href")

	// 6. Define the detail page
	detail := s.NewDetailPage()
	detail.SetFields(map[string]yySpider.Field{
		"img": {
			Type:     yySpider.Image,
			Selector: "body > div.comment_box.clearfix > div.down_infor_top > div > img",
		},
	})

	// 7. Set the Excel output file name
	s.SetXlsxName("xlsx/" + uuid.NewV4().String())

	// 8. Start
	if err := s.Start(); err != nil {
		fmt.Println(err)
	}
}
```

What does this example do?

1. Open a website.
2. Visit the list page.
3. Find each `li` on the list page.
4. Extract the title and size from each item.
5. Follow the detail link inside each item.
6. Extract the image from the detail page.
7. Export the final result to Excel.

---

## How a full scraping flow works

You can think of the full process like this:

1. Create a `YySpider`
2. Configure basics such as host, headers, and proxy
3. Add the first page, usually a `ListPage`
4. Define fields for that page
5. If that page can lead to another page, define the next-page link
6. Add the second page, third page, and so on
7. Call `Start()`
8. The result goes either to Excel or to your custom callback

So the main idea of `yySpider` is not "request a random page and parse something".

The real idea is:

**Move through the page chain step by step.**

Examples:

- list page -> detail page
- category page -> product list page -> product detail page
- novel index -> chapter list -> chapter content

---

## Common APIs explained clearly

## 1. `YySpider` level configuration

### `NewYySpider(context.Context)`

Creates the scraper.

```go
s := yySpider.NewYySpider(context.Background())
```

Most of the time, `context.Background()` is enough.

If you want cancellation or timeout support, you can pass a context with timeout or cancel.

---

### `Host(host string)`

Sets the website host.

```go
s.Host("https://example.com")
```

Notes:

- this is the base host
- `NewListPage()` usually receives relative paths
- the library combines them into a full URL internally

---

### `Headers(map[string]string)`

Sets request headers.

```go
s.Headers(map[string]string{
	"user-agent": "Mozilla/5.0 ...",
	"referer":    "https://example.com",
})
```

Typical use cases:

- set `User-Agent`
- add `Referer`
- imitate a browser request
- handle simple anti-scraping checks

---

### `SetProxy(proxyUrl string)`

Sets a proxy.

```go
s.SetProxy("http://127.0.0.1:7897")
```

Useful when:

- you need a proxy for network access
- the website is region-restricted
- you want to inspect requests through a proxy tool

---

### `Debug()`

Turns on debug mode.

```go
s.Debug()
```

With debug mode enabled, you can see helpful runtime information such as:

- current request URL
- missing selectors
- image download failures
- browser waiting messages

If you are a beginner, it is a good idea to keep this on while learning.

---

### `UseBrowserMode()`

Enables browser mode, which uses `chromedp`.

```go
s.UseBrowserMode()
```

When should you use it?

- the page is rendered by JavaScript
- direct HTTP requests do not contain the real content
- content appears only after clicking something
- you need to wait for JS execution
- you want to capture the real download link from the browser

When should you avoid it?

- the page is static
- normal HTTP requests already return the needed data

If normal mode works, prefer normal mode first. Browser mode is heavier and slower.

---

### `SetDeviceType(d DeviceType)`

Sets the device type for browser emulation.

Supported values:

- `yySpider.DevicePC`
- `yySpider.DeviceAndroid`
- `yySpider.DeviceIPhone`

Example:

```go
s.UseBrowserMode()
s.SetDeviceType(yySpider.DeviceAndroid)
```

Useful when:

- the mobile page is simpler
- the mobile layout is easier to scrape
- download logic differs between desktop and mobile

---

### `SetXlsxName(name string)`

Sets the Excel output file name.

```go
s.SetXlsxName("xlsx/result")
```

This will become:

```text
xlsx/result.xlsx
```

If you do not set it, the library generates a random file name automatically.

---

### `ResultCallback(func(item map[string]string))`

Provides a custom result handler.

```go
s.ResultCallback(func(item map[string]string) {
	fmt.Println(item)
})
```

This is very important:

Once you set `ResultCallback`, results are handled by your callback and **will no longer be written to Excel by default**.

Useful when:

- you want to save results to a database
- you want to write JSON
- you just want to print and inspect the data

---

### `SetImageDir(path string)`

Sets the image directory name.

```go
s.SetImageDir("images")
```

Default:

```text
image
```

---

### `SetSavePath(path string)`

Sets the root path where images will be saved.

```go
s.SetSavePath("./data")
```

For example:

- `SetSavePath("./data")`
- `SetImageDir("images")`

Then images will roughly be saved under:

```text
./data/images/...
```

---

### `SetLazyImageAttrName(name string)`

Sets the global lazy-load image attribute name.

```go
s.SetLazyImageAttrName("data-original")
```

Some websites do not put the real image URL in `src`. They may use:

- `data-src`
- `data-original`
- `data-lazy`

That is when this setting helps.

---

### `DisableAutoCoding()`

Disables automatic character encoding conversion.

By default, the library tries to detect and convert encodings from the response.

If a website behaves badly with automatic conversion, you can try:

```go
s.DisableAutoCoding()
```

---

### `SetDisableImageExtensionCheck(bool)`

Disables image extension checking.

```go
s.SetDisableImageExtensionCheck(true)
```

This is useful when image URLs do not have standard file extensions or the extension cannot be trusted.

---

### `SetAllowImageExtension([]string)`

Limits which image extensions are allowed.

```go
s.SetAllowImageExtension([]string{"jpg", "jpeg", "png", "webp"})
```

Useful if you want strict control over image formats.

---

### `SetCustomDownloadFun(func(imgUrl, imgPath string) error)`

Lets you provide your own image download logic.

```go
s.SetCustomDownloadFun(func(imgUrl string, imgPath string) error {
	// implement your own download logic here
	return nil
})
```

Useful when:

- image download rules are unusual
- authentication is required
- you want to use your own downloader
- you want to upload directly to object storage

---

### `GetRedirectUrl(u string)`

Gets the final URL after redirects.

```go
finalURL, err := s.GetRedirectUrl(shortURL)
```

Useful for:

- short links
- redirected download URLs
- e-commerce out links

---

### `If(condition, trueVal, falseVal)`

A small ternary-style helper.

```go
v := s.If(price != "", price, "No price")
```

This is not a core scraping feature. It is just a small utility helper.

---

## 2. How to use `ListPage`

### `NewListPage(channel, listSelector, pageStart, pageLength)`

Creates a list page.

```go
list := s.NewListPage(
	"/news/page_[PAGE].html",
	".news-list li",
	1,
	5,
)
```

The 4 parameters mean:

1. the list page URL rule
2. the selector for a single list item
3. the starting page number
4. how many pages to scrape

`[PAGE]` is the page-number placeholder.

For example:

```text
/news/page_[PAGE].html
```

During runtime it becomes:

- `/news/page_1.html`
- `/news/page_2.html`
- `/news/page_3.html`

---

### `SetFields(map[string]Field)`

Defines which fields to extract from the list page.

```go
list.SetFields(map[string]yySpider.Field{
	"title": {
		Type:     yySpider.Text,
		Selector: "a.title",
	},
	"link": {
		Type:     yySpider.Attr,
		Selector: "a.title",
		AttrKey:  "href",
	},
})
```

One very important detail:

**List page field selectors are relative to each list item.**

So if your list item selector is:

```css
.news-list li
```

and your field selector is:

```css
a.title
```

then the library looks for `a.title` inside the current `li`, not from the top of the whole page.

---

### `SetNextPageLinkSelector(selector, attr)`

Tells the list page where the next-level page link is.

```go
list.SetNextPageLinkSelector("a.title", "href")
```

In most cases, this is the detail page link.

If you do not set it, the scraper will not continue to the next page level after parsing the list item.

---

### `Callback(func(item map[string]string))`

Callback for each list item.

```go
list.Callback(func(item map[string]string) {
	fmt.Println(item)
})
```

Useful for:

- debugging what is being extracted
- temporary printing
- adding some lightweight pacing logic

---

### `CallbackWithBreak(func(item map[string]string) bool)`

A callback that returns a boolean.

```go
list.CallbackWithBreak(func(item map[string]string) bool {
	if item["title"] == "" {
		return false
	}
	return true
})
```

Meaning of the return value:

- `true`: continue processing
- `false`: skip this item and do not continue deeper

This is very handy for filtering data.

---

### `RequestListPrefixCallback(func(listUrl string, currentIndex int))`

Callback before requesting a list page.

```go
list.RequestListPrefixCallback(func(listUrl string, currentIndex int) {
	fmt.Println("about to request:", listUrl)
})
```

Useful for:

- printing logs
- request-level metrics
- inspecting current pagination behavior

---

### `SetPreviousLinkCallback(func(listUrl string) string, startPage, maxPage int)`

The method name may look a bit confusing at first, but the idea is simple:

**Take an existing URL and expand it into a paginated URL rule.**

For example, suppose you first get:

```text
/category/phone
```

Then you want:

```text
/category/phone?page=1
/category/phone?page=2
/category/phone?page=3
```

You can do this:

```go
list2.SetPreviousLinkCallback(func(listUrl string) string {
	return listUrl + "?page=[PAGE]"
}, 1, 3)
```

---

### `SetHtmlCallback(func(htmlStr string, httpCode int, url string))`

Callback after getting the raw HTML.

```go
list.SetHtmlCallback(func(htmlStr string, httpCode int, url string) {
	fmt.Println(url, len(htmlStr))
})
```

Useful for:

- debugging raw HTML
- saving HTML samples
- checking why selectors are failing

---

### `SetWaitElement(selector, timeout)`

Only effective in browser mode.

```go
list.SetWaitElement(".news-list", 20*time.Second)
```

It means:

- open the page
- wait for the given element to appear
- continue only after that

If you do not configure it, the current version waits for `body` by default, with a default timeout of `15s`.

This is very useful for JavaScript-rendered pages.

---

### `SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error)`

Only effective in browser mode.

This allows you to operate on the page before the HTML is collected.

Examples:

- close a popup
- click an expand button
- click a download button
- wait for an additional section to load

Example:

```go
list.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {
	return chromedp.Click(".open-more", chromedp.ByQuery).Do(ctx)
})
```

---

### `SetDownload(downloadKey string)`

Mainly useful in browser mode.

It captures the download URL triggered in the browser and stores it into a result field.

```go
detail.SetDownload("download_url")
```

Then your final result will include:

```text
download_url
```

---

## 3. How to use `DetailPage`

`DetailPage` is much simpler than `ListPage`.

The most common methods are:

- `SetFields`
- `Callback`
- `CallbackWithBreak`
- `SetHtmlCallback`
- `SetWaitElement`
- `SetChromedpBeforeCallback`
- `SetDownload`

The usage is almost the same as `ListPage`, except it does not have pagination or next-page-entry logic.

A very common pattern:

```go
detail := s.NewDetailPage()

detail.SetFields(map[string]yySpider.Field{
	"title": {
		Type:     yySpider.Text,
		Selector: "h1",
	},
	"content": {
		Type:     yySpider.OnlyHtml,
		Selector: ".content",
	},
})
```

---

## Full field type reference

This section is important. Many beginners struggle here simply because field types are not fully clear yet.

## 1. `Text`

Extracts text.

```go
"title": {
	Type:     yySpider.Text,
	Selector: "h1",
}
```

Good for:

- title
- price
- author
- time

---

## 2. `Attr`

Extracts a single attribute from an element.

```go
"link": {
	Type:     yySpider.Attr,
	Selector: "a",
	AttrKey:  "href",
}
```

Good for:

- links
- image URLs
- download URLs

---

## 3. `Attrs`

Extracts multiple attributes and joins them into one string.

```go
"imgs": {
	Type:     yySpider.Attrs,
	Selector: ".gallery img",
	AttrKey:  "src",
}
```

Good for:

- multiple image URLs
- multiple download URLs

---

## 4. `Image`

Extracts a single image and downloads it locally.

```go
"cover": {
	Type:     yySpider.Image,
	Selector: ".cover img",
}
```

The stored result is the local image path, not the original image URL.

---

## 5. `MultipleImages`

Extracts multiple images and downloads them locally.

```go
"gallery": {
	Type:     yySpider.MultipleImages,
	Selector: ".content img",
}
```

The result will be a joined string of local image paths.

---

## 6. `OnlyHtml`

Extracts HTML without downloading images inside it.

```go
"content": {
	Type:     yySpider.OnlyHtml,
	Selector: ".article-content",
}
```

Good for:

- keeping the HTML structure only
- cases where local image download is not needed

---

## 7. `HtmlWithImage`

Extracts HTML, downloads the images inside it, and replaces image URLs in the HTML.

```go
"content": {
	Type:     yySpider.HtmlWithImage,
	Selector: ".article-content",
}
```

This is very useful for article scraping and rich-text migration.

---

## 8. `Fixed`

Returns a fixed value.

```go
"source": {
	Type:     yySpider.Fixed,
	Selector: "yySpider",
}
```

Important note: here `Selector` is not really a selector. It is the fixed value you want returned.

Good for:

- tagging source websites
- setting a default category
- writing constant fields

---

## 9. `Regular`

Extracts data using a regular expression.

```go
"price": {
	Type:         yySpider.Regular,
	Selector:     `"price":"([^"]+)"`,
	RegularIndex: 1,
}
```

Good for:

- JSON snippets inside the page
- values that are hard to get via selectors
- data hidden inside scripts

---

## 10. `File`

Downloads a file.

```go
"pdf": {
	Type:     yySpider.File,
	Selector: ".download-btn",
	AttrKey:  "href",
}
```

Good for:

- PDF files
- zip archives
- installers

---

## Advanced `Field` options

Besides `Type` and `Selector`, there are several advanced options that become useful later.

### `AttrKey`

Used by field types like `Attr`, `Attrs`, and `File`.

Examples:

- `href`
- `src`
- `data-src`

---

### `ConversionFunc`

A field conversion function.

```go
"size": {
	Type:     yySpider.Text,
	Selector: ".size",
	ConversionFunc: func(item string) string {
		return strings.TrimSpace(strings.ReplaceAll(item, "大小：", ""))
	},
}
```

Useful for:

- removing prefixes
- trimming spaces
- converting units
- formatting text

---

### `PrefixNotSelector`

Removes nodes before field parsing.

It means: remove some nodes from the HTML before the field is processed.

Useful for:

- removing ad blocks
- removing unrelated areas

---

### `AfterNotSelector`

Removes child nodes after locating the target selection.

It means: first locate the target node, then remove some child elements from it.

Useful for:

- keeping only the main content and dropping recommendations
- removing buttons, footnotes, or ads before extracting text

---

### `LazyImageAttrName`

A field-level lazy-load image attribute name.

If one specific field uses a special lazy image attribute, you can set it locally instead of globally.

---

### `ImageDir`

Sets a custom subdirectory for images of the current field.

Supported placeholders:

- `[date:Y-m-d]`
- `[random:1-100]`
- `[singleField:title]`

This is useful if you want to organize images by date, random bucket, or another field value.

---

### `ImagePrefix`

Sets a prefix for the returned image path.

Useful when you want the stored image path to include a custom prefix.

---

### `RegularIndex`

The regex capture group index.

Usually the default is group `1`.

---

## 5 common usage scenarios

## Scenario 1: list page -> detail page

This is the most common setup.

Examples:

- news list -> news detail
- product list -> product detail

Typical idea:

1. Create a `ListPage`
2. Extract summary fields from the list
3. Define the detail link
4. Create a `DetailPage`
5. Extract content, images, and so on from the detail page

---

## Scenario 2: multi-level pages

Example for a novel site:

- novel list
- chapter list
- chapter content

You can chain multiple pages like this:

```go
list1 := s.NewListPage(...)
list2 := s.NewListPage(...)
detail := s.NewDetailPage()
```

The library follows them in the order you add them.

---

## Scenario 3: you want results only, not Excel

Use `ResultCallback`.

```go
s.ResultCallback(func(item map[string]string) {
	fmt.Println(item)
})
```

---

## Scenario 4: the page is JS-rendered

Use browser mode:

```go
s.UseBrowserMode()
list.SetWaitElement(".list", 20*time.Second)
```

If you also need to click something:

```go
detail.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {
	return chromedp.Click(".load-more", chromedp.ByQuery).Do(ctx)
})
```

---

## Scenario 5: you need the real download URL after clicking

Use browser mode together with `SetDownload()`:

```go
detail.SetDownload("download_url")
```

Then trigger the download action in the browser, and the result will contain that field.

---

## How to think about browser mode

Many beginners get stuck here the first time, so here is the simple version:

- normal mode: send an HTTP request and parse the returned HTML
- browser mode: open a real browser and read the rendered HTML from it

Browser mode is useful when:

- the page is rendered by frontend JavaScript
- content appears only after clicking
- you need scrolling, waiting, or clicking
- you need to capture a download link

### Common browser-mode setup

```go
s.UseBrowserMode()
s.SetDeviceType(yySpider.DeviceAndroid)
list.SetWaitElement(".list", 20*time.Second)
detail.SetChromedpBeforeCallback(func(ctx context.Context, htmlUrl string) error {
	return chromedp.Click(".btn", chromedp.ByQuery).Do(ctx)
})
```

### Waiting rules in browser mode

In the current version:

- if you set `SetWaitElement()`, it waits for that element
- if you do not set it, it waits for `body` by default
- the default wait timeout is `15s`

So in practice:

- explicitly set `SetWaitElement()` for important pages
- do not rely only on the default waiting behavior

---

## How result output works

By default, results are written to Excel.

Each final record becomes a `map[string]string`, and then a row is written.

### Default output

- output file: `xxx.xlsx`
- first row: field names
- following rows: scraped data

### Custom output

If you set:

```go
s.ResultCallback(func(item map[string]string) {
	fmt.Println(item)
})
```

then the library will hand the result to your callback and will no longer write Excel automatically.

---

## A recommended beginner template

If you are not sure how to start, use this as a template:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PeterYangs/yySpider"
)

func main() {
	s := yySpider.NewYySpider(context.Background())

	s.Host("https://example.com")
	s.Debug()
	// s.Headers(map[string]string{"user-agent": "Mozilla/5.0 ..."})
	// s.SetProxy("http://127.0.0.1:7897")
	// s.UseBrowserMode()

	list := s.NewListPage(
		"/list_[PAGE].html",
		".list-item",
		1,
		2,
	)

	// For JS-rendered pages, adding a wait rule is recommended
	list.SetWaitElement(".list-item", 15*time.Second)

	list.SetFields(map[string]yySpider.Field{
		"title": {
			Type:     yySpider.Text,
			Selector: ".title",
		},
	})

	list.SetNextPageLinkSelector("a", "href")

	detail := s.NewDetailPage()
	detail.SetFields(map[string]yySpider.Field{
		"title": {
			Type:     yySpider.Text,
			Selector: "h1",
		},
		"content": {
			Type:     yySpider.OnlyHtml,
			Selector: ".content",
		},
	})

	s.ResultCallback(func(item map[string]string) {
		fmt.Println(item)
	})

	if err := s.Start(); err != nil {
		fmt.Println("scrape failed:", err)
	}
}
```

---

## Common beginner mistakes

## 1. Forgetting to call `Host()`

If the host is missing, relative URLs cannot be combined correctly.

---

## 2. Using a full-page selector where a list-item-relative selector is needed

For list fields, selectors are resolved inside each list item, not from the top of the page.

This is a very common mistake.

---

## 3. Using normal mode on a JS-rendered page

Typical symptoms:

- you get HTML, but the real content is missing
- selectors never match

In this case, you probably need `UseBrowserMode()`.

---

## 4. Forgetting to set a wait element in browser mode

If the page has not finished rendering yet, scraping starts too early and extraction fails.

For important pages, add:

```go
SetWaitElement(selector, timeout)
```

---

## 5. Setting `ResultCallback` and then still looking for an Excel file

Once `ResultCallback` is set, results are no longer written to Excel automatically.

---

## 6. Not handling lazy-loaded image attributes

Some websites do not use `src` for the real image URL. They may use:

- `data-src`
- `data-original`

In that case, remember to set:

```go
s.SetLazyImageAttrName("data-original")
```

Or set `LazyImageAttrName` on an individual field.

---

## 7. Relative links are not resolved as expected

The library can help complete relative links, but only if your `Host()` is correct.

---

## Debugging tips

If you cannot get the expected data, do not assume the library is broken right away.

Check things in this order:

1. turn on `s.Debug()`
2. verify the request URL
3. use `SetHtmlCallback()` to inspect or save raw HTML
4. verify selectors again in browser dev tools
5. check whether the page is frontend-rendered
6. if yes, add `SetWaitElement()` in browser mode
7. if the content appears only after interaction, add `SetChromedpBeforeCallback()`

This is usually the fastest way to diagnose problems.

---

## When to look at the `test/` directory

In this project, the `test/` directory is closer to an example collection than a traditional test suite.

If you want to see more real-world usage examples, start with:

- `test/goods.go`
- `test/slickdeals.go`
- `test/test-289.go`
- `test/test-cssmoban.go`

These examples cover:

- normal scraping
- multi-level list flows
- browser mode
- download-link capture

---

## Final advice for beginners

If this is your first time using the project, do not start with a very complex website.

A better learning path is:

1. first scrape a simple static list page
2. then add one detail page
3. then try `ResultCallback`
4. then try image downloading
5. finally try browser mode

That order is much less confusing.

If you want to build your own first scraper right now, copy the beginner template and focus on getting these 7 steps working first:

- `Host()`
- `NewListPage()`
- `SetFields()`
- `SetNextPageLinkSelector()`
- `NewDetailPage()`
- `detail.SetFields()`
- `Start()`

Once those 7 steps work, you can gradually add proxy support, browser mode, download-link capture, image directories, and the other advanced features.
