package lru

import "container/list"

type Cache struct {
	maxBytes int64 //允许使用的最大内存
	nbytes   int64 //当前已使用饿的内存
	ll       *list.List
	cache    map[string]*list.Element //键是字符串，值是双向链表中对应节点的指针。

	OnEvicted func(key string, value Value) //是某条记录被移除时的回调函数，可以为 nil。
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//Get look up a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {

	if ele, ok := c.cache[key]; ok {

		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

func (c *Cache) Put(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) //key存在，所以不考虑key的长度，只需要考虑新val和旧val长度差值
		kv.value = value
		c.ll.MoveToFront(ele)
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// RemoveOldest Remove
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() //返回尾节点或者空节点
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())

		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}

}

func (c *Cache) Len() int {
	return c.ll.Len()
}
