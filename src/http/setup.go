package http

import (
	"github.com/golang/glog"
	"github.com/hashicorp/go-retryablehttp"
)

var gHttpClient = retryablehttp.NewClient()

func Setup() {
	gHttpClient.Logger = &glogAdapter{}

	glog.Info("HTTP client setup complete")
}
