package GuavaCache

import "sync"

type call struct {
	wg  sync.WaitGroup
	val Value
	err error
}

type SingleFight struct {
	mu sync.Mutex
	m  map[Key]*call
}

func (g *SingleFight) fight(key Key, fn func(key Key) (Value, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[Key]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	c.val, c.err = fn(key)
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
