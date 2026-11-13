package s3

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

var (
	gS3Client    *awss3.Client
	gS3Uploader  *manager.Uploader
	gS3Bucket    string
	gSetupS3Once sync.Once
)

// Setup configures S3 from the application flags. It can only occur once.
func Setup() {
	gSetupS3Once.Do(func() {
		if *flags.S3Bucket == "" {
			panic("S3 bucket is required")
		}
		if *flags.S3Region == "" {
			panic("S3 region is required")
		}
		accessKey, secretKey := *flags.S3AccessKey, *flags.S3SecretAccessKey
		if (accessKey == "") != (secretKey == "") {
			panic("Both S3 access key and secret access key are required when using explicit credentials")
		}

		options := []func(*config.LoadOptions) error{config.WithRegion(*flags.S3Region)}
		if accessKey != "" {
			options = append(options, config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, os.Getenv("AWS_SESSION_TOKEN")),
			))
		}
		cfg, err := config.LoadDefaultConfig(context.TODO(), options...)
		if err != nil {
			panic(fmt.Sprintf("Error initializing AWS S3 config: %v", err))
		}

		gS3Client = awss3.NewFromConfig(cfg)
		gS3Uploader = manager.NewUploader(gS3Client)
		gS3Bucket = *flags.S3Bucket
		glog.Info("AWS S3 setup complete!")
	})
}
