package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/hashicorp/go-retryablehttp"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

// PublishEmbed publishes an embed to the configured
// WebHook. If color is 0x0, it will default to
// 0x3498db.
func PublishEmbed(ctx context.Context, title, description string, color int) {
	if gDiscordHttpClient == nil {
		return // not configured
	}

	if color == 0 {
		color = 0x3498db
	}

	data := &postData{
		Embeds: []*embed{
			{
				Title:       title,
				Description: description,
				Color:       color,
			},
		},
	}

	if *flags.DiscordAlertRoleId > 0 {
		data.Content = fmt.Sprintf("<@&%d>", *flags.DiscordAlertRoleId)
		data.AllowedMentions = new(allowedMentions)
		data.AllowedMentions.Roles = []uint64{*flags.DiscordAlertRoleId}
	}

	marshalledData, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Sprintf("Error while marshalling request: %v", err))
	}

	request, err := retryablehttp.NewRequestWithContext(ctx, "POST", *flags.DiscordWebHookUri, bytes.NewBuffer(marshalledData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		panic(fmt.Sprintf("Error while creating request: %v", err))
	}

	response, err := gDiscordHttpClient.Do(request)
	if err != nil {
		glog.Errorf("Error while sending request: %v", err)
		return
	}

	if response.StatusCode != 204 {
		respBytes, _ := io.ReadAll(response.Body)

		glog.Errorf("Error when sending Discord embed: %s", string(respBytes))
	}
}
