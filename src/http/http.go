package http

import "github.com/hashicorp/go-retryablehttp"

// Client returns the global HTTP client instance used for making requests.
func Client() *retryablehttp.Client {
	return gHttpClient
}
