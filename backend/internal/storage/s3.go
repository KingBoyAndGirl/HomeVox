package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Object struct {
	Data        []byte
	ContentType string
	Size        int64
}

type ObjectStore interface {
	PutObject(ctx context.Context, key string, contentType string, data []byte) error
	GetObject(ctx context.Context, key string) (Object, error)
	DeleteObject(ctx context.Context, key string) error
	VerifyBucket(ctx context.Context) error
	ObjectURL(key string) string
}

var ErrObjectNotFound = fmt.Errorf("object not found")

type s3Store struct {
	client         *s3.Client
	bucket         string
	endpoint       string
	pathStyle      bool
	baseObjectPath string
}

func NewS3Store(cfg Config) (ObjectStore, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("s3 config is incomplete")
	}

	parsedEndpoint, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse S3 endpoint: %w", err)
	}

	endpointResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{PartitionID: "aws", URL: cfg.Endpoint, SigningRegion: region}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(endpointResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &s3Store{
		client:    s3.NewFromConfig(awsCfg, func(o *s3.Options) { o.UsePathStyle = true }),
		bucket:    cfg.Bucket,
		endpoint:  strings.TrimRight(parsedEndpoint.String(), "/"),
		pathStyle: true,
		baseObjectPath: path.Join(
			strings.TrimRight(parsedEndpoint.EscapedPath(), "/"),
			cfg.Bucket,
		),
	}, nil
}

func (s *s3Store) PutObject(ctx context.Context, key string, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
		Metadata:    map[string]string{"x-project-origin": "homevox"},
	})
	return err
}

func (s *s3Store) GetObject(ctx context.Context, key string) (Object, error) {
	response, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if isS3NotFound(err) {
			return Object{}, ErrObjectNotFound
		}
		return Object{}, err
	}
	defer response.Body.Close()

	contentType := aws.ToString(response.ContentType)
	size := int64(0)
	if response.ContentLength != nil {
		size = *response.ContentLength
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return Object{}, err
	}
	return Object{Data: data, ContentType: contentType, Size: size}, nil
}

func (s *s3Store) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return err
	}
	return nil
}

func (s *s3Store) VerifyBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err != nil {
		if isS3NotFound(err) {
			return fmt.Errorf("bucket %s is not accessible", s.bucket)
		}
		return err
	}
	return nil
}

func (s *s3Store) ObjectURL(key string) string {
	u, err := url.Parse(s.endpoint)
	if err != nil {
		return "/"
	}

	if s.pathStyle {
		u.Path = path.Join(s.baseObjectPath, key)
		return u.String()
	}

	u.Path = path.Join("/", key)
	return u.String()
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound *s3types.NoSuchKey
	var noBucket *s3types.NoSuchBucket
	if strings.Contains(err.Error(), "NoSuchKey") {
		return true
	}
	return errors.As(err, &notFound) || errors.As(err, &noBucket) || strings.Contains(err.Error(), "404")
}

// Config defines the minimal S3 client settings used by HomeVox.
type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}
