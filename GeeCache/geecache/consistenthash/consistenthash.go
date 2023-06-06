package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash // 哈希算法。（1.计算key哈希，2.计算节点哈希，3.计算虚拟节点哈希）
	replicas int
	keys     []int          // 哈希环。 ？？？
	hashMap  map[int]string //虚拟节点和真实节点的映射关系map，键是虚拟节点的哈希值，值是真实节点的名称。
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE //默认crc32
	}
	return m
}

//添加真实节点/虚拟节点。
func (m *Map) Add(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < m.replicas; i++ {
			subNodeHash := int(m.hash([]byte(strconv.Itoa(i) + node))) //创建虚拟节点
			m.keys = append(m.keys, subNodeHash)                       //添加到环上
			m.hashMap[subNodeHash] = node                              //虚拟节点与真实节点映射关系
		}
	}
	sort.Ints(m.keys) //给环上的哈希值排序
}

//需要查找的key选择节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	//计算key哈希
	hash := int(m.hash([]byte(key)))

	//找到匹配的虚拟节点下表。
	//二分查找。第一个参数代表切片长度。第二个用于判断给定索引的元素是否满足搜索条件。
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]]
}

// Remove use to remove a key and its virtual keys on the ring and map
func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(m.keys, hash)
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
		delete(m.hashMap, hash)
	}
}
