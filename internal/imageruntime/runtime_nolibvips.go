//go:build !libvips

package imageruntime

func EnsureStarted() error {
	return nil
}

func Shutdown() {}
