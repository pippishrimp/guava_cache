package GuavaCache

import (
	"container/list"
	"sync"
)

type LruCache struct {
	mu   sync.RWMutex
	cap  int
	ls   list.List
	data *sync.Map
}

func (l *LruCache) Init(cap int) {
	l.data = &sync.Map{}
	l.cap = cap
	l.mu = sync.RWMutex{}
	l.ls.Init()
}

func (l *LruCache) Get(key Key) (Value Value, ok bool) {
	return l.data.Load(key)
}

func (l *LruCache) Add(en *CacheEntry) *CacheEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	el, hit := l.data.Load(en.key)
	if hit {
		setEntry(el, en)
		//l.data.Store(en.key, el)
		l.ls.MoveToFront(el.(*list.Element))
		return nil
	}
	//如果当前容量还够，把节点放到头部
	if l.cap <= 0 || l.ls.Len() < l.cap {
		el = l.ls.PushFront(en)
		l.data.Store(en.key, el)
		return nil
	}
	// 如果当前容量不够的话，就会删除最后一个无素
	el = l.ls.Back()
	if el == nil {
		// Can happen if cap is zero
		return en
	}
	//元素移动到队头
	remEn := *getEntry(el)
	setEntry(el, en)
	l.ls.MoveToFront(el.(*list.Element))
	//删除原来的key,同进设置新key的值
	l.data.Delete(remEn.key)
	l.data.Store(en.key, el)
	return &remEn
}

func (l *LruCache) Hit(el *list.Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ls.MoveToFront(el)

}

func (l *LruCache) Remove(el *list.Element) *CacheEntry {
	en := getEntry(el)
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.data.Load(en.key); !ok {
		return nil
	}
	l.ls.Remove(el)
	l.data.Delete(en.key)
	return en
}

func (l *LruCache) Walk(f func(list *list.List)) {
	f(&l.ls)
}

func (l *LruCache) WalkCache(f func(key, value interface{}) bool) {
	l.data.Range(f)
}
