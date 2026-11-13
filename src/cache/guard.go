package cache

import (
	"errors"
	"io"
	gourl "net/url"
	"os"
	"path"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/http"
)

var ErrExistsInCache = errors.New("file already exists in cache")

// CacheGuard represents a cached file and provides methods to access it.
type CacheGuard struct {
	filePath string
}

// GetFilePath returns the file path of the cached file.
func (cg *CacheGuard) GetFilePath() string {
	return cg.filePath
}

// Get reads and returns the contents of the cached file.
func (cg *CacheGuard) Get() ([]byte, error) {
	return os.ReadFile(cg.filePath)
}

func resolveCachePath(userId, titleId, mediaId, fileName string) string {
	return path.Join(*flags.CachePath, userId, titleId, mediaId, fileName)
}

// CacheGuardedHttpGet makes an HTTP GET request to the specified URL
// If the file is already cached, it returns ErrExistsInCache. Otherwise, it downloads the file and caches it.
func CacheGuardedHttpGet(userId, titleId, mediaId, url string) (*CacheGuard, error) {
	parsedUri, err := gourl.Parse(url)
	if err != nil {
		return nil, err
	}

	fileName := path.Base(parsedUri.Path)

	resolvedPath := resolveCachePath(userId, titleId, mediaId, fileName)

	glog.V(10000).Infof("Resolved cache path for User ID, Title ID, and Media ID: %s@%s@%s, resolved path: %s", userId, titleId, mediaId, resolvedPath)

	if isCached(userId, titleId, mediaId) {
		return &CacheGuard{filePath: resolvedPath}, ErrExistsInCache
	}

	// If the file is not cached or the ETag does not match, cache the response body to a file
	resp, err := http.Client().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Cache the response body to a file
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(path.Dir(resolvedPath), os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(resolvedPath, bytes, 0644)
	if err != nil {
		return nil, err
	}

	addToCache(userId, titleId, mediaId)

	return &CacheGuard{filePath: resolvedPath}, nil
}
