package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/src-d/lookout/provider"
)

func TestWatcher_Watch(t *testing.T) {
	w := NewWatcher()
	err := w.Watch(context.TODO(), provider.WatchOptions{
		URL: "github.com/mcuadros/test",
	})

	fmt.Println(err)
}
