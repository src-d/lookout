package cache

import (
	"testing"

	"github.com/gregjones/httpcache"
	"github.com/stretchr/testify/require"
)

func TestValidable_SetAndValidate(t *testing.T) {
	require := require.New(t)

	cache := NewValidableCache(httpcache.NewMemoryCache())
	cache.Set("foo", []byte("qux"))

	data, ok := cache.Get("foo")
	require.False(ok)
	require.Nil(data)

	cache.Validate("foo")

	data, ok = cache.Get("foo")
	require.True(ok)
	require.Equal(data, []byte("qux"))
}
