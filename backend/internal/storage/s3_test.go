package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestS3StorePutGetDeleteLifecycle(t *testing.T) {
	cfg := loadTestS3Config(t)
	if cfg == nil {
		return
	}

	store, err := NewS3Store(*cfg)
	if err != nil {
		t.Fatalf("new s3 store: %v", err)
	}
	ctx := context.Background()
	if err := store.VerifyBucket(ctx); err != nil {
		t.Fatalf("verify bucket: %v", err)
	}

	key := fmt.Sprintf("homevox-test/%d.png", unixNanoUnique())
	payload := []byte("hello homevox image")
	if err := store.PutObject(ctx, key, "image/png", payload); err != nil {
		t.Fatalf("put object: %v", err)
	}

	obj, err := store.GetObject(ctx, key)
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if obj.Size != int64(len(payload)) {
		t.Fatalf("size = %d, want %d", obj.Size, len(payload))
	}
	if obj.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", obj.ContentType)
	}
	if string(obj.Data) != string(payload) {
		t.Fatalf("data = %q, want %q", obj.Data, payload)
	}

	if err := store.DeleteObject(ctx, key); err != nil {
		t.Fatalf("delete object: %v", err)
	}

	if _, err := store.GetObject(ctx, key); err == nil {
		t.Fatal("expected object missing after delete")
	}
}

func loadTestS3Config(t *testing.T) *Config {
	t.Helper()

	endpoint := os.Getenv("HOMEVOX_TEST_S3_ENDPOINT")
	bucket := os.Getenv("HOMEVOX_TEST_S3_BUCKET")
	access := os.Getenv("HOMEVOX_TEST_S3_ACCESS_KEY")
	secret := os.Getenv("HOMEVOX_TEST_S3_SECRET_KEY")
	if endpoint == "" || bucket == "" || access == "" || secret == "" {
		t.Skip("HOMEVOX_TEST_S3_* not all set; skipping integration test")
		return nil
	}

	return &Config{
		Endpoint:  endpoint,
		Bucket:    bucket,
		AccessKey: access,
		SecretKey: secret,
	}
}

func unixNanoUnique() int64 {
	return time.Now().UnixNano()
}
