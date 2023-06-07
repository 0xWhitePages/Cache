# MyCache是参考GroupCache实现的自定义分布式缓存
> 参考链接：  
> 
>         https://geektutu.com/post/geecache.html  
>         
>         https://github.com/golang/groupcache  
> 	  
>         https://blog.csdn.net/cnm10050/article/details/128307898

## 设计一个缓存需要考虑哪些问题？  

1.缓存内存不够时该如何处理?  

2.并发写入冲突如何解决？  

3.单机性能不够该怎么办？  


> 从这三个问题出发，去尝试实现一个缓存，并在学习过程中提出新的问题。

----------------------------------------------------------------

## Q&A



> 缓存空间不足时如何应对？

答：

应对缓存空间不足的情况，那就需要删除数据。删除数据就需要考虑什么时候删除，删除哪些数据。思考什么算法更佳，LRU（最近最少使用）是一个二维算法，考虑到了时间以及频率。所以选择LRU算法来设计底层cache数据结构。



> 如何设计LRU的数据结构？


答：
针对LRU1.增 2.删 3.改 4.查特性，查能满足O（1）时间复杂度的就是map，修改删除也要O(1)那就可以排除列表、栈，因为都不能中间查询。所以双链表是最佳选择。

​		并且因为map是线程不安全的，所以我们必须加锁。



> 如果查询的数据缓存中不存在的话，该如何处理？


答：
缓存如果不存在，我们就需要到数据源去取数据返回，如何取？我们给每个类别缓存（Group）封装一个回调函数（Getter），并且将这个数据源复制一份到缓存中保存。



----------------------------------------------

> 分布式缓存有哪些好处，为什么我们需要分布式缓存？


答：

1. 性能/资源：首先是单机资源有限（计算、性能），不足以支撑更大的业务量和需求。我们需要利用多台计算机资源 优化性能。
2. 高可用/容错：单节点一旦崩掉，系统就会中断，十分脆弱。多节点能提供高可用和容错性。
3. 数据共享/负载均衡：分布式系统使得多个节点之间可以共享和访问数据，从而实现协作和协调。多个节点可以同时读写数据，进行并发操作，通过分布式数据存储和共享机制，实现数据的一致性和可靠性。
4. （个人）能减轻数据库的压力，当前节点没有数据会先去其他节点查询，如果没有再去数据库取。

​			 .....

> 分布式缓存需要考虑哪些因素呢？


答：

首先，分布式缓存那就必然需要考虑节点之间的通信问题，可以使用HTTP或者RPC通信。

其次，上述第4点当本节点没有所请求的数据时，找其他节点的算法该如何设计。如果没有设计好算法，可能会导致《缓存雪崩》。

我们使用了一致性哈希来解决节点选择的问题（其中对于数据倾斜，我们使用了虚拟节点来解决）。


```go
//一致性哈希中的哈希环是每一个节点都需要维护一个吗？
项目实现确实是每个节点都维护了一个。
是不是可以把这个一致性哈希抽象出来，作为单独的`Proxy`层。
```



> 一个节点在分布式系统中扮演什么角色，职责是什么？


答：

一个节点既是服务端，也是客户端。

来看一个节点的角色变换：

先是作为服务端收到key，找缓存，缓存中没有的话那就PickPeer一个节点，然后变成客户端getFromPeer请求远程节点。





> 既然有一致性哈希来通过key选择节点了，为什么还需要singleflight防止缓存击穿。


答：一致性哈希算法解决了key选择节点的问题。缓存击穿是因为选择节点后并发请求热点key，然后在缓存未命中的情况下对数据库造成压力。




## Usage

```go
package main

import (
	"GeeCache/geecache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup(name string) *geecache.Group {
	return geecache.NewGroup(name, 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup("score")
	if api {
		go startAPIServer(apiAddr, gee)

	}
	startCacheServer(addrMap[port], []string(addrs), gee)

}
```
为了方便，我们编写一个`shell`脚本封装命令,命名为`run.sh`
```shell
#!/bin/bash
trap "rm server;kill 0" EXIT

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

wait
```
执行`run.sh`
