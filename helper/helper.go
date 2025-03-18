package helper

import (
	"Zminio/console"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/notification"
)

type Minio struct {
	Bucket         string
	MinioAddress   string
	MinioSecretKey string
	MinioAccessKey string
	MinioSecure    bool
	Functionality  MinioFunctions
}

type MinioFunctions interface {
	UploadInMinio(connection *minio.Client, pathfile, bucket string) error
	MinioConnection(url, username, password, bucket string, secure bool) (*minio.Client, error)
	DownloadFromMinio(connection *minio.Client, pathfile, objname, bucket string) error
	DeleteFromMinio(connection *minio.Client, objname, bucket string) error
	ListAllObjectsFromMinio(connection *minio.Client, bucket string) ([]ObjectInfo, error)
	MoveObjectInMinio(connection *minio.Client, bucketSrc, bucketDest, objname string) error
	CheckBucketExists(connection *minio.Client, bucket string) error
	NotificationFromMinio(connection *minio.Client, prefix, suffix string, events []string) <-chan notification.Info
	GetObjectInfo(connection *minio.Client, bucket string, objname string) (interface{}, error)
}

/*
This function is for init the Minio struct and set the url, secret and access key to it.
*/
func (m *Minio) InitConnection() *Minio {
	functions := MinioMethods{}

	return &Minio{
		Bucket:         console.Bucket,
		MinioAddress:   console.Url,
		MinioSecretKey: console.Secret_key,
		MinioAccessKey: console.Access_key,
		MinioSecure:    console.SecureSSL,
		Functionality:  functions,
	}
}

/*
This function is for upload the objects into the minio, you should give the path of objects that you want to upload, in argument
*/
func (m *Minio) Upload(pathfile string) error {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return err
	} else {
		err := m.Functionality.UploadInMinio(clm, pathfile, m.Bucket)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

/*
This function is for Download the objects from the minio, you should give the save path of objects that you want to download, in argument
*/
func (m *Minio) Download(path string, objname string) error {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return err
	} else {
		err := m.Functionality.DownloadFromMinio(clm, path, objname, m.Bucket)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

/*
This function is for Delete the objects from the minio, you should give the object name that you want to delete , in argument.
*/
func (m *Minio) Delete(objname string) error {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return err
	} else {
		err := m.Functionality.DeleteFromMinio(clm, objname, m.Bucket)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

/*
This function is for select the objects from the minio, you should give the bucket name that you want to select , in argument.
*/
func (m *Minio) ListObjects(bucket string) ([]ObjectInfo, error) {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return nil, err
	} else {
		result, err := m.Functionality.ListAllObjectsFromMinio(clm, m.Bucket)
		if err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}
}

/*
This function is for move the objects from a bucket to another bucket in minio, you should give the object name that you want to move and sourse bucket and destination bucket , in argument.
*/
func (m *Minio) MoveObject(objname, bucketSrc, bucketDest string) error {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return err
	} else {
		err := m.Functionality.MoveObjectInMinio(clm, bucketSrc, bucketDest, objname)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

/*
This function is for check that bucket exists or not, if it is not create it.
*/
func (m *Minio) CheckBucket(bucket string) error {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return err
	} else {
		err := m.Functionality.CheckBucketExists(clm, bucket)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

/*
This function is for get the object info from minio.
*/
func (m *Minio) ObjectInfo(bucket string, objname string) (interface{}, error) {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return nil, err
	} else {
		info, err := m.Functionality.GetObjectInfo(clm, bucket, objname)
		if err != nil {
			return nil, err
		} else {
			return info, nil
		}
	}
}

/*
This function is for upload the objects into the minio, you should give the path of objects that you want to upload, in argument
*/
func (m *Minio) ListenNotification(prefix, suffix string, events []string) (<-chan notification.Info, error) {
	clm, err := m.Functionality.MinioConnection(m.MinioAddress, m.MinioAccessKey, m.MinioSecretKey, m.Bucket, m.MinioSecure)
	if err != nil {
		return nil, err
	} else {
		return m.Functionality.NotificationFromMinio(clm, prefix, suffix, events), nil
	}

}
