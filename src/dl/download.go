package dl

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/dvr"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/users"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

func cacheDownloadMedia(userId string, media *models.Media) (*cache.CacheGuard, error) {
	if media == nil {
		return nil, fmt.Errorf("cannot download nil media")
	}

	if len(media.MediaUris) == 0 {
		return nil, nil
	}

	for _, uri := range media.MediaUris {
		if uri == nil || uri.URI == "" {
			continue
		}

		// Format the size in human readable
		size := ""
		if uri.FileSize > 0 {
			size = fmt.Sprintf("%.2f MB", float64(uri.FileSize)/(1024*1024))
		}

		glog.Infof(
			"Downloading media %q for user %q (size: %s)",
			media.Id,
			userId,
			size,
		)

		return cache.CacheGuardedHttpGet(userId, strconv.Itoa(media.TitleId), media.Id, uri.URI)
	}

	return nil, fmt.Errorf("media %q has no usable download URI", media.Id)
}

type UserMediaCacheGuard struct {
	// Gamertag is the Xbox gamertag of the user who owns the media.
	Gamertag string

	// WasCached indicates whether the media was already present in the cache.
	// This is used in the upload flow to determine if an already downloaded
	// file should be re-uploaded because it actually didn't get uploaded.
	//
	// This is to fine grain the remote HEAD check to reduce the calls as
	// non-cached media was most likely not uploaded yet.
	WasCached bool

	// Media is the media struct associated with the file.
	Media *models.Media

	// CachedFile is the path to the cached file associated with the media.
	CachedFile *cache.CacheGuard
}

func downloadAllMediaForUser(user users.User, mediaList []*models.Media) ([]*UserMediaCacheGuard, error) {
	userMediaCacheGuards := make([]*UserMediaCacheGuard, 0)

	for _, media := range mediaList {
		cacheGuard, err := cacheDownloadMedia(user.UserID, media)
		if err != nil && !errors.Is(err, cache.ErrExistsInCache) {
			return nil, err
		}

		userMediaCacheGuards = append(userMediaCacheGuards, &UserMediaCacheGuard{
			Gamertag:   user.Gamertag,
			Media:      media,
			CachedFile: cacheGuard,
			WasCached:  errors.Is(err, cache.ErrExistsInCache),
		})
	}

	return userMediaCacheGuards, nil
}

// UserGroupedMedia represents a mapping from user ID to a list of media associated with that user.
type UserGroupedMedia map[string][]*UserMediaCacheGuard

// DownloadUserMedia fetches and downloads the media belonging to one user.
func DownloadUserMedia(apiKey string, user users.User) ([]*UserMediaCacheGuard, error) {
	media, err := dvr.GetDVRMedia(apiKey)
	if err != nil {
		return nil, fmt.Errorf("get DVR media: %w", err)
	}
	guards, err := downloadAllMediaForUser(user, media)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	return guards, nil
}

// DownloadAllUserMedias downloads all media for all users in the provided map.
// It iterates over each user and attempts to download all media associated with them.
//
// Errors for individual users are logged and skipped.
func DownloadAllUserMedias(usersMap users.ApiKeyToUserMap) (UserGroupedMedia, error) {
	groupedMediaForAllUsers := make(UserGroupedMedia)

	for apiKey, user := range usersMap {
		cacheGuards, err := DownloadUserMedia(apiKey, user)
		if err != nil {
			glog.Errorf("failed to download media for user %q: %v", user.UserID, err)
			continue
		}

		groupedMediaForAllUsers[user.UserID] = cacheGuards
	}

	return groupedMediaForAllUsers, nil
}
