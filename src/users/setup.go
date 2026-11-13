package users

import (
	"strings"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

// Setup initializes the global users map by fetching users based on the provided API keys. It panics if no API keys are provided or if fetching users fails.
func Setup() {
	apiKeys := strings.Split(*flags.XblApiKeys, ",")
	if len(apiKeys) == 0 || strings.TrimSpace(*flags.XblApiKeys) == "" {
		panic("no API keys provided")
	}
	users, err := fetchUsersByUserIds(apiKeys)
	if err != nil {
		panic(err)
	}

	gUsersMap = users

	if len(gUsersMap) == 0 {
		panic("no users found")
	}

	for _, user := range gUsersMap {
		glog.Infof("Loaded user: %s (%s)", user.Gamertag, user.UserID)
	}
}
