package GuavaCache

import (
	"container/list"
	"fmt"
	"time"
)

type CacheEntry struct {
	key    Key
	value  Value
	access time.Time
	write  time.Time
}

func (e *CacheEntry) toString() string {
	if e == nil {
		panic("为null")
	}
	return fmt.Sprintf("key:%s, value:%s", e.key, e.value)
}

func getEntry(el interface{}) *CacheEntry {
	return el.(*list.Element).Value.(*CacheEntry)
}

func setEntry(el interface{}, en *CacheEntry) {
	el.(*list.Element).Value = en
}
