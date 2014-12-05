package web

import (
	"io/ioutil"
	"net/http"
	"unsafe"

	"runtime"
	"runtime/debug"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const cacheItemSize = int(unsafe.Sizeof(CacheItem{}))
const cacheSize = int(unsafe.Sizeof(Cache{}))

type Cache struct {
	cache      map[string]*CacheItem
	size       int
	sizemax    int
	sizefree   int
	mutex      sync.RWMutex
	autodelete bool
	isRunning  bool
	Requests   chan CacheRequest
}

type CacheRequest struct {
	Url      string
	Response chan []byte
}

type CacheItem struct {
	Data        []byte
	AccessCount uint32
	Added       time.Time
}

func (c *CacheItem) GetSize() int {
	return cacheItemSize + len(c.Data) + int(unsafe.Sizeof(c.Data)) + int(unsafe.Sizeof(c.AccessCount)) + int(unsafe.Sizeof(c.Added))
}

func (c *CacheItem) SetNil() {
	c.Data = nil
}

func NewRequest(url string) CacheRequest {
	return CacheRequest{
		Url:      url,
		Response: make(chan []byte),
	}
}

func NewCache(cachesize int, sizefree int, autodelete bool) *Cache {
	c := &Cache{}
	c.sizemax = cachesize
	c.sizefree = sizefree
	c.autodelete = autodelete
	c.cache = make(map[string]*CacheItem)
	c.isRunning = false
	c.size = cacheSize
	c.Requests = make(chan CacheRequest, 8)
	return c
}

func (c *Cache) worker(requests <-chan CacheRequest) {
	for req := range requests {
		file := c.get(req.Url)
		if file == nil {
			ok := c.add(req.Url)
			if ok {
				file := c.get(req.Url)
				req.Response <- file
			} else {
				req.Response <- i404
			}

		} else {
			req.Response <- file
		}
	}

}

func (c *Cache) Stop() {
	close(c.Requests)
}

func (c *Cache) Start() {
	go c.worker(c.Requests)
	go c.worker(c.Requests)
	go c.worker(c.Requests)
	go c.worker(c.Requests)
}

func (c *Cache) GetSizeInMb() int {
	return (c.size / 1024 / 1024)
}

func (c *Cache) GetSize() int {
	return len(c.cache)
}

func (c *Cache) add(url string) (success bool) {
	//check if element exists
	success = false
	ok := c.exists(url)
	if ok {
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Add("Accept-Language", "de-DE")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Accept-Encoding", "gzip, deflate")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	by, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return false
	}

	file := by
	size := len(by)

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
	c.cache[url] = data
	c.size = c.size + data.GetSize()
	c.mutex.Unlock()
	success = true
	return
}

func (c *Cache) get(url string) []byte {
	ok := c.exists(url)
	if !ok {
		return nil
	}

	c.mutex.RLock()
	item, bl := c.cache[url]
	item.AccessCount++
	c.mutex.RUnlock()
	if bl {
		return item.Data
	}
	return nil
}

func (c *Cache) remove(url string) {
	c.mutex.Lock()
	val, ok := c.cache[url]
	if ok {
		c.size = c.size - val.GetSize()
		val.SetNil()
		delete(c.cache, url)
	}
	c.mutex.Unlock()
}

func (c *Cache) freeMemory() {
	if c.isRunning {
		return
	}
	c.isRunning = true
	log.WithField("tag", "cache").Infof("freeMemory() cachesize: %vmb, size to be freed: %vmb", c.GetSizeInMb(), (c.sizefree / 1024 / 1024))
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
						c.remove(key)
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
	log.WithField("tag", "cache").Infof("removed %v elements in %fsec", count-c.GetSize(), end.Sub(start).Seconds())
	start = time.Now()
	runtime.GC()
	debug.FreeOSMemory()
	end = time.Now()
	log.WithField("tag", "cache").Infof("run gc manually to free up memory asap, took %fsec", end.Sub(start).Seconds())
	c.isRunning = false
}

func (c *Cache) exists(url string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.cache[url]
	return ok
}
