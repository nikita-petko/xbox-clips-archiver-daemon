package sns

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/golang/glog"
)

var (
	gSnsClient             *awssns.Client = nil
	gSetupSnsTopicOnceFlag sync.Once
)

// SetupSnsTopic sets up the SNS topic.
// Can only occur once.
func SetupSnsTopic() {
	gSetupSnsTopicOnceFlag.Do(func() {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(fmt.Sprintf("Error initializing AWS SNS Config: %v", err))
		}

		gSnsClient = awssns.NewFromConfig(cfg)

		glog.Info("AWS SNS setup complete!")
	})
}
