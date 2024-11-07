package main

import (
	"Zminio/console"
	"Zminio/helper"
	logger "Zminio/log"
	"Zminio/prometheus"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load(".env")
	zMinioBaseDir, _ := os.Getwd()
	// set the flags
	console.InitConsole()
	// init logger
	if console.Logger == "stdout" {
		logger.InitLoggerStdout()
	} else if console.Logger == "file" {
		logger.InitLoggerFile(zMinioBaseDir + "/zminio.log")
	}

}

func main() {
	if console.Prometheus != "" {
		prometheusIp := strings.Split(console.Prometheus, ":")[0]
		if prometheus.IsValidIpv4(prometheusIp) {
			go prometheus.ControllerPrometheusInit(console.Prometheus)
			time.Sleep(time.Second * 1)
		} else {
			logger.ErrorLogger.Fatalln("the ip address is not valid for prometheus!")
		}
	}

	helper.ActionHelper()
}
