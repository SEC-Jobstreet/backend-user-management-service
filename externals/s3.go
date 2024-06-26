package externals

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/SEC-Jobstreet/backend-candidate-service/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	handlerName = "s3"
	// Presign GET URLs for this number of seconds.
	presignDuration = 120
)

type awsconfig struct {
	AccessKeyId     string   `json:"access_key_id"`
	SecretAccessKey string   `json:"secret_access_key"`
	Region          string   `json:"region"`
	DisableSSL      bool     `json:"disable_ssl"`
	ForcePathStyle  bool     `json:"force_path_style"`
	Endpoint        string   `json:"endpoint"`
	BucketName      string   `json:"bucket"`
	BucketSubFolder string   `json:"-"`
	CorsOrigins     []string `json:"cors_origins"`
}

type AWSHandler struct {
	svc  *s3.S3
	conf awsconfig
}

func NewAWSHandler() *AWSHandler {
	return &AWSHandler{}
}

// readerCounter is a byte counter for bytes read through the io.Reader
type readerCounter struct {
	io.Reader
	count  int64
	reader io.Reader
}

// Read reads the bytes and records the number of read bytes.
func (rc *readerCounter) Read(buf []byte) (int, error) {
	n, err := rc.reader.Read(buf)
	atomic.AddInt64(&rc.count, int64(n))
	return n, err
}

// Init initializes the media handler.
func (ah *AWSHandler) Init(config utils.Config) error {
	var err error

	ah.conf = awsconfig{
		AccessKeyId:     config.S3AccessKeyId,
		SecretAccessKey: config.S3SecretAccessKey,
		Region:          config.S3Region,
		DisableSSL:      config.S3DisableSSL,
		ForcePathStyle:  config.S3ForcePathStyle,
		Endpoint:        config.S3EndPoint,
		BucketName:      config.S3BucketName,
		BucketSubFolder: config.S3BucketSubFolder,
	}

	if ah.conf.AccessKeyId == "" {
		return errors.New("missing Access Key ID")
	}
	if ah.conf.SecretAccessKey == "" {
		return errors.New("missing Secret Access Key")
	}
	if ah.conf.Region == "" {
		return errors.New("missing Region")
	}
	if ah.conf.BucketName == "" {
		return errors.New("missing Bucket")
	}

	var sess *session.Session
	if sess, err = session.NewSession(&aws.Config{
		Region:           aws.String(ah.conf.Region),
		DisableSSL:       aws.Bool(ah.conf.DisableSSL),
		S3ForcePathStyle: aws.Bool(ah.conf.ForcePathStyle),
		Endpoint:         aws.String(ah.conf.Endpoint),
		Credentials:      credentials.NewStaticCredentials(ah.conf.AccessKeyId, ah.conf.SecretAccessKey, ""),
	}); err != nil {
		return err
	}

	// Create S3 service client
	ah.svc = s3.New(sess)

	// Check if bucket already exists.
	//_, err = ah.svc.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(ah.conf.BucketName)})
	_, err = ah.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(ah.conf.BucketName),
		Prefix:  aws.String(ah.conf.BucketSubFolder),
		MaxKeys: aws.Int64(1),
	})
	if err == nil {
		// Bucket exists
		return nil
	}

	if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NotFound" {
		// Hard error.
		return err
	}

	// Bucket does not exist. Create one.
	_, err = ah.svc.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(ah.conf.BucketName)})
	if err != nil {
		// Check if someone has already created a bucket (possible in a cluster).
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeBucketAlreadyExists ||
				aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou ||
				// Someone is already creating this bucket:
				// OperationAborted: A conflicting conditional operation is currently in progress against this resource.
				aerr.Code() == "OperationAborted" {
				// Clear benign error
				err = nil
			}
		}
	} else {
		// This is a new bucket.

		// The following serves two purposes:
		// 1. Setup CORS policy to be able to serve media directly from S3.
		// 2. Verify that the bucket is accessible to the current user.
		origins := ah.conf.CorsOrigins
		if len(origins) == 0 {
			origins = append(origins, "*")
		}
		_, err = ah.svc.PutBucketCors(&s3.PutBucketCorsInput{
			Bucket: aws.String(ah.conf.BucketName),
			CORSConfiguration: &s3.CORSConfiguration{
				CORSRules: []*s3.CORSRule{{
					AllowedMethods: aws.StringSlice([]string{http.MethodGet, http.MethodHead}),
					AllowedOrigins: aws.StringSlice(origins),
					AllowedHeaders: aws.StringSlice([]string{"*"}),
				}},
			},
		})
	}
	return err
}

// Upload processes request for a file upload. The file is given as io.Reader.
func (ah *AWSHandler) Upload(filename string, file io.ReadSeeker) (string, int64, error) {
	var err error

	uploader := s3manager.NewUploaderWithClient(ah.svc)

	rc := readerCounter{reader: file}
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(ah.conf.BucketName),
		Key:    aws.String(fmt.Sprintf("%s/%s", ah.conf.BucketSubFolder, filename)),
		Body:   &rc,
	})

	if err != nil {
		return "", 0, err
	}

	return result.Location, rc.count, nil
}

// Delete deletes files from aws by provided slice of locations.
func (ah *AWSHandler) Delete(locations []string) error {
	toDelete := make([]s3manager.BatchDeleteObject, len(locations))
	for i, key := range locations {
		toDelete[i] = s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Key:    aws.String(key),
				Bucket: aws.String(ah.conf.BucketName),
			}}
	}
	batcher := s3manager.NewBatchDeleteWithClient(ah.svc)
	return batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
		Objects: toDelete,
	})
}

//func init() {
//	media.RegisterMediaHandler(handlerName, &awshandler{})
//}
