package helper

import (
	logger "Zminio/log"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/notification"
)

type MinioMethods struct{}

/*
uploader into the minio
get the file path, minio connection and bucket name and returns err
*/
func (m MinioMethods) UploadInMinio(connection *minio.Client, pathfile, bucket string) error {
	// Open the file
	file, err := os.Open(pathfile)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()

	// Upload the file to the MinIO bucket
	_, err = connection.PutObject(context.Background(), bucket, path.Base(pathfile), file, -1, minio.PutObjectOptions{})
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

/*
downloader from the minio
get the path, minio connection and bucket name and returns err
it will download the file save it into the path that you provided
*/
func (m MinioMethods) DownloadFromMinio(connection *minio.Client, pathfile, objname, bucket string) error {
	object, err := connection.GetObject(context.Background(), bucket, objname, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer object.Close()

	// Create a local file to save the downloaded object
	localFile, err := os.Create(pathfile + objname)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Copy the object content to the local file
	if _, err = io.Copy(localFile, object); err != nil {
		return err
	}

	return nil
}

/*
delete from the minio
get the object name, minio connection and bucket name and returns err
it will delete the object that you provided
*/
func (m MinioMethods) DeleteFromMinio(connection *minio.Client, objname, bucket string) error {
	// Remove the object
	err := connection.RemoveObject(context.Background(), bucket, objname, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

/*
This function is for set the connection with minio server
and it will return minio.Client for do some stuff!
*/
func (m MinioMethods) MinioConnection(url, username, password, bucket string, secure bool) (*minio.Client, error) {
	// Create a new MinIO client
	var client *minio.Client
	var minioOption *minio.Options
	var err error
	var errorLog bool = true

	if secure {
		// Create a custom transport with InsecureSkipVerify set to true
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip SSL verification
			},
		}

		minioOption = &minio.Options{
			Creds:     credentials.NewStaticV4(username, password, ""),
			Secure:    secure,
			Transport: transport,
		}
	} else {
		minioOption = &minio.Options{
			Creds:     credentials.NewStaticV4(username, password, ""),
			Secure:    secure,
		}
	}

	// make loop for connecting to the minio if we have connection lost
	for {
		client, err = minio.New(url, minioOption)
		if err != nil {
			if errorLog {
				logger.ErrorLogger.Println("we have an error in connecting to minio, err = ", err)
				errorLog = !errorLog
			}
			time.Sleep(1 * time.Second)
			continue
		} else {
			break
		}
	}

	// Create a new bucket
	bucketName := bucket
	location := "us-east-1"
	err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check if bucket already exists
		exists, errBucketExists := client.BucketExists(context.Background(), bucketName)
		if errBucketExists != nil && !exists {
			return nil, err
		}
	}

	return client, nil
}

/*
check if bucket exists or not, if it is not create it
get the minio connection and bucket name and returns error
*/
func (m MinioMethods) CheckBucketExists(connection *minio.Client, bucket string) error {
	location := "us-east-1"
	err := connection.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check if bucket already exists
		exists, errBucketExists := connection.BucketExists(context.Background(), bucket)
		if errBucketExists != nil && !exists {
			return err
		}
	}

	return nil
}

/*
get all objects from the minio
get the minio connection and bucket name and returns err and list of objects key
it will return list of object in the bucket that you provided
*/
func (m MinioMethods) ListAllObjectsFromMinio(connection *minio.Client, bucket string) ([]string, error) {
	// save key array
	var objects []string

	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// List objects in the bucket
	objectCh := connection.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

/*
move one object from a bucket to another bucket in the minio
get the minio connection, source bucket name, destination bucket name and the object name and returns err
it will return list of object in the bucket that you provided
*/
func (m MinioMethods) MoveObjectInMinio(connection *minio.Client, bucketSrc, bucketDest, objname string) error {
	// Copy object to the destination bucket
	srcOpts := minio.CopySrcOptions{
		Bucket: bucketSrc,
		Object: objname,
	}
	dstOpts := minio.CopyDestOptions{
		Bucket: bucketDest,
		Object: objname,
	}

	// Perform the copy operation
	_, err := connection.CopyObject(context.Background(), dstOpts, srcOpts)
	if err != nil {
		return err
	}

	// Remove the object from the source bucket
	err = connection.RemoveObject(context.Background(), bucketSrc, objname, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

/*
get notification from the minio
get the minio connection and prefix and suffix and event list the returns chan notification.Info
*/
func (m MinioMethods) NotificationFromMinio(connection *minio.Client, prefix, suffix string, events []string) <-chan notification.Info {
	// Listen for notifications
	return connection.ListenNotification(context.Background(), prefix, suffix, events)
}

/*
get the object info from the minio
get the minio connection and bucket name and object name and return an interface and error
*/
func (m MinioMethods) GetObjectInfo(connection *minio.Client, bucket string, objname string) (interface{}, error) {
	info, err := connection.StatObject(context.Background(), bucket, objname, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	} else {
		return info, nil
	}
}
