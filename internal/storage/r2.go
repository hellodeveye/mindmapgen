package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var ErrMissingR2Config = errors.New("missing R2 storage configuration")

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	Domain          string
}

type R2Client struct {
	client     *s3.Client
	bucketName string
	domain     string
}

// LoadR2ConfigFromEnv reads the standard R2_* environment variables and returns
// a configuration struct. If any required value is missing, ErrMissingR2Config
// is returned.
func LoadR2ConfigFromEnv() (R2Config, error) {
	cfg := R2Config{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("R2_ACCESS_KEY_SECRET"),
		BucketName:      os.Getenv("R2_BUCKET_NAME"),
		Domain:          os.Getenv("R2_DOMAIN"),
	}

	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.BucketName == "" || cfg.Domain == "" {
		return R2Config{}, ErrMissingR2Config
	}

	return cfg, nil
}

// NewR2ClientFromEnv constructs an R2 client using environment variables.
func NewR2ClientFromEnv() (*R2Client, error) {
	cfg, err := LoadR2ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewR2Client(cfg)
}

func NewR2Client(cfg R2Config) (*R2Client, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID),
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.AccessKeySecret,
			"",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %v", err)
	}

	client := s3.NewFromConfig(awsCfg)
	return &R2Client{
		client:     client,
		bucketName: cfg.BucketName,
		domain:     cfg.Domain,
	}, nil
}

func (c *R2Client) UploadImage(ctx context.Context, imageData []byte, contentType string) (string, error) {
	key := fmt.Sprintf("mindmaps/%s.png", time.Now().Format("20060102150405"))

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %v", err)
	}

	// Return public URL
	return fmt.Sprintf("%s/%s", c.domain, key), nil
}
