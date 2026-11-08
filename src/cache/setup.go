package cache

import (
	"os"
	"path"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

var lock *lockFile

// Setup initializes the cache by ensuring that the cache directory exists.
// If the cache path is not provided, it logs a fatal error and exits the program.
// If the cache directory does not exist, it creates it with appropriate permissions.
func Setup() {
	if *flags.CachePath == "" {
		glog.Fatalf("Cache path is required. Please provide a cache path using the --cache-path flag or the CACHE_PATH environment variable.")

		return
	}

	if *flags.ClearCache {
		err := os.RemoveAll(*flags.CachePath)
		if err != nil && !os.IsNotExist(err) {
			glog.Fatalf("Failed to clear cache directory: %v", err)
		}
	}

	// Create cache directory if it doesn't exist
	err := os.MkdirAll(*flags.CachePath, os.ModePerm)
	if err != nil {
		glog.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create cache lock file
	lockFilePath := path.Join(*flags.CachePath, "cache.lock")
	lock, err = tryLock(lockFilePath)
	if err != nil {
		glog.Fatalf("Failed to acquire lock on cache directory: %v", err)
	}

	initCache()

	glog.Infof("Cache setup complete. Cache directory: %s", *flags.CachePath)
}

// Close releases the lock on the cache directory and performs any necessary cleanup.
func Close() {
	if lock != nil {
		err := lock.Unlock()
		if err != nil {
			glog.Errorf("Failed to release lock on cache directory: %v", err)
		}
	}
}
