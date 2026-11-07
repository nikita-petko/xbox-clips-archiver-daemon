package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

// PublishToSnsTopic publishes a message to the configured SNS topic.
func PublishToSnsTopic(ctx context.Context, body string) {
	if gSnsClient == nil {
		return
	}

	_, err := gSnsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: flags.SnsTopicArn,
		Message:  &body,
	})

	if err != nil {
		glog.Errorf("Error occurred when publishing to SNS Topic: %v (topic ARN: %s)", err, *flags.SnsTopicArn)
	}
}
