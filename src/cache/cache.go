package cache

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

const downloadCacheMaxAge = 30 // Max age in days for downloaded media entries
const downloadCacheFileName = "download_cache.json"

var gDownloadedMedia map[string]string // Maps {userId}:{titleId}:{mediaId} to the date they were downloaded. On setup, expired media entries are removed.

func initCache() {
	gDownloadedMedia = make(map[string]string)
	cacheFileName := path.Join(*flags.CachePath, downloadCacheFileName)

	// Load existing download cache from file if it exists
	if _, err := os.Stat(cacheFileName); err == nil {
		data, err := os.ReadFile(cacheFileName)
		if err == nil {
			json.Unmarshal(data, &gDownloadedMedia)
		}

		// Remove expired media entries from the cache
		for cacheKey, downloadDate := range gDownloadedMedia {
			if isExpired(downloadDate) {
				delete(gDownloadedMedia, cacheKey)
				saveCache()
			}
		}
	}
}

func saveCache() {
	cacheFileName := path.Join(*flags.CachePath, downloadCacheFileName)

	data, err := json.Marshal(gDownloadedMedia)
	if err == nil {
		os.WriteFile(cacheFileName, data, 0644)
	}
}

func isExpired(downloadDate string) bool {
	// Parse the downloadDate
	parsedDate, err := time.Parse("2006-01-02", downloadDate)
	if err != nil {
		return true // If the date cannot be parsed, consider it expired
	}

	// Check if the download date is older than the max age
	expirationDate := parsedDate.AddDate(0, 0, downloadCacheMaxAge)

	return time.Now().After(expirationDate)
}

func deriveCacheKey(userID, titleID, mediaID string) string {
	return userID + ":" + titleID + ":" + mediaID
}

// isCached checks if a media ID is present in the download cache and not expired.
func isCached(userID, titleID, mediaID string) bool {
	cacheKey := deriveCacheKey(userID, titleID, mediaID)

	downloadDate, exists := gDownloadedMedia[cacheKey]
	if !exists {
		return false
	}

	if isExpired(downloadDate) {
		delete(gDownloadedMedia, cacheKey)
		saveCache()
		return false
	}

	return true
}

// addToCache adds a media ID to the download cache with the current date.
func addToCache(userID, titleID, mediaID string) {
	gDownloadedMedia[deriveCacheKey(userID, titleID, mediaID)] = time.Now().Format("2006-01-02")
	saveCache()
}
