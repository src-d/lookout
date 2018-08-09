package cache

import (
	"sync"

	"github.com/gregjones/httpcache"
	"gopkg.in/src-d/go-errors.v1"
)

var ErrNotFoundKey = errors.NewKind("Unable to find the given key, %s")

// ValidableCache represents a cache, were each new entry should be validate
// before be read or write a new entry, otherwise is forget and discarded.
type ValidableCache struct {
	httpcache.Cache

	// use regular mutex instead of sync.Map
	// because our case is different from what sync.Map is optimized for
	m     sync.Mutex
	inMem map[string][]byte // TODO(mcuadros): optimize memory usage
}

// NewValidableCache returns a new ValidableCache based on the given cache.
func NewValidableCache(cache httpcache.Cache) *ValidableCache {
	return &ValidableCache{Cache: cache, inMem: make(map[string][]byte)}
}

// Set stores the []byte representation of a response against a key. This
// data is stored in a temporal space, if isn't validate before `Set` is called
// again, the information get lost.
func (c *ValidableCache) Set(key string, responseBytes []byte) {
	c.m.Lock()
	defer c.m.Unlock()

	c.inMem[key] = responseBytes
}

// Validate validates the given key as a valid cache entry,
func (c *ValidableCache) Validate(key string) error {
	c.m.Lock()
	content, ok := c.inMem[key]
	c.m.Unlock()
	if !ok {
		return ErrNotFoundKey.New(key)
	}

	c.Cache.Set(key, content)

	return nil
}
