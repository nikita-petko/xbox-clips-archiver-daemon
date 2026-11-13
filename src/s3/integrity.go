package s3

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
)

// ValidateIntegrity compares a cached file to the checksum stored for an S3 key.
// It returns false for a checksum mismatch and an error when verification cannot
// be performed. S3 checksum metadata is required; ETags are not checksums. For
// SSE-KMS objects, retrieving checksums can require additional KMS permissions.
func ValidateIntegrity(ctx context.Context, key string, guard *cache.CacheGuard) (bool, error) {
	if gS3Client == nil {
		return false, errors.New("S3 is not set up")
	}
	if key == "" {
		return false, errors.New("S3 object key is required")
	}
	if guard == nil {
		return false, errors.New("cache guard is required")
	}

	file, err := os.Open(guard.GetFilePath())
	if err != nil {
		return false, fmt.Errorf("open cached file for S3 integrity check: %w", err)
	}
	defer file.Close()

	metadata, err := gS3Client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(gS3Bucket), Key: aws.String(key), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return false, fmt.Errorf("retrieve S3 checksum: %w", err)
	}
	h, encoded, err := objectChecksum(metadata)
	if err != nil {
		return false, err
	}

	base, partSuffix, hasSuffix := strings.Cut(encoded, "-")
	partCount := 0
	if hasSuffix {
		partCount, err = strconv.Atoi(partSuffix)
		if err != nil || partCount < 1 || partCount > 10000 {
			return false, errors.New("invalid S3 composite checksum part count")
		}
	}
	switch metadata.ChecksumType {
	case types.ChecksumTypeComposite:
		if !hasSuffix {
			return false, errors.New("S3 composite checksum has no part count")
		}
	case types.ChecksumTypeFullObject:
		if hasSuffix {
			return false, errors.New("S3 full-object checksum has a part count")
		}
	case "":
		// Older objects may not expose a checksum type; the suffix identifies composites.
	default:
		return false, fmt.Errorf("unsupported S3 checksum type %q", metadata.ChecksumType)
	}
	expected, err := base64.StdEncoding.Strict().DecodeString(base)
	if err != nil || len(expected) != h.Size() {
		return false, errors.New("invalid S3 checksum encoding or length")
	}

	reader := &contextReader{ctx: ctx, reader: file}
	if hasSuffix {
		if aws.ToString(metadata.VersionId) == "" && aws.ToString(metadata.ETag) == "" {
			return false, errors.New("S3 object version or ETag is required to verify a composite checksum")
		}
		composite := h
		for i := 1; i <= partCount; i++ {
			part, err := gS3Client.HeadObject(ctx, &awss3.HeadObjectInput{
				Bucket: aws.String(gS3Bucket), Key: aws.String(key), PartNumber: aws.Int32(int32(i)),
				VersionId: metadata.VersionId, IfMatch: metadata.ETag,
			})
			if err != nil {
				return false, fmt.Errorf("retrieve S3 part %d metadata: %w", i, err)
			}
			if part.ContentLength == nil || *part.ContentLength < 0 {
				return false, fmt.Errorf("S3 part %d has no valid length", i)
			}
			partHash, _, err := objectChecksum(metadata)
			if err != nil {
				return false, err
			}
			if _, err := io.CopyN(partHash, reader, *part.ContentLength); err != nil {
				if errors.Is(err, io.EOF) {
					return false, nil
				}
				return false, fmt.Errorf("read cached file for S3 integrity check: %w", err)
			}
			_, _ = composite.Write(partHash.Sum(nil))
		}
		var extra [1]byte
		n, err := reader.Read(extra[:])
		if err != nil && !errors.Is(err, io.EOF) {
			return false, fmt.Errorf("read cached file for S3 integrity check: %w", err)
		}
		if n != 0 {
			return false, nil
		}
	} else if _, err := io.Copy(h, reader); err != nil {
		return false, fmt.Errorf("read cached file for S3 integrity check: %w", err)
	}
	return bytes.Equal(h.Sum(nil), expected), nil
}

func objectChecksum(metadata *awss3.HeadObjectOutput) (hash.Hash, string, error) {
	switch {
	case metadata.ChecksumSHA256 != nil:
		return sha256.New(), *metadata.ChecksumSHA256, nil
	case metadata.ChecksumSHA1 != nil:
		return sha1.New(), *metadata.ChecksumSHA1, nil
	case metadata.ChecksumCRC64NVME != nil:
		return crc64.New(crc64.MakeTable(0x9a6c9329ac4bc9b5)), *metadata.ChecksumCRC64NVME, nil
	case metadata.ChecksumCRC32C != nil:
		return crc32.New(crc32.MakeTable(crc32.Castagnoli)), *metadata.ChecksumCRC32C, nil
	case metadata.ChecksumCRC32 != nil:
		return crc32.NewIEEE(), *metadata.ChecksumCRC32, nil
	default:
		return nil, "", errors.New("S3 object has no supported checksum")
	}
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
