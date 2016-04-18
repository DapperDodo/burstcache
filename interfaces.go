package burstcache

/*
	Keyer implementations produce a key or hash given a request.
	This interface also includes the responsewriter, allowing communication between upstream
	handlers and keyers by using response headers. It also allows for keyers to
	write the key to downstream handlers or the client as a response header.
*/
type Keyer interface {
	Key(w http.ResponseWriter, r *http.Request) string
}

/*
	Chainer allows BurstCache to be used as standard http middleware
*/
type Chainer interface {
	Chain(next http.Handler) http.Handler
}
