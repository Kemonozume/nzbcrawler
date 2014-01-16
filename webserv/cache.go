package webserv

import (
	"io/ioutil"
	"sync"
	"time"
)

type Cache struct {
	cache      map[string]*CacheItem
	size       int
	sizemax    int
	sizefree   int
	mutex      sync.RWMutex
	autodelete bool
}

type CacheItem struct {
	Data        []byte
	AccessCount uint
	Added       time.Time
}

func (c *CacheItem) IncreaseHits() {
	c.AccessCount++
}

func NewCache(cachesize int, sizefree int, autodelete bool) *Cache {
	c := &Cache{}
	c.sizemax = cachesize
	c.sizefree = sizefree
	c.autodelete = autodelete
	c.cache = make(map[string]*CacheItem)
	return c
}

func (c *Cache) GetSizeInMb() int {
	return (c.size / 1024 / 1024)
}

func (c *Cache) GetSize() int {
	return len(c.cache)
}

func (c *Cache) Add(filename string) (success bool) {
	//check if element exists
	success = false
	ok := c.exists(filename)
	if ok {
		return
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	size := len(file)

	//check if the cache is full
	if c.size+size > c.sizemax {
		if c.autodelete {
			c.freeMemory()
		} else {
			return
		}
	}

	c.mutex.Lock()
	data := &CacheItem{file, 1, time.Now()}
	c.cache[filename] = data
	c.size = c.size + size
	c.mutex.Unlock()
	success = true
	return
}

func (c *Cache) Get(filename string) []byte {
	ok := c.exists(filename)
	if !ok {
		return nil
	}

	c.mutex.RLock()
	item, bl := c.cache[filename]
	item.IncreaseHits()
	c.mutex.RUnlock()
	if bl {
		return item.Data
	}
	return nil
}

func (c *Cache) Remove(filename string) {
	c.mutex.Lock()
	val, ok := c.cache[filename]
	if ok {
		delete(c.cache, filename)
		c.size = c.size - len(val.Data)
	}
	c.mutex.Unlock()
}

func (c *Cache) freeMemory() {
	low := uint(1)
	b := false
	now := time.Now()
	min5 := int64(time.Minute * 5)
	for {
		for key, value := range c.cache {
			if c.sizemax-c.size < c.sizefree {
				if value.AccessCount <= low {
					//5 minute immunity to protect freshly added files
					if int64(now.Sub(value.Added)) >= min5 {
						c.mutex.Lock()
						size := len(value.Data)
						c.size = c.size - size
						delete(c.cache, key)
						c.mutex.Unlock()
					}
				}
			} else {
				b = true
				break
			}
		}
		low++
		if b {
			break
		}
	}
}

func (c *Cache) exists(filename string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.cache[filename]
	return ok
}
