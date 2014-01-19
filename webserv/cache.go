package webserv

import (
	"io/ioutil"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	log "github.com/dvirsky/go-pylog/logging"
)

type Cache struct {
	cache      map[string]*CacheItem
	size       int
	sizemax    int
	sizefree   int
	mutex      sync.RWMutex
	autodelete bool
	isRunning  bool
}

type CacheItem struct {
	Data        []byte
	AccessCount uint32
	Added       time.Time
}

func (c *CacheItem) GetSize() int {
	return len(c.Data) + 4 + 16
}

func (c *CacheItem) SetNil() {
	c.Data = nil
}

func NewCache(cachesize int, sizefree int, autodelete bool) *Cache {
	c := &Cache{}
	c.sizemax = cachesize
	c.sizefree = sizefree
	c.autodelete = autodelete
	c.cache = make(map[string]*CacheItem)
	c.isRunning = false
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
	c.size = c.size + data.GetSize()
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
	item.AccessCount++
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
		c.size = c.size - val.GetSize()
		val.SetNil()
		delete(c.cache, filename)
	}
	c.mutex.Unlock()
}

func (c *Cache) freeMemory() {
	if c.isRunning {
		return
	}
	c.isRunning = true
	log.Info("[Cache] freeMemory() cachesize: %vmb, size to be freed: %vmb", c.GetSizeInMb(), (c.sizefree / 1024 / 1024))
	low := uint32(1)
	count := c.GetSize()
	start := time.Now()
	sec20 := int64(time.Second * 20)
	sec5 := int64(time.Second * 5)
	ignoreimmunity := false
	b := false
	for {
		for key, value := range c.cache {
			if c.sizemax-c.size < c.sizefree {
				if value.AccessCount <= low {
					//5 minute immunity to protect freshly added files
					if int64(start.Sub(value.Added)) >= sec20 || ignoreimmunity {
						c.Remove(key)
					}
				}
			} else {
				b = true
				break
			}
		}
		low++
		if int64(time.Now().Sub(start)) >= sec5 {
			ignoreimmunity = true
		}
		if b {
			break
		}
	}
	end := time.Now()
	log.Info("[Cache] removed %v elements in %fsec", count-c.GetSize(), end.Sub(start).Seconds())
	start = time.Now()
	runtime.GC()
	debug.FreeOSMemory()
	end = time.Now()
	log.Info("[Cache] run gc manually to free up memory asap, took %fsec", end.Sub(start).Seconds())
	c.isRunning = false
	//GoRuntimeStats()
}

func (c *Cache) exists(filename string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.cache[filename]
	return ok
}

func GoRuntimeStats() {
	m := &runtime.MemStats{}

	log.Info("# goroutines: %v", runtime.NumGoroutine())
	runtime.ReadMemStats(m)
	log.Info("Memory Acquired: %vmb", (m.Sys / 1024 / 1024))
	log.Info("Memory Used    : %vmb", (m.Alloc / 1024 / 1024))
}
