package xerror

import "fmt"

type XError struct {
	Domain  string
	Code    int
	Message string
}

func (e *XError) Error() string {

	if e == nil {
		return "xferror info >> nil"
	}

	if e.Message == "" {
		return fmt.Sprintf("xferror info >> domain: %s code: %d error", e.Domain, e.Code)
	}

	return fmt.Sprintf("xferror info >> domain: %s code: %d message: %s", e.Domain, e.Code, e.Message)
}

// 使用
var defaultDomain string

func Init(domain string) {
	defaultDomain = domain
}

func New(code int) *XError {
	return &XError{
		Domain: defaultDomain,
		Code:   code,
	}
}

func Newf(code int, format string, args ...interface{}) *XError {
	return &XError{
		Domain:  defaultDomain,
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
