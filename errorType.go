package yySpider

type ErrorType int

const (
	HtmlRequestError ErrorType = 0x00000
	HtmlCodeError    ErrorType = 0x00001
)

type SpiderError struct {
	ErrorType ErrorType
	Message   string
	Link      string
}

func NewSpiderError(errorType ErrorType, message string, Link string) *SpiderError {

	return &SpiderError{
		errorType, message, Link,
	}
}
