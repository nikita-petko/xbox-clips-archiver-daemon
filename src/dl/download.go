package dl

import (
	"errors"
	"fmt"
	"maps"
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

		return cache.CacheGuardedHttpGet(userId, strconv.Itoa(media.TitleId), media.Id, uri.URI)
	}

	return nil, fmt.Errorf("media %q has no usable download URI", media.Id)
}

type UserMediaCacheGuard struct {
	// Media is the media struct associated with the file.
	Media *models.Media

	// CachedFile is the path to the cached file associated with the media.
	CachedFile *cache.CacheGuard
}

func downloadAllMediaForUser(userId string, mediaList []*models.Media) ([]*UserMediaCacheGuard, error) {
	userMediaCacheGuards := make([]*UserMediaCacheGuard, 0)

	for _, media := range mediaList {
		cacheGuard, err := cacheDownloadMedia(userId, media)
		if err != nil {
			if errors.Is(err, cache.ErrExistsInCache) {
				glog.Warningf("media %q already exists in cache, skipping download", media.Id)
				continue
			}

			return nil, err
		}

		userMediaCacheGuards = append(userMediaCacheGuards, &UserMediaCacheGuard{
			Media:      media,
			CachedFile: cacheGuard,
		})
	}

	return userMediaCacheGuards, nil
}

// UserGroupedMedia represents a mapping from user ID to a list of media associated with that user.
type UserGroupedMedia map[string][]*UserMediaCacheGuard

// DownloadAllUserMedias downloads all media for all users in the provided map.
// It iterates over each user and attempts to download all media associated with them.
//
// Returns grouped media for each user, combined title ID to title name mapping, and any errors encountered during the download process.
func DownloadAllUserMedias(usersMap users.ApiKeyToUserMap) (UserGroupedMedia, dvr.TitleIdToTitleNameMap, error) {
	groupedMediaForAllUsers := make(UserGroupedMedia)
	combinedTitleIdToTitleNameMap := make(dvr.TitleIdToTitleNameMap)

	for apiKey, user := range usersMap {
		groupedMedia, titleIdToTitleNameMap, err := dvr.GetDVRMedia(apiKey)
		if err != nil {
			glog.Errorf("failed to get grouped DVR media for user %q: %v", user.UserID, err)
			continue
		}

		maps.Copy(combinedTitleIdToTitleNameMap, titleIdToTitleNameMap)

		cacheGuards, err := downloadAllMediaForUser(user.UserID, groupedMedia)
		if err != nil {
			glog.Errorf("failed to download media for user %q: %v", user.UserID, err)
			continue
		}

		groupedMediaForAllUsers[user.UserID] = cacheGuards
	}

	return groupedMediaForAllUsers, combinedTitleIdToTitleNameMap, nil
}
