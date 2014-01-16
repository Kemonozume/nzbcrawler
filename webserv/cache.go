package webserv

import (
	"io/ioutil"
	"sync"
)

type Cache struct {
	cache      map[string][]byte
	size       int
	sizemax    int
	mutex      sync.RWMutex
	autodelete bool
}

func NewCache(cachesize int, autodelete bool) *Cache {
	c := &Cache{}
	c.SetMaxSize(cachesize)
	c.setAutoDelete(autodelete)
	return c
}

func (c *Cache) Init() {
	c.cache = make(map[string][]byte)
}

func (c *Cache) setAutoDelete(adelete bool) {
	c.autodelete = adelete
}

func (c *Cache) SetMaxSize(cachesize int) {
	c.sizemax = cachesize
}

func (c *Cache) GetSizeInMb() int {
	return (c.size / 1024 / 1024)
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
			for c.size+size > c.sizemax {
				c.freeMemory()
			}
		} else {
			return
		}
	}

	c.mutex.Lock()
	c.cache[filename] = file
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
	file, bl := c.cache[filename]
	c.mutex.RUnlock()
	if bl {
		return file
	}
	return nil
}

func (c *Cache) Remove(filename string) {
	c.mutex.Lock()
	val, ok := c.cache[filename]
	if ok {
		delete(c.cache, filename)
		c.size = c.size - len(val)
	}
	c.mutex.Unlock()
}

func (c *Cache) freeMemory() {
	for key, value := range c.cache {
		if c.sizemax-c.size < 21000000 {
			c.mutex.Lock()
			size := len(value)
			c.size = c.size - size
			delete(c.cache, key)
			c.mutex.Unlock()
		} else {
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
