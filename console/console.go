package console

import (
	"flag"
	"fmt"
	"os"
)

var (
	Secret_key      string
	Access_key      string
	Bucket          string
	Url             string
	Secret_key_sync string
	Access_key_sync string
	Bucket_sync     string
	Url_sync        string
	Pathfile        string
	Type            string
	Logger          string
	OutPutFile      string
	Object          string
	MBucket         string
	NumberOfWorker  int
)

var AppVersion string = "v2.1.0"

// initial cli commands
func InitConsole() {
	flag.StringVar(&Secret_key, "sk", "", "set your minio secret key")
	flag.StringVar(&Access_key, "ak", "", "set your minio access key")
	flag.StringVar(&Url, "u", "", "set your minio url address")
	flag.StringVar(&Bucket, "b", "", "set your minio bucket name")
	flag.StringVar(&Secret_key_sync, "sks", "", "set your minio secret key")
	flag.StringVar(&Access_key_sync, "aks", "", "set your minio access key")
	flag.StringVar(&Url_sync, "us", "", "set your minio url address")
	flag.StringVar(&Bucket_sync, "bs", "", "set your minio bucket name")
	flag.StringVar(&MBucket, "mb", "", "set your minio bucket name that you want to move files to it")
	flag.StringVar(&Object, "obj", "", "set your minio object name")
	flag.StringVar(&Pathfile, "f", "", "set the path of the file that you wanna upload")
	flag.StringVar(&OutPutFile, "o", "", "set the path of the file that you wanna download")
	flag.StringVar(&Logger, "l", "stdout", "set app logger type , stdout or file")
	flag.StringVar(&Type, "do", "", "set the job you want to do. (download, upload, move, delete, list, sync, uploadDir)")
	flag.IntVar(&NumberOfWorker, "n", 10, "set the count of worker for run")
	version := flag.Bool("v", false, "zsploit version")
	flag.Parse()

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	if Secret_key == "" {
		Secret_key = os.Getenv("MINIO_SECRET_KEY")
	}
	if Access_key == "" {
		Access_key = os.Getenv("MINIO_ACCESS_KEY")
	}
	if Url == "" {
		Url = os.Getenv("MINIO_ENDPOINT")
	}
	if Bucket == "" {
		Bucket = os.Getenv("MINIO_BUCKET_NAME")
	}
}
