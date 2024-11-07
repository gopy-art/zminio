package helper

import (
	"Zminio/console"
	logger "Zminio/log"
	"Zminio/prometheus"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
)

func ActionHelper() {
	config := Minio{}
	minioHelper := config.InitConnection()

	switch console.Type {
	case "upload":
		if console.Pathfile == "" {
			logger.ErrorLogger.Println("path of the file is not set!")
			os.Exit(0)
		}

		err := minioHelper.Upload(console.Pathfile)
		if err != nil {
			logger.ErrorLogger.Println(err)
			os.Exit(0)
		}
		if console.Prometheus != "" {
			prometheus.IncreasePrometheusCount("upload")
		}
		logger.SuccessLogger.Printf("file with path [%v] successfully uploaded in [%v] bucket!", console.Pathfile, console.Bucket)
	case "uploadDir":
		if console.Pathfile == "" {
			logger.ErrorLogger.Println("path of the file is not set!")
			os.Exit(0)
		}

		start := time.Now()
		var wg sync.WaitGroup

		// Read the directory
		entries, err := os.ReadDir(console.Pathfile)
		if err != nil {
			logger.ErrorLogger.Fatal(err)
		}

		// Iterate through the entries and print their names
		for i, entry := range entries {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cmd := exec.Command("sh", "-c",
					fmt.Sprintf("./Zminio -ak %v -sk %v -u %v -b %v -do upload -f %v",
						os.Getenv("MINIO_ACCESS_KEY"),
						os.Getenv("MINIO_SECRET_KEY"),
						os.Getenv("MINIO_ENDPOINT"),
						os.Getenv("MINIO_BUCKET_NAME"),
						console.Pathfile+entry.Name()))
				// get the output from command
				_, err := cmd.CombinedOutput()
				if err != nil {
					logger.ErrorLogger.Println(err)
				}

				logger.SuccessLogger.Printf("file with path [%v] successfully uploaded in [%v] bucket!", console.Pathfile+entry.Name(), console.Bucket)
				if console.Prometheus != "" {
					prometheus.IncreasePrometheusCount("upload")
				}
			}()

			if len(entries) < 50 {
				wg.Wait()
			} else {
				if i%console.NumberOfWorker == 0 {
					wg.Wait()
					time.Sleep(1 * time.Second)
				}
			}
		}

		end := time.Since(start)
		logger.SuccessLogger.Printf("The count of file has been uploaded to the minio is = %v\n", len(entries))
		logger.SuccessLogger.Printf("This operation took = %v\n", end)
	case "download":
		if console.OutPutFile == "" {
			logger.ErrorLogger.Println("output path is not set!")
			os.Exit(0)
		}

		// Check if directory exists
		_, err := os.Stat(console.OutPutFile)
		if os.IsNotExist(err) {
			err := os.MkdirAll(console.OutPutFile, 0755)
			if err != nil {
				logger.ErrorLogger.Println(err)
			}
		}

		if console.Object == "" {
			logger.ErrorLogger.Println("object name is not set!")
			os.Exit(0)
		}

		var wg sync.WaitGroup
		var count int64 = 0

		if console.Object == "all" {
			list, err := minioHelper.ListObjects(console.Bucket)
			if err != nil {
				logger.ErrorLogger.Println(err)
				os.Exit(0)
			}

			for i, l := range list {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := minioHelper.Download(console.OutPutFile+"/"+l, l)
					if err != nil {
						logger.ErrorLogger.Println(err)
						os.Exit(0)
					}
					atomic.AddInt64(&count, 1)
					if console.Prometheus != "" {
						prometheus.IncreasePrometheusCount("download")
					}
					logger.SuccessLogger.Printf("The object with name {%v} downloaded to the this path {%v} successfully.", console.Object, console.OutPutFile)
				}()

				if len(list) < 50 {
					wg.Wait()
				} else {
					if i%50 == 0 {
						wg.Wait()
					}
				}
			}
			logger.SuccessLogger.Printf("The count of file has been dwonloaded from the minio is = %v\n", count)
		} else {
			err := minioHelper.Download(console.OutPutFile, console.Object)
			if err != nil {
				logger.ErrorLogger.Println(err)
				os.Exit(0)
			}
			if console.Prometheus != "" {
				prometheus.IncreasePrometheusCount("download")
			}
			logger.SuccessLogger.Printf("The object with name {%v} downloaded to the this path {%v} successfully.", console.Object, console.OutPutFile)
		}
	case "delete":
		if console.Object == "" {
			logger.ErrorLogger.Println("object name is not set!")
			os.Exit(0)
		}

		if console.Object == "all" {
			var wg sync.WaitGroup
			var count int64 = 0

			list, err := minioHelper.ListObjects(console.Bucket)
			if err != nil {
				logger.ErrorLogger.Println(err)
				os.Exit(0)
			}

			for i, l := range list {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := minioHelper.Delete(l)
					if err != nil {
						logger.ErrorLogger.Println(err)
						os.Exit(0)
					}
					atomic.AddInt64(&count, 1)
					logger.SuccessLogger.Printf("The object with name {%v} deleted successfully.", l)
				}()

				if len(list) < 50 {
					wg.Wait()
				} else {
					if i%50 == 0 {
						wg.Wait()
					}
				}
			}
			logger.SuccessLogger.Printf("The count of file has been deleted from the minio is = %v\n", count)
		} else {
			err := minioHelper.Delete(console.Object)
			if err != nil {
				logger.ErrorLogger.Println(err)
				os.Exit(0)
			}
			logger.SuccessLogger.Printf("The object with name {%v} deleted successfully.", console.Object)
		}
	case "list":
		list, err := minioHelper.ListObjects(console.Bucket)
		if err != nil {
			logger.ErrorLogger.Println(err)
			os.Exit(0)
		}

		for _, l := range list {
			fmt.Println(l)
		}

		logger.SuccessLogger.Printf("Getting list from minio bucket was successfully.")
	case "info":
		if console.Object == "" {
			logger.ErrorLogger.Fatalf("object name can not be empty")
		}

		info, err := minioHelper.ObjectInfo(console.Bucket, console.Object)
		if err != nil {
			logger.ErrorLogger.Println(err)
			os.Exit(0)
		}

		logger.InfoLogger.Printf("Info about that object is = %+v \n", info)

		logger.SuccessLogger.Printf("Getting info from minio was successfully.")
	case "move":
		if console.MBucket == "" {
			logger.ErrorLogger.Println("move bucket is not set!")
			os.Exit(0)
		}

		if console.Object == "" {
			logger.ErrorLogger.Println("object name is not set!")
			os.Exit(0)
		}

		err := minioHelper.MoveObject(console.Object, console.Bucket, console.MBucket)
		if err != nil {
			logger.ErrorLogger.Println(err)
			os.Exit(0)
		}
		logger.SuccessLogger.Printf("The object with name {%v} moved from this bucket {%v} to the this bucket {%v} successfully.", console.Object, console.Bucket, console.MBucket)
	case "sync":
		if console.Bucket_sync == "" || console.Access_key_sync == "" || console.Secret_key_sync == "" || console.Url_sync == "" {
			logger.ErrorLogger.Println("nessesery information is not set!")
			os.Exit(0)
		}

		if console.NumberOfWorker == 0 {
			logger.ErrorLogger.Fatalln("number of worker is invalid!")
		}

		functions := MinioMethods{}
		configSync := Minio{
			Bucket:         console.Bucket_sync,
			MinioAddress:   console.Url_sync,
			MinioSecretKey: console.Secret_key_sync,
			MinioAccessKey: console.Access_key_sync,
			MinioSecure:    console.SecureSSL_sync,
			Functionality:  functions,
		}

		sourceClient, err := minioHelper.Functionality.MinioConnection(console.Url, console.Secret_key, console.Access_key, console.Bucket, console.SecureSSL)
		if err != nil {
			logger.ErrorLogger.Fatal(err)
		}

		dstClient, err := configSync.Functionality.MinioConnection(console.Url_sync, console.Secret_key_sync, console.Access_key_sync, console.Bucket_sync, console.SecureSSL_sync)
		if err != nil {
			logger.ErrorLogger.Fatal(err)
		}

		// Ensure the destination bucket exists
		err = dstClient.MakeBucket(context.Background(), console.Bucket_sync, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			// Check if bucket already exists
			exists, errBucketExists := dstClient.BucketExists(context.Background(), console.Bucket_sync)
			if errBucketExists != nil && !exists {
				logger.ErrorLogger.Println(err)
			}
		}

		start := time.Now()
		// Channel for object keys
		objectCh := make(chan minio.ObjectInfo)

		// WaitGroup to manage concurrency
		var wg sync.WaitGroup

		// Start worker goroutines
		for i := 0; i < console.NumberOfWorker; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for objectKey := range objectCh {
					syncObject(sourceClient, dstClient, console.Bucket, console.Bucket_sync, objectKey.ContentType, objectKey.Key, objectKey.Size)
				}
			}()
		}

		// List objects in the source bucket and send to channel
		objectList := sourceClient.ListObjects(context.Background(), console.Bucket, minio.ListObjectsOptions{Recursive: true})
		for object := range objectList {
			if object.Err != nil {
				logger.ErrorLogger.Println(object.Err)
			}
			objectCh <- object
		}
		close(objectCh) // Close the channel when done

		// Wait for all workers to finish
		wg.Wait()

		elapsed := time.Since(start)
		logger.SuccessLogger.Println("sync operation took = ", elapsed)
	case "listenDownload":
		if console.OutPutFile == "" {
			logger.ErrorLogger.Fatalln("the output path does not set!")
		}

		// Check if directory exists
		_, err := os.Stat(console.OutPutFile)
		if os.IsNotExist(err) {
			err := os.MkdirAll(console.OutPutFile, 0755)
			if err != nil {
				logger.ErrorLogger.Println(err)
			}
		}

		connection, err := minioHelper.Functionality.MinioConnection(console.Url, console.Secret_key, console.Access_key, console.Bucket, console.SecureSSL)
		if err != nil {
			logger.ErrorLogger.Fatalln("error in connect to the minio")
		}

		listenChannel := minioHelper.Functionality.NotificationFromMinio(connection, "", "", []string{"s3:ObjectCreated:Put", "s3:ObjectCreated:CompleteMultipartUpload"})

		var wg sync.WaitGroup
		for range console.NumberOfWorker {
			wg.Add(1)
			go func() {
				for notification := range listenChannel {
					for _, event := range notification.Records {
						if event.S3.Bucket.Name == console.Bucket {
							logger.InfoLogger.Printf("recieve object with name { %s }", event.S3.Object.Key)
							err := minioHelper.Functionality.DownloadFromMinio(connection, console.OutPutFile, event.S3.Object.Key, console.Bucket)
							if err != nil {
								logger.ErrorLogger.Printf("error in download this object {%s}, error = %v", event.S3.Object.Key, err)
							} else {
								logger.SuccessLogger.Printf("object {%s} downloaded successfully!", event.S3.Object.Key)

								if console.Prometheus != "" {
									prometheus.IncreasePrometheusCount("download")
								}

								err := minioHelper.Functionality.DeleteFromMinio(connection, event.S3.Object.Key, console.Bucket)
								if err != nil {
									logger.ErrorLogger.Printf("error in delete object {%s}, error = %v", event.S3.Object.Key, err)
								} else {
									logger.SuccessLogger.Printf("object {%s} deleted successfully!", event.S3.Object.Key)
								}
							}
						}
					}
				}
			}()
		}
		wg.Wait()
	default:
		logger.ErrorLogger.Println("type in invalid!")
		return
	}
}

// syncObject downloads an object from the source and uploads it to the destination
func syncObject(sourceClient, dstClient *minio.Client, sourceBucket, dstBucket, contentType, objectKey string, objectSize int64) {
	// Download the object from the source bucket
	reader, err := sourceClient.GetObject(context.Background(), sourceBucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		logger.ErrorLogger.Printf("Error downloading %s: %v\n", objectKey, err)
		return
	}
	defer reader.Close()

	// Upload the object to the destination bucket
	_, err = dstClient.PutObject(context.Background(), dstBucket, objectKey, reader, objectSize, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		logger.ErrorLogger.Printf("Error uploading %s: %v\n", objectKey, err)
		return
	}

	if console.DeleteInSync {
		err = sourceClient.RemoveObject(context.Background(), sourceBucket, objectKey, minio.RemoveObjectOptions{ForceDelete: true})
		if err != nil {
			logger.ErrorLogger.Printf("Error deleting %s: %v\n", objectKey, err)
			return
		}
	}

	logger.SuccessLogger.Printf("The object with name {%v} moved from this bucket {%v} to the this bucket {%v} successfully.", objectKey, console.Bucket, console.Bucket_sync)
}
