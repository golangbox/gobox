package s3

import (
	"os"
	"time"

	"github.com/golangbox/goamz/aws"
	"github.com/golangbox/goamz/s3"
)

var client *s3.S3
var bucket *s3.Bucket

func init() {
	key := os.Getenv("GOBOX_AWS_ACCESS_KEY_ID")
	secret := os.Getenv("GOBOX_AWS_SECRET_ACCESS_KEY")
	auth := aws.Auth{AccessKey: key, SecretKey: secret}
	client = s3.New(auth, aws.Regions["us-west-2"])
	bucket = client.Bucket("gobox")
}

func TestKeyExistence(hash string) (exists bool, err error) {
	exists, err = bucket.Exists(hash)
	return exists, err
}

func GenerateSignedUrl(hash string) (url string, err error) {
	return bucket.SignedURL(hash, time.Now().Add(time.Duration(time.Minute*10)))
}

func UploadFile(hash string, fileBody []byte) error {
	var options s3.Options
	return bucket.Put(hash, fileBody, "", s3.PublicRead, options)
}
