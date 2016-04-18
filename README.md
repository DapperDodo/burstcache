# burstcache

A http middleware cache optimized for short term (typically 1 sec or less) caching.

### TODO

Currently this is not even compiling. It has some dependencies with unpublished libraries. I need to remove all dependencies first.

### description

Burstcache is service middleware for api side caching

Burstcache is optimized for short term (typically 1 sec or less) caching in order to improve resilience against short bursts of traffic.

Burstcache keys consist of the api function url (without querystring)
and a scope UUID retrieved from context.

tips for tweaking:

TTL: consider this a throttle. how many requests per second can you easily handle?

	4 req/s -> TTL = 250 msec
	
	1 req/s -> TTL = 1 sec

TTD: how long can you tolerate stale data? subtract TTL from that.

	1 sec tolerable staleness at 4 req/sec -> 1 sec - 250 msec = 750 msec
	
	5 sec tolerable staleness at 1 req/sec -> 5 sec - 1 sec = 4 sec
	
TTD should be at least the avg time regeneration of the originating request takes, plus a couple of stddevs (or 5 if you're a scientist).

	avg req duration: 100 msec, stddev: 30 msec -> TTD should be at least 100+30+30 = 160 msec
