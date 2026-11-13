package xbox

import (
	"encoding/json"
	"io"
	gohttp "net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/http"
)

func makeApiCall(endpoint string, apiKey string, isPaged bool) (response *gohttp.Response, ct string, err error) {
	uri, err := url.Parse(XBL_API_URL + endpoint)
	if err != nil {
		return nil, "", err
	}

	request, err := retryablehttp.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set(XBL_API_AUTH_HEADER, apiKey)

	response, err = http.Client().Do(request)
	if err != nil {
		return nil, "", err
	}

	// Read response as a paged response if necessary
	if isPaged {
		// Read as dictionary, continuation token lies at:
		// response_body.content.pagingInfo.continuationToken

		data, err := io.ReadAll(response.Body)
		if err != nil {
			return response, "", err
		}

		var jsonResponse map[string]interface{}
		err = json.Unmarshal(data, &jsonResponse)
		if err != nil {
			return response, "", err
		}

		if content, ok := jsonResponse["content"].(map[string]interface{}); ok {
			if pagingInfo, ok := content["pagingInfo"].(map[string]interface{}); ok {
				if continuationToken, ok := pagingInfo["continuationToken"].(string); ok {
					return response, continuationToken, nil
				}
			}
		}
	}

	return response, "", nil
}
