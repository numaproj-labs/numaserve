package downloader

import (
	"context"
	"fmt"
	"github.com/caitlinelfring/go-env-default"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/numaproj-labs/numaserve/pkg/common"
	"log"
)

// Todo POC will support minio/S3. It need to refactor to support different cloud storage
// TODO Move the config in MLInference Spec
func DownloadModel() {
	fmt.Println("Downloading model from S3")
	endpoint := env.GetDefault(common.S3_HOST, "localhost:9090")
	accessKeyID := env.GetDefault(common.S3_ASSESS_KEY_ID, "")
	secretAccessKey := env.GetDefault(common.S3_SECRET_ACCESS_KEY, "")
	bucket := env.GetDefault(common.S3_BUCKET_NAME, "mlserve")
	objectKey := env.GetDefault(common.OBJECT_KEY, "README.md")
	outputPath := env.GetDefault(common.OUTPUT_PATH, "/tmp/tmp")
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	opts := minio.ListObjectsOptions{
		UseV1:     true,
		Recursive: true,
	}
	for object := range minioClient.ListObjects(context.Background(), "mlserve", opts) {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		fmt.Println(object)
	}

	err = minioClient.FGetObject(context.Background(), bucket, objectKey, outputPath, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Model saved successfully")
}

//func main() {
//	os.Setenv(common.S3_HOST, "localhost:9000")
//	os.Setenv(common.S3_ASSESS_KEY_ID, "minioadmin")
//	os.Setenv(common.S3_SECRET_ACCESS_KEY, "minioadmin")
//	os.Setenv(common.S3_BUCKET_NAME, "mlserve")
//	DownloadModel()
//
//}
