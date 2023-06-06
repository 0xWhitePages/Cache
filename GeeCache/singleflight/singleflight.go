package singleflight

import "sync"

/*
为了防止多个请求并发访问同一个 key, 需要过滤掉重复请求.

当一个请求到来时, 记录一下;

在一个请求处理完成之前, 如果有相同请求到达就排队等待, 直到正在处理的请求完成;

排队结束请求可以直接拿到前面请求完成返回的结果.

接下来就要考虑两个事情:

1.如何定义一次请求
2.如何存储这个请求对象

*/

// Call 定义一个请求
type Call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

//Group 存储请求对象
type Group struct {
	m  map[string]*Call
	mu sync.Mutex //涉及到map就要考虑线程安全

}

// Do 访问缓存
func (g *Group) Do(key string, fn func() (interface{}, error)) (val interface{}, err error) {

	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*Call)
	}

	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	//创建一个请求对象
	c := new(Call)
	c.wg.Add(1)
	g.m[key] = c
	// TODO ========= 请求缓存逻辑
	// 请求完成，要给其他请求放行
	c.val, c.err = fn()
	c.wg.Done()
	g.mu.Unlock()

	//删除
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err

}
