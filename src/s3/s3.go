package s3

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
)

// Upload uploads the cached file to key. An existing object may be overwritten.
func Upload(ctx context.Context, key string, guard *cache.CacheGuard) error {
	if gS3Uploader == nil {
		return errors.New("S3 is not set up")
	}
	if key == "" {
		return errors.New("S3 object key is required")
	}
	if guard == nil {
		return errors.New("cache guard is required")
	}

	file, err := os.Open(guard.GetFilePath())
	if err != nil {
		return fmt.Errorf("open cached file for S3 upload: %w", err)
	}
	defer file.Close()

	_, err = gS3Uploader.Upload(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(gS3Bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("upload cached file to S3: %w", err)
	}
	return nil
}

// Exists reports whether key exists. Access and service errors are returned.
func Exists(ctx context.Context, key string) (bool, error) {
	if gS3Client == nil {
		return false, errors.New("S3 is not set up")
	}
	if key == "" {
		return false, errors.New("S3 object key is required")
	}
	_, err := gS3Client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(gS3Bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}

	var responseErr *smithyhttp.ResponseError
	if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == 404 {
		// HeadObject also returns 404 for a missing bucket.
		_, bucketErr := gS3Client.HeadBucket(ctx, &awss3.HeadBucketInput{Bucket: aws.String(gS3Bucket)})
		if bucketErr != nil {
			return false, fmt.Errorf("check S3 bucket after missing object: %w", bucketErr)
		}
		return false, nil
	}
	return false, fmt.Errorf("check S3 object: %w", err)
}
