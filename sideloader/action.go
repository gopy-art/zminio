package sideloader

import (
	"Zminio/console"
	logger "Zminio/log"
	"os"

	"github.com/minio/minio-go/v7"
)

func SIDELOADER(srcconnection *minio.Client) {
	if err := Validation(); err != nil {
		logger.ErrorLogger.Fatalf("error in validating sideloader config, error = %v \n", err)
	}
	if console.SideLoaderType == "server" {
		if err := ServerSideloader(os.Getenv("SIDELOADER_ADDRESS"), srcconnection); err != nil {
			logger.ErrorLogger.Fatalf("error in load server, err = %v \n", err)
		}
	}
}

func ServerSideloader(listenAddress string, connection *minio.Client) error {
	if err := Server(os.Getenv("SIDELOADER_ADDRESS"), "tcp", connection); err != nil {
		return err
	}

	return nil
}

func ClientSideloader(object, srcBucket, dstBucket string, connection *minio.Client) error {
	if err := Client(os.Getenv("SIDELOADER_SERVER_ADDRESS"), "tcp", object, srcBucket, dstBucket, connection); err != nil {
		return err
	}
	return nil
}
