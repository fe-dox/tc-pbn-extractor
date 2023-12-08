package data

import (
	"crypto/md5"
	"fmt"
)

func JobHash(url string, options Options) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s %s %v", url, options.EventName, options.SplitOnDiscontinuation))))
}
