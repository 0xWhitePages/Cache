package geecache

import (
	"fmt"
	"log"
	"testing"
)

//模拟一个数据库
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGroup_Get(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	geecache := NewGroup("stu_scores", 2048, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[MongoDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))

	for k, v := range db {
		if view, err := geecache.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		if _, err := geecache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := geecache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
