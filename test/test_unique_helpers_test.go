package test

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var uniqueCounter uint64

func uniqueTestSuffix() string {
	n := atomic.AddUint64(&uniqueCounter, 1)
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatUint(n, 36)
}

func uniqueEmail(base string) string {
	b := strings.TrimSpace(strings.ToLower(base))
	at := strings.IndexByte(b, '@')
	if at <= 0 || at >= len(b)-1 {
		return "test-" + uniqueTestSuffix() + "@test.local"
	}
	local := b[:at]
	domain := b[at+1:]
	return local + "-" + uniqueTestSuffix() + "@" + domain
}

