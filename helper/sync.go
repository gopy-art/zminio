package helper

import (
	"Zminio/console"
	logger "Zminio/log"
	"Zminio/prometheus"
	sl "Zminio/sideloader"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/notification"
)

type ObjectEvent struct {
	ObjectName  string
	BucketName  string
	ContentType string
	ObjectSize  int64
}

func SyncBucketToBucket(minioHelper *Minio) {
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

	sourceClient, err := minioHelper.Functionality.MinioConnection(console.Url, console.Access_key, console.Secret_key, console.Bucket, console.SecureSSL)
	if err != nil {
		logger.ErrorLogger.Fatal(err)
	}

	dstClient, err := configSync.Functionality.MinioConnection(console.Url_sync, console.Access_key_sync, console.Secret_key_sync, console.Bucket_sync, console.SecureSSL_sync)
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
	sideloader := make(chan minio.ObjectInfo, 1)

	// WaitGroup to manage concurrency
	var wg sync.WaitGroup

	// start sideloader workers
	if console.SideLoader && console.SideLoaderType == "client" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for obj := range sideloader {
				if console.CacheUsage && os.Getenv("REDIS_BACKEND_URL") != "" {
					err := cache.InitConnection()
					if err != nil {
						logger.ErrorLogger.Fatalf("error in connect to cache, error = %v", err)
					}
			
					if _, err = cache.GetSpecificKey(fmt.Sprintf("%v/%v", obj.Key, obj.Key)); err == nil {
						logger.WarningLogger.Printf("this object { %v } already exists in { %v } bucket in the destination.", obj.Key, obj.Key)
						continue
					}
				}
			
				if console.MinioCache {
					if _, err := dstClient.StatObject(context.Background(), console.Bucket_sync, obj.Key, minio.StatObjectOptions{}); err == nil {
						logger.WarningLogger.Printf("this object { %v } already exists in { %v } bucket in the destination.", obj.Key, obj.Key)
						continue
					}
				}

				logger.InfoLogger.Printf("object with name {%v} and size {%.2f} received in sideloader worker", obj.Key, float64(obj.Size)/(1024*1024))
				if err := sl.ClientSideloader(console.Bucket+"/-/"+obj.Key, console.Bucket, console.Bucket_sync, dstClient); err != nil {
					logger.ErrorLogger.Printf("[SIDELOADER] : error in transfer this file {%v}, error = %v \n", obj.Key, err)
				}

				if console.DeleteInSync {
					err = sourceClient.RemoveObject(context.Background(), console.Bucket, obj.Key, minio.RemoveObjectOptions{ForceDelete: true})
					if err != nil {
						logger.ErrorLogger.Printf("Error deleting %s: %v\n", obj.Key, err)
						return
					}
				}
			}
		}()
	}

	// Start worker goroutines
	for i := 0; i < console.NumberOfWorker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for objectKey := range objectCh {
				if console.Prometheus != "" {
					prometheus.IncreasePrometheusCount("sync")
				}
				syncObject(sourceClient, dstClient, console.Bucket, console.Bucket_sync, objectKey.ContentType, objectKey.Key, objectKey.Size)
				if console.Prometheus != "" {
					go prometheus.DecreasePrometheusCount()
				}
			}
		}()
	}

	if console.ListenSync {
		go func() {
			for {
				if console.CacheUsage && os.Getenv("REDIS_BACKEND_URL") != "" {
					err := cache.InitConnection()
					if err != nil {
						logger.ErrorLogger.Fatalf("error in connect to cache, error = %v", err)
					}

					objects, err := configSync.Functionality.ListAllObjectsFromMinio(dstClient, console.Bucket_sync)
					if err != nil {
						logger.ErrorLogger.Fatalf("error in get list of objects, error = %v", err)
					}

					for _, obj := range objects {
						if ok, err := cache.SetKeyWithValue(fmt.Sprintf("%v/%v", console.Bucket_sync, obj), obj.Key); ok != "ok" && err != nil {
							logger.ErrorLogger.Printf("error in set value in cache, error = %v \n", err)
						}
					}
				}

				objectList := sourceClient.ListObjects(context.Background(), console.Bucket, minio.ListObjectsOptions{Recursive: true})
				for object := range objectList {
					if object.Err != nil {
						logger.ErrorLogger.Println(object.Err)
					}

					// seprate sideloader and normal workers
					if console.SideLoader && console.SideLoaderType == "client" {
						if object.Size >= console.MaxSizeSideload {
							sideloader <- object
						} else {
							objectCh <- object
						}
					} else {
						objectCh <- object
					}
				}

				logger.SuccessLogger.Printf("interval run successfully!")
				time.Sleep(time.Duration(console.Interval) * time.Hour)
			}
		}()

		listenChannel := minioHelper.Functionality.NotificationFromMinio(sourceClient, "", "", []string{"s3:ObjectCreated:Put", "s3:ObjectCreated:CompleteMultipartUpload"})

		var wg2 sync.WaitGroup
		for range console.NumberOfWorker {
			wg2.Add(1)
			go func() {
				for notification := range listenChannel {
					for _, event := range notification.Records {
						if event.S3.Bucket.Name == console.Bucket {
							logger.InfoLogger.Printf("recieve object with name { %s }", event.S3.Object.Key)
							obj := minio.ObjectInfo{
								Key:         event.S3.Object.Key,
								Size:        event.S3.Object.Size,
								ContentType: event.S3.Object.ContentType,
							}

							// seprate sideloader and normal workers
							if console.SideLoader && console.SideLoaderType == "client" {
								if obj.Size >= console.MaxSizeSideload {
									sideloader <- obj
								} else {
									objectCh <- obj
								}
							} else {
								objectCh <- obj
							}
						}
					}
				}
			}()
		}
		wg2.Wait()
	} else {
		if console.CacheUsage && os.Getenv("REDIS_BACKEND_URL") != "" {
			err := cache.InitConnection()
			if err != nil {
				logger.ErrorLogger.Fatalf("error in connect to cache, error = %v", err)
			}

			objects, err := configSync.Functionality.ListAllObjectsFromMinio(dstClient, console.Bucket_sync)
			if err != nil {
				logger.ErrorLogger.Fatalf("error in get list of objects, error = %v", err)
			}

			for _, obj := range objects {
				if ok, err := cache.SetKeyWithValue(fmt.Sprintf("%v/%v", console.Bucket_sync, obj), obj.Key); ok != "ok" && err != nil {
					logger.ErrorLogger.Printf("error in set value in cache, error = %v \n", err)
				}
			}
		}

		// List objects in the source bucket and send to channel
		objectList := sourceClient.ListObjects(context.Background(), console.Bucket, minio.ListObjectsOptions{Recursive: true})
		for object := range objectList {
			if object.Err != nil {
				logger.ErrorLogger.Println(object.Err)
			}

			// seprate sideloader and normal workers
			if console.SideLoader && console.SideLoaderType == "client" {
				if object.Size >= console.MaxSizeSideload {
					sideloader <- object
				} else {
					objectCh <- object
				}
			} else {
				objectCh <- object
			}
		}
		close(objectCh) // Close the channel when done
		if console.SideLoader && console.SideLoaderType == "client" {
			close(sideloader)
		}
		// Wait for all workers to finish
		wg.Wait()
	}

	elapsed := time.Since(start)
	logger.SuccessLogger.Println("sync operation took = ", elapsed)
}

func SyncAllBucketToAllBucket(minioHelper *Minio) {
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

	sourceClient, err := minioHelper.Functionality.MinioConnection(console.Url, console.Access_key, console.Secret_key, console.Bucket, console.SecureSSL)
	if err != nil {
		logger.ErrorLogger.Fatal(err)
	}

	dstClient, err := configSync.Functionality.MinioConnection(console.Url_sync, console.Access_key_sync, console.Secret_key_sync, console.Bucket_sync, console.SecureSSL_sync)
	if err != nil {
		logger.ErrorLogger.Fatal(err)
	}

	listOfbuckets, err := sourceClient.ListBuckets(context.Background())
	if err != nil {
		logger.ErrorLogger.Fatalf("error in get list of bucket, err = %v", err)
	}

	for _, bucket := range listOfbuckets {
		// Ensure the destination bucket exists
		err = dstClient.MakeBucket(context.Background(), bucket.Name, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			// Check if bucket already exists
			exists, errBucketExists := dstClient.BucketExists(context.Background(), bucket.Name)
			if errBucketExists != nil && !exists {
				logger.ErrorLogger.Println(err)
			}
		}
	}

	logger.SuccessLogger.Println("All buckets created in the destination.")

	start := time.Now()
	// Channel for object keys
	objectCh := make(chan ObjectEvent)
	objectChEvent := make(chan notification.Event)
	sideloader := make(chan ObjectEvent, 1)

	// WaitGroup to manage concurrency
	var wg sync.WaitGroup

	// start sideloader workers
	if console.SideLoader && console.SideLoaderType == "client" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for obj := range sideloader {
				if console.CacheUsage && os.Getenv("REDIS_BACKEND_URL") != "" {
					err := cache.InitConnection()
					if err != nil {
						logger.ErrorLogger.Fatalf("error in connect to cache, error = %v", err)
					}
			
					if _, err = cache.GetSpecificKey(fmt.Sprintf("%v/%v", obj.BucketName, obj.ObjectName)); err == nil {
						logger.WarningLogger.Printf("this object { %v } already exists in { %v } bucket in the destination.", obj.ObjectName, obj.BucketName)
						continue
					}
				}
			
				if console.MinioCache {
					if _, err := dstClient.StatObject(context.Background(), obj.BucketName, obj.ObjectName, minio.StatObjectOptions{}); err == nil {
						logger.WarningLogger.Printf("this object { %v } already exists in { %v } bucket in the destination.", obj.ObjectName, obj.BucketName)
						continue
					}
				}

				logger.InfoLogger.Printf("object with name {%v} and size {%.2f} received in sideloader worker", obj.ObjectName, float64(obj.ObjectSize)/(1024*1024))
				if err := sl.ClientSideloader(obj.BucketName+"/-/"+obj.ObjectName, obj.BucketName, obj.BucketName, dstClient); err != nil {
					logger.ErrorLogger.Printf("[SIDELOADER] : error in transfer this file {%v}, error = %v \n", obj.ObjectName, err)
				}

				if console.DeleteInSync {
					err = sourceClient.RemoveObject(context.Background(), obj.BucketName, obj.ObjectName, minio.RemoveObjectOptions{ForceDelete: true})
					if err != nil {
						logger.ErrorLogger.Printf("Error deleting %s: %v\n", obj.ObjectName, err)
						return
					}
				}
			}
		}()
	}

	// Start worker goroutines
	for i := 0; i < console.NumberOfWorker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for objectKey := range objectChEvent {
				if console.Prometheus != "" {
					prometheus.IncreasePrometheusCount("sync")
				}
				syncObject(sourceClient, dstClient, objectKey.S3.Bucket.Name, objectKey.S3.Bucket.Name, objectKey.S3.Object.ContentType, objectKey.S3.Object.Key, objectKey.S3.Object.Size)
				if console.Prometheus != "" {
					go prometheus.DecreasePrometheusCount()
				}
			}
		}()
	}

	// Start worker goroutines
	for i := 0; i < console.NumberOfWorker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for objectKey := range objectCh {
				if console.Prometheus != "" {
					prometheus.IncreasePrometheusCount("sync")
				}
				syncObject(sourceClient, dstClient, objectKey.BucketName, objectKey.BucketName, objectKey.ContentType, objectKey.ObjectName, objectKey.ObjectSize)
				if console.Prometheus != "" {
					go prometheus.DecreasePrometheusCount()
				}
			}
		}()
	}

	if console.ListenSync {
		go func() {
			for {
				if console.CacheUsage && os.Getenv("REDIS_BACKEND_URL") != "" {
					err := cache.InitConnection()
					if err != nil {
						logger.ErrorLogger.Fatalf("error in connect to cache, error = %v", err)
					}

					for _, bucket := range listOfbuckets {
						objects, err := configSync.Functionality.ListAllObjectsFromMinio(dstClient, bucket.Name)
						if err != nil {
							logger.ErrorLogger.Fatalf("error in get list of objects, error = %v", err)
						}

						for _, obj := range objects {
							if ok, err := cache.SetKeyWithValue(fmt.Sprintf("%v/%v", bucket.Name, obj), obj.Key); ok != "ok" && err != nil {
								logger.ErrorLogger.Printf("error in set value in cache, error = %v \n", err)
							}
						}
					}
				}

				for _, bucket := range listOfbuckets {
					objectList := sourceClient.ListObjects(context.Background(), bucket.Name, minio.ListObjectsOptions{Recursive: true})
					for object := range objectList {
						if object.Err != nil {
							logger.ErrorLogger.Println(object.Err)
						}
						obj := ObjectEvent{
							ObjectName:  object.Key,
							ObjectSize:  object.Size,
							BucketName:  bucket.Name,
							ContentType: object.ContentType,
						}

						// seprate sideloader and normal workers
						if console.SideLoader && console.SideLoaderType == "client" {
							if obj.ObjectSize >= console.MaxSizeSideload {
								sideloader <- obj
							} else {
								objectCh <- obj
							}
						} else {
							objectCh <- obj
						}
					}
					time.Sleep(1 * time.Second)
				}

				logger.SuccessLogger.Printf("interval run successfully!")
				time.Sleep(time.Duration(console.Interval) * time.Hour)
			}
		}()

		listenChannel := minioHelper.Functionality.NotificationFromMinio(sourceClient, "", "", []string{"s3:ObjectCreated:Put", "s3:ObjectCreated:CompleteMultipartUpload"})

		var wg2 sync.WaitGroup
		for range console.NumberOfWorker {
			wg2.Add(1)
			go func() {
				for notification := range listenChannel {
					for _, event := range notification.Records {
						logger.InfoLogger.Printf("recieve object with name { %s } from this bucket { %s }", event.S3.Object.Key, event.S3.Bucket.Name)
						// seprate sideloader and normal workers
						if console.SideLoader && console.SideLoaderType == "client" {
							if event.S3.Object.Size >= console.MaxSizeSideload {
								obj := ObjectEvent{
									ObjectName:  event.S3.Object.Key,
									ObjectSize:  event.S3.Object.Size,
									BucketName:  event.S3.Bucket.Name,
									ContentType: event.S3.Object.ContentType,
								}
								sideloader <- obj
							} else {
								objectChEvent <- event
							}
						} else {
							objectChEvent <- event
						}
					}
				}
			}()
		}
		wg2.Wait()
	} else {
		if console.CacheUsage && os.Getenv("REDIS_BACKEND_URL") != "" {
			err := cache.InitConnection()
			if err != nil {
				logger.ErrorLogger.Fatalf("error in connect to cache, error = %v", err)
			}

			for _, bucket := range listOfbuckets {
				objects, err := configSync.Functionality.ListAllObjectsFromMinio(dstClient, bucket.Name)
				if err != nil {
					logger.ErrorLogger.Fatalf("error in get list of objects, error = %v", err)
				}

				for _, obj := range objects {
					if ok, err := cache.SetKeyWithValue(fmt.Sprintf("%v/%v", bucket.Name, obj), obj.Key); ok != "ok" && err != nil {
						logger.ErrorLogger.Printf("error in set value in cache, error = %v \n", err)
					}
				}
			}
		}

		for _, bucket := range listOfbuckets {
			objectList := sourceClient.ListObjects(context.Background(), bucket.Name, minio.ListObjectsOptions{Recursive: true})
			for object := range objectList {
				if object.Err != nil {
					logger.ErrorLogger.Println(object.Err)
				}
				obj := ObjectEvent{
					ObjectName:  object.Key,
					ObjectSize:  object.Size,
					BucketName:  bucket.Name,
					ContentType: object.ContentType,
				}
				// seprate sideloader and normal workers
				if console.SideLoader && console.SideLoaderType == "client" {
					if obj.ObjectSize >= console.MaxSizeSideload {
						sideloader <- obj
					} else {
						objectCh <- obj
					}
				} else {
					objectCh <- obj
				}
			}
			time.Sleep(1 * time.Second)
		}
		close(objectCh)
		if console.SideLoader && console.SideLoaderType == "client" {
			close(sideloader)
		}
		// Wait for all workers to finish
		wg.Wait()
	}

	elapsed := time.Since(start)
	logger.SuccessLogger.Println("sync operation took = ", elapsed)
}
