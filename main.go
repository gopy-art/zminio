package main

import (
	"Zminio/console"
	"Zminio/helper"
	logger "Zminio/log"
	"os"

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
		logger.InitLoggerFile(zMinioBaseDir + "/zsploit.log")
	}

}

func main() {
	helper.ActionHelper()
}
