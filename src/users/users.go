package users

var gUsersMap ApiKeyToUserMap

// GetUsersMap returns the global users map, which maps API keys to user information.
func GetUsersMap() ApiKeyToUserMap {
	return gUsersMap
}
