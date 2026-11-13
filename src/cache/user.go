package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path"
	"sync"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

const userCacheFileName = "user_profile_cache.json"

// UserProfile contains the cached Xbox user ID and gamertag.
type UserProfile struct {
	UserID   string `json:"user_id"`
	Gamertag string `json:"gamertag"`
}

var (
	userCacheMu   sync.RWMutex
	gUserProfiles map[string]UserProfile
)

func initUserCache() error {
	gUserProfiles = make(map[string]UserProfile)
	cacheFileName := path.Join(*flags.CachePath, userCacheFileName)

	data, err := os.ReadFile(cacheFileName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &gUserProfiles)
}

// GetUserProfile returns the profile cached for an API key.
func GetUserProfile(apiKey string) (UserProfile, bool) {
	userCacheMu.RLock()
	defer userCacheMu.RUnlock()

	profile, ok := gUserProfiles[userCacheKey(apiKey)]
	return profile, ok
}

// SetUserProfile caches the profile associated with an API key.
func SetUserProfile(apiKey string, profile UserProfile) error {
	userCacheMu.Lock()
	defer userCacheMu.Unlock()

	updatedCache := make(map[string]UserProfile, len(gUserProfiles)+1)
	maps.Copy(updatedCache, gUserProfiles)
	updatedCache[userCacheKey(apiKey)] = profile

	data, err := json.Marshal(updatedCache)
	if err != nil {
		return err
	}

	cacheFileName := path.Join(*flags.CachePath, userCacheFileName)
	if err := os.WriteFile(cacheFileName, data, 0644); err != nil {
		return err
	}

	gUserProfiles = updatedCache
	return nil
}

func userCacheKey(apiKey string) string {
	key := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(key[:])
}
