package discord

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	gDiscordHttpClient          *retryablehttp.Client = nil
	gSetupDiscordClientOnceFlag sync.Once
)

// SetupDisciordClient sets up the Discord HTTP Client.
// Can only occur once.
func SetupDiscordClient() {
	gSetupDiscordClientOnceFlag.Do(func() {
		gDiscordHttpClient = retryablehttp.NewClient()
		gDiscordHttpClient.HTTPClient.Timeout = 15 * time.Second
		gDiscordHttpClient.Logger = nil

		glog.Info("Discord WebHook setup complete!")
	})
}
