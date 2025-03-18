package console

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
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
	Prometheus      string
	EnvFile         string
	SideLoaderType  string
	SecureSSL       bool
	SecureSSL_sync  bool
	DeleteInSync    bool
	ListenSync      bool
	SaveObjects     bool
	Version         bool
	CacheUsage      bool
	MinioCache      bool
	SyncAll         bool
	SideLoader      bool
	NumberOfWorker  int
	Interval        int
	MaxSizeSideload int64
)

var rootCmd = &cobra.Command{
	Use:   "zminio",
	Short: "fast and efficient minio client written in golang",
	Run: func(cmd *cobra.Command, args []string) {
		if Version {
			fmt.Println(AppVersion)
			os.Exit(0)
		}
	},
}

var AppVersion string = "v2.7.0"

// initial cli commands
func init() {
	rootCmd.Flags().StringVar(&Access_key, "ak", "", "set your minio access key")
	rootCmd.Flags().StringVar(&Secret_key, "sk", "", "set your minio secret key")
	rootCmd.Flags().StringVarP(&Url, "endpoint", "u", "", "set your minio url address")
	rootCmd.Flags().StringVarP(&Bucket, "bucket", "b", "", "set your minio bucket name")
	rootCmd.Flags().StringVar(&Secret_key_sync, "sks", "", "set your minio secret key")
	rootCmd.Flags().StringVar(&Access_key_sync, "aks", "", "set your minio access key")
	rootCmd.Flags().StringVar(&Url_sync, "us", "", "set your minio url address")
	rootCmd.Flags().StringVar(&Bucket_sync, "bs", "", "set your minio bucket name")
	rootCmd.Flags().StringVar(&MBucket, "mb", "", "set your minio bucket name that you want to move files to it")
	rootCmd.Flags().StringVar(&Object, "obj", "", "set your minio object name")
	rootCmd.Flags().StringVarP(&Pathfile, "input", "f", "", "set the path of the file that you wanna upload")
	rootCmd.Flags().StringVarP(&OutPutFile, "output", "o", "", "set the path of the file that you wanna download")
	rootCmd.Flags().StringVarP(&Logger, "logger", "l", "stdout", "set app logger type , stdout or file")
	rootCmd.Flags().StringVarP(&SideLoaderType, "type", "t", "", "set the side loader type. (server, client)")
	rootCmd.Flags().StringVar(&EnvFile, "env", "", "set your env file path")
	rootCmd.Flags().StringVar(&Prometheus, "pr", "", "run Prometheus on ip:port to monitor aminio metrics,if not set this flag prometheus disabled. (exaple:-pr 0.0.0.0:1234)")
	rootCmd.Flags().StringVar(&Type, "do", "", "set the job you want to do. (download, upload, move, delete, list, sync, uploadDir)")
	rootCmd.Flags().IntVarP(&NumberOfWorker, "workers", "n", 10, "set the count of worker for run.")
	rootCmd.Flags().IntVarP(&Interval, "interval", "i", 1, "set the interval for sync objects")
	rootCmd.Flags().BoolVar(&DeleteInSync, "ds", false, "delete the object from bucket after sync")
	rootCmd.Flags().BoolVar(&SideLoader, "sideloader", false, "sideloader for transfer large objects over tcp connection.")
	rootCmd.Flags().BoolVar(&SecureSSL, "se", false, "set your secure ssl option in connecting to the minio. (default false)")
	rootCmd.Flags().BoolVar(&SecureSSL_sync, "ses", false, "set your secure ssl option in connecting to the minio. (default false)")
	rootCmd.Flags().BoolVar(&ListenSync, "ls", false, "set the listen bucket on sync proccess!")
	rootCmd.Flags().BoolVar(&CacheUsage, "cache", false, "use the redis as a cache for the sync proccess!")
	rootCmd.Flags().BoolVar(&MinioCache, "mincache", false, "use the minio as a cache for the sync proccess!")
	rootCmd.Flags().BoolVar(&SaveObjects, "save", false, "save the objects throw the sync process!")
	rootCmd.Flags().BoolVar(&SyncAll, "all", false, "sync all the buckets and objects")
	rootCmd.Flags().BoolVarP(&Version, "version", "v", false, "zminio version")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ValidateFlags() {
	if Type == "" && !SideLoader {
		fmt.Println("you have to set --do flag")
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
	if os.Getenv("MINIO_SSL_SECRET") != "" {
		if os.Getenv("MINIO_SSL_SECRET") == "true" || os.Getenv("MINIO_SSL_SECRET") == "True" {
			SecureSSL = true
		} else {
			SecureSSL = false
		}
	}

	if Type == "sync" {
		if Secret_key_sync == "" {
			Secret_key_sync = os.Getenv("SYNC_MINIO_SECRET_KEY")
		}
		if Access_key_sync == "" {
			Access_key_sync = os.Getenv("SYNC_MINIO_ACCESS_KEY")
		}
		if Url_sync == "" {
			Url_sync = os.Getenv("SYNC_MINIO_ENDPOINT")
		}
		if Bucket_sync == "" {
			Bucket_sync = os.Getenv("SYNC_MINIO_BUCKET_NAME")
		}
		if os.Getenv("SYNC_MINIO_SSL_SECRET") != "" {
			if os.Getenv("SYNC_MINIO_SSL_SECRET") == "true" || os.Getenv("SYNC_MINIO_SSL_SECRET") == "True" {
				SecureSSL_sync = true
			} else {
				SecureSSL_sync = false
			}
		}
	}

	if SideLoader && SideLoaderType == "" {
		fmt.Println("you did not set the sideloader type.")
		os.Exit(0)
	}

	if os.Getenv("SIDELOADER_MAXIMUM_SIZE_START") != "" {
		intsize, err := strconv.ParseInt(os.Getenv("SIDELOADER_MAXIMUM_SIZE_START"), 15, 64)
		if err != nil {
			fmt.Println("the type of SIDELOADER_MAXIMUM_SIZE_START is invalid")
			os.Exit(0)
		}
		MaxSizeSideload = intsize * 1024 * 1024
	}
}
