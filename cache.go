package burstcache

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

/*
	Burstcache is service middleware for api side caching

	Burstcache is optimized for short term (typically 1 sec or less) caching
	in order to improve resilience against short bursts of traffic.

	Burstcache keys consist of the api function url (without querystring)
	and a scope UUID retrieved from d.

	tips for tweaking:

	TTL: consider this a throttle. how many requests per second can you easily handle?
		4 req/s -> TTL = 250 msec
		1 req/s -> TTL = 1 sec

	TTD: how long can you tolerate stale data? subtract TTL from that.
		1 sec tolerable staleness at 4 req/sec -> 1 sec - 250 msec = 750 msec
		5 sec tolerable staleness at 1 req/sec -> 5 sec - 1 sec = 4 sec
	TTD should be at least the avg time regeneration of the originating request takes, plus a couple of stddevs (or 5 if you're a scientist).
		avg req duration: 100 msec, stddev: 30 msec -> TTD should be at least 100+30+30 = 160 msec
*/
type Cache struct {
	Keymaker Keyer // provides unique keys given the request parameters

	TTL time.Duration // time to live, amount of time before fresh caches becomes stale
	TTD time.Duration // time to die , amount of time before stale caches are killed

	mu     sync.RWMutex
	caches map[string]*ResponseCacher // caching responsewriter
}

/*
	Factory function
*/
func NewCache(keymaker Keyer, l api.ILogger, ttl time.Duration, ttd time.Duration) *Cache {

	return &Cache{
		Keymaker: keymaker,
		L:        l,
		TTL:      ttl,
		TTD:      ttd,
		caches:   map[string]*ResponseCacher{},
	}
}

// implement Chainer, can be used as http middleware

func (c *Cache) Chain(next http.Handler) http.Handler {

	f := func(w http.ResponseWriter, r *http.Request) {

		key := c.Keymaker.Key(w, r)

		exists, _, fresh, regen := c.status(key)

		if !exists {

			// fill cache and wait for it
			c.regenerate(next, key, w, r)

			// serve from cache without marking the response
			c.serve(key, w, false)
			return
		}

		if !fresh && !regen {

			// mark this cache is regenerating so other requests don't stampede
			c.regen(key)

			// refill cache but this time do not wait for it
			go c.regenerate(next, key, w, r)
		}

		// serve from cache, marking the response as cached
		c.serve(key, w, true)
		return
	}
	return service.HandlerFunc(f)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////
// private parts
///////////////////////////////////////////////////////////////////////////////////////////////////////

func (c *Cache) regenerate(next http.Handler, key string, w http.ResponseWriter, r *http.Request) {

	// TODO: make proper UUID
	id := rand.Intn(1000000)

	cache := NewResponseCacher(id)

	// down the rabbit hole......
	next.ServeHTTP(cache, r)

	// swap stale with fresh result
	c.swap(key, cache)

	// schedule two-phase cache expiration
	go func() {

		// when ttl expires, cache becomes stale
		time.Sleep(c.TTL)
		exists, stale_id, fresh, _ := c.status(key)
		if exists && fresh {
			if stale_id == id {
				c.stale(key)
			}
		}

		// when ttd expires, cache is killed
		time.Sleep(c.TTD)
		exists, stale_id, fresh, regen := c.status(key)
		if exists && !fresh && !regen {
			if stale_id == id {
				c.kill(key)
			}
		}
	}()

	// success!
	return
}

/*
	Replace a stale cache with a newly filled response
*/
func (c *Cache) swap(key string, cache *ResponseCacher) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caches[key] = cache
}

/*
	Mark the cache as stale. In this state a subsequent request may start a regeneration.
*/
func (c *Cache) stale(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caches[key].fresh = false
}

/*
	kill the cache, removing the response completely from the map
*/
func (c *Cache) kill(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caches[key] = nil
}

/*
	Mark the cache as regenerating. This will prevent other requests from
	starting a regeneration.
*/
func (c *Cache) regen(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caches[key].regen = true
}

/*
	serve the cached response as an actual response.
	When mark==true a header will be set to mark the response as a cached one.
*/
func (c *Cache) serve(key string, w http.ResponseWriter, mark bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.caches[key].Serve(w, mark)
}

/*
	status returns
	- whether a cache exists (initialized)
	- its id (unique identifier)
	- is still fresh
	- is being regenerated
*/
func (c *Cache) status(key string) (exists bool, id int, fresh bool, regen bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cache, ok := c.caches[key]
	if ok && cache != nil {
		return true, cache.id, cache.fresh, cache.regen
	}
	return false, 0, false, false
}
