package s3

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
)

// URI returns the S3 URI for key in the configured bucket.
func URI(key string) (string, error) {
	if gS3Client == nil {
		return "", errors.New("S3 is not set up")
	}
	if key == "" {
		return "", errors.New("S3 object key is required")
	}
	return (&url.URL{Scheme: "s3", Host: gS3Bucket, Path: "/" + key}).String(), nil
}

// AppendObjectKey appends an object key to a URL without changing the base path.
func AppendObjectKey(base *url.URL, key string) (*url.URL, error) {
	if base == nil {
		return nil, errors.New("base URL is required")
	}
	if key == "" {
		return nil, errors.New("S3 object key is required")
	}
	result := *base
	escapedBase := strings.TrimRight(base.EscapedPath(), "/")
	basePath, err := url.PathUnescape(escapedBase)
	if err != nil {
		return nil, fmt.Errorf("decode base URL path: %w", err)
	}
	result.Path = basePath + "/" + key
	// S3 interprets a literal plus in an object URL as a space.
	escapedKey := strings.ReplaceAll((&url.URL{Path: "/" + key}).EscapedPath(), "+", "%2B")
	result.RawPath = escapedBase + escapedKey
	return &result, nil
}

// HTTPSURL returns an unsigned regional HTTPS URL for the configured S3 object.
// Access to the object still depends on its S3 permissions.
func HTTPSURL(ctx context.Context, key string) (string, error) {
	if gS3Client == nil {
		return "", errors.New("S3 is not set up")
	}
	if key == "" {
		return "", errors.New("S3 object key is required")
	}
	endpoint, err := awss3.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, awss3.EndpointParameters{
		Region: aws.String(gS3Client.Options().Region), Bucket: aws.String(gS3Bucket), ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("resolve S3 HTTPS endpoint: %w", err)
	}
	objectURL, err := AppendObjectKey(&endpoint.URI, key)
	if err != nil {
		return "", err
	}
	return objectURL.String(), nil
}

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

	contentType := mime.TypeByExtension(strings.ToLower(path.Ext(key)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = gS3Uploader.Upload(ctx, &awss3.PutObjectInput{
		Bucket:      aws.String(gS3Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
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
