//go:build libvips

package imageruntime

import (
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

var (
	startupOnce sync.Once
	startupErr  error
)

func EnsureStarted() error {
	startupOnce.Do(func() {
		startupErr = vips.Startup(nil)
	})
	return startupErr
}

func Shutdown() {
	vips.Shutdown()
}
