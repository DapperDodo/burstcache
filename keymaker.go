package burstcache

import (
	"net/http"
)

/*
	Keymaker is the vanilla implementation of Keyer.
	It generates cache keys based on the url path only.
	Feel free to implement your own keymaker and inject that into your burstcache.
*/
type Keymaker struct{}

func (k *Keymaker) Key(w http.ResponseWriter, r *http.Request) string {

	key := r.URL.Path

	return key
}
