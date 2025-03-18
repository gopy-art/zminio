package sideloader

import (
	"Zminio/console"
	"fmt"
	"os"
)

func Validation() error {
	if console.SideLoader && (os.Getenv("SIDELOADER_ADDRESS") == "" || os.Getenv("SIDELOADER_SIZE") == "") {
		return fmt.Errorf("SIDELOADER_ADDRESS or SIDELOADER_SIZE in .env file is empty")
	}
	if console.SideLoader && console.SideLoaderType != "server" && console.SideLoaderType != "client" {
		return fmt.Errorf("sideloader type is invalid")
	}
	return nil
}