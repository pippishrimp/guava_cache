package GuavaCache

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type Key interface{}
type Value interface{}

type LoadFunc func(key Key) (Value, error)
type Option func(g *LoadingCache)
type CustomExpire func(value Value) bool

const DefaultSize int = 1 << 30

type LruContainer interface {
	Get(key Key) (Value Value, ok bool)
	Init(maximumSize int)
	Add(newEntry *CacheEntry) *CacheEntry
	Hit(element *list.Element)
	Remove(element *list.Element) *CacheEntry
	Walk(func(list *list.List))
	WalkCache(func(key, value interface{}) bool)
}
type LoadingCache struct {
	expireAfterAccess time.Duration
	expireAfterWrite  time.Duration
	refreshAfterWrite time.Duration
	maximumSize       int
	loader            LoadFunc
	addEvent          chan *CacheEntry
	removeEvent       chan *CacheEntry
	hitEvent          chan *list.Element
	singleFight       SingleFight
	lruContainer      LruContainer
	refreshMap        map[Key]bool
	refreshMapLock    sync.Mutex
	stats             StatsCounter
}

func newLoadingCache() *LoadingCache {
	return &LoadingCache{
		maximumSize: DefaultSize,
		refreshMap:  map[Key]bool{},
		stats:       &statsCounter{},
	}
}

func (g *LoadingCache) initCache() {
	if g.lruContainer == nil {
		g.lruContainer = new(LruCache)
	}
	g.lruContainer.Init(g.maximumSize)
	g.addEvent = make(chan *CacheEntry)
	g.hitEvent = make(chan *list.Element, 10)
	g.removeEvent = make(chan *CacheEntry)
	go g.processEvent()
	//最取小的值
	expireTime := getMinTimeDurationExcludeZero(g.expireAfterWrite, g.expireAfterAccess)
	go g.clearExpireKeyTask(expireTime)
}

func getMinTimeDurationExcludeZero(t1, t2 time.Duration) time.Duration {
	minTime := t1
	if t2 > 0 && (minTime == 0 || minTime > t2) {
		minTime = t2
	}
	return minTime
}

func BuildLoadingCache(options ...Option) *LoadingCache {
	g := newLoadingCache()
	for _, opt := range options {
		opt(g)
	}
	if g.loader == nil {
		panic("please set loader")
	}
	g.initCache()
	return g
}

/**
 * 1.如果没有命中，则直接load
 * 2.如果超时(access or write 超时)。直接loader
 * 3.如果access or write还没有超时。但是refreshAfterWrite的超时，会异步loader返回旧值。
 */
func (g *LoadingCache) Get(key Key) (Value, error) {
	return g.GetWithExpiredFunc(key, nil)
}
func (g *LoadingCache) GetWithExpiredFunc(key Key, f CustomExpire) (Value, error) {
	element, hit := g.lruContainer.Get(key)
	//如果缓存中没有数据。刚运行load方法
	if !hit {
		g.stats.RecordMisses(1)
		return g.loadInfo(key)
	}
	entry := getEntry(element)
	//判断数据是效性
	if g.isExpired(entry, time.Now()) || (f != nil && f(entry.value)) {
		return g.loadInfo(key)
	} else {
		g.checkRefreshAfterWrite(entry)
	}
	//变更entry的访问时间
	entry.access = time.Now()
	g.hitEvent <- element.(*list.Element)
	return entry.value, nil
}

func (g *LoadingCache) Put(key Key, v Value) {
	el, hit := g.lruContainer.Get(key)
	if hit {
		getEntry(el).value = v
	} else {
		en := &CacheEntry{
			key:    key,
			value:  v,
			write:  time.Now(),
			access: time.Now(),
		}
		g.lruContainer.Add(en)
	}
}

func (g *LoadingCache) Remove(key Key) {
	element, hit := g.lruContainer.Get(key)
	if hit {
		g.removeEl(element.(*list.Element))
	}
}

func (g *LoadingCache) checkRefreshAfterWrite(en *CacheEntry) {
	if g.refreshAfterWrite > 0 && en.write.Before(time.Now().Add(-g.refreshAfterWrite)) {
		loading := false
		g.refreshMapLock.Lock()
		_, ok := g.refreshMap[en.key]
		if ok {
			g.refreshMapLock.Unlock()
			return
		}
		loading = true
		g.refreshMap[en.key] = loading
		g.refreshMapLock.Unlock()
		go g.refresh(en)
	}

}

func (g *LoadingCache) refresh(en *CacheEntry) {
	if g.loader == nil {
		return
	}
	fight := func(key Key) (value Value, e error) {
		value, e = g.loader(key)
		if e == nil {
			en.value = value
			en.access = time.Now()
			en.write = time.Now()
			return value, nil
		} else {
			fmt.Errorf("refresh cache error, %s", e)
		}
		return nil, e
	}
	g.singleFight.fight(en.key, fight)
	g.refreshMapLock.Lock()
	delete(g.refreshMap, en.key)
	g.refreshMapLock.Unlock()
}

func (g *LoadingCache) loadInfo(key Key) (Value, error) {
	if g.loader == nil {
		return nil, EmptyError
	}
	fight := func(key Key) (value Value, e error) {
		startTime := time.Now()
		value, e = g.loader(key)
		if e == nil {
			entry := &CacheEntry{
				key:    key,
				value:  value,
				access: time.Now(),
				write:  time.Now(),
			}
			if removeEn := g.lruContainer.Add(entry); removeEn != nil {
				g.removeEvent <- removeEn
			}
			g.addEvent <- entry
			g.stats.RecordLoadSuccess(time.Now().Sub(startTime))
			return value, nil
		} else {
			g.stats.RecordLoadError(time.Now().Sub(startTime))
			fmt.Errorf("refresh cache error, %s", e)
		}
		return nil, e

	}
	return g.singleFight.fight(key, fight)
}

func (g *LoadingCache) removeEl(ele *list.Element) Value {
	entry := g.lruContainer.Remove(ele)
	if entry == nil {
		return nil
	}
	//g.removeEvent <- CacheEntry
	return entry.value
}

//定时清理
func (g *LoadingCache) clearExpiredKey() {
	g.lruContainer.Walk(func(list *list.List) {
		for el := list.Front(); el != nil; el = el.Next() {
			if g.isExpired(getEntry(el), time.Now()) {
				g.removeEl(el)
			}
		}
	})
}

func (g *LoadingCache) printAllCache() {
	g.lruContainer.WalkCache(func(key, value interface{}) bool {
		fmt.Println(key.(int), getEntry(value))
		return true
	})
}

func (g *LoadingCache) printLruCache() {
	g.lruContainer.Walk(func(list *list.List) {
		for el := list.Front(); el != nil; el = el.Next() {
			fmt.Println(getEntry(el).toString())
		}
	})
}

func (g *LoadingCache) isExpired(en *CacheEntry, now time.Time) bool {
	//相对访问的超时
	if g.expireAfterAccess > 0 && en.access.Before(now.Add(-g.expireAfterAccess)) {
		return true
	}
	if g.expireAfterWrite > 0 && en.write.Before(now.Add(-g.expireAfterWrite)) {
		return true
	}
	return false
}

// WithExpireAfterAccess returns an option to expire a cache CacheEntry after the
// given duration without being accessed.
func WithExpireAfterAccess(d time.Duration) Option {
	return func(g *LoadingCache) {
		g.expireAfterAccess = d
	}
}

// WithExpireAfterWrite returns an option to expire a cache CacheEntry after the
// given duration from creation.
func WithExpireAfterWrite(d time.Duration) Option {
	return func(g *LoadingCache) {
		g.expireAfterWrite = d
	}
}

// WithRefreshAfterWrite returns an option to refresh a cache CacheEntry after the
// given duration. This option is only applicable for LoadingCache.
func WithRefreshAfterWrite(d time.Duration) Option {
	return func(g *LoadingCache) {
		g.refreshAfterWrite = d
	}
}

func WithLoader(loader LoadFunc) Option {
	return func(g *LoadingCache) {
		g.loader = loader
	}
}

func WithMaximumSize(size int) Option {
	return func(g *LoadingCache) {
		g.maximumSize = size
	}
}

func (g *LoadingCache) Stats(t *Stats) {
	g.stats.Snapshot(t)
}

func WithLruContainer(lru LruContainer) Option {
	return func(g *LoadingCache) {
		g.lruContainer = lru
	}
}

func (g *LoadingCache) clearExpireKeyTask(expireTime time.Duration) {
	if expireTime == 0 {
		return
	}
	timer := time.NewTimer(expireTime)
	for {
		select {
		case <-timer.C:
			g.clearExpiredKey()
			timer.Reset(expireTime)
		}
	}
	timer.Stop()
}

func (g *LoadingCache) processEvent() {
	//defer close(c.closeCh)
	for {
		select {
		case el := <-g.hitEvent:
			g.lruContainer.Hit(el)
			g.stats.RecordHits(1)
		case <-g.removeEvent:
			//fmt.Println("remove", el.toString())
			g.stats.RecordEviction()
		case <-g.addEvent:

		}
	}
}
