package sideloader

import (
	logger "Zminio/log"
	"Zminio/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
)

func Server(address, kind string, connection *minio.Client) error {
	listener, err := net.Listen(kind, address)
	if err != nil {
		return err
	}

	for {
		con, err := listener.Accept()
		if err != nil {
			logger.ErrorLogger.Printf("error in accpet connection from %v , err = %v \n", con.RemoteAddr().String(), err)
		}

		go handleConnection(con, connection)
	}
}

func handleConnection(con net.Conn, connection *minio.Client) {
	buf := make([]byte, 1024*1024)

	size, err := con.Read(buf)
	if err != nil {
		logger.ErrorLogger.Printf("error in read message from %v , err = %v \n", con.RemoteAddr().String(), err)
	}

	if strings.Contains(string(buf[:size]), "FILE : ") {
		objName := strings.Split(string(buf[:size]), ":")
		logger.SuccessLogger.Printf("request recieve from %v for download %v object \n", con.RemoteAddr().String(), strings.TrimSpace(strings.Split(objName[1], "/-/")[1]))

		filename := fmt.Sprintf("/var/zminio/%s/%s", filepath.Base(strings.TrimSpace(strings.Split(objName[1], "/-/")[1])), filepath.Base(strings.TrimSpace(strings.Split(objName[1], "/-/")[1])))

		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			logger.WarningLogger.Printf("this object { %v } already exist in the path.\n", filepath.Base(strings.TrimSpace(strings.Split(objName[1], "/-/")[1])))
		} else {
			object, err := connection.GetObject(context.Background(), strings.TrimSpace(strings.Split(objName[1], "/-/")[0]), strings.TrimSpace(strings.Split(objName[1], "/-/")[1]), minio.GetObjectOptions{})
			if err != nil {
				if _, err := con.Write([]byte(err.Error())); err != nil {
					logger.ErrorLogger.Printf("error in write message to %v , err = %v \n", con.RemoteAddr().String(), err)
				}
				clear(buf)
			}
			defer object.Close()

			dirPath := filepath.Dir(filename)

			err = os.MkdirAll(dirPath, 0755)
			if err != nil {
				logger.ErrorLogger.Printf("Error creating directories: %v\n", err)
			}

			// Create a local file to save the downloaded object
			localFile, err := os.Create(filename)
			if err != nil {
				if _, err := con.Write([]byte(err.Error())); err != nil {
					logger.ErrorLogger.Printf("error in write message to %v , err = %v \n", con.RemoteAddr().String(), err)
				}
				clear(buf)
			}
			defer localFile.Close()

			// Copy the object content to the local file
			if _, err = io.Copy(localFile, object); err != nil {
				if _, err := con.Write([]byte(err.Error())); err != nil {
					logger.ErrorLogger.Printf("error in write message to %v , err = %v \n", con.RemoteAddr().String(), err)
				}
				clear(buf)
			}
		}

		if result, err := splitFiles(filename); err != nil {
			if _, err := con.Write([]byte(err.Error())); err != nil {
				logger.ErrorLogger.Printf("error in write message to %v , err = %v \n", con.RemoteAddr().String(), err)
			}
			clear(buf)
		} else {
			cr, err := utils.CompressZlib(result)
			if err != nil {
				logger.ErrorLogger.Printf("error in compress json message to %v , err = %v \n", con.RemoteAddr().String(), err)
			}
			if _, err := con.Write(cr); err != nil {
				logger.ErrorLogger.Printf("error in write message to %v , err = %v \n", con.RemoteAddr().String(), err)
			}
			clear(buf)
			clear(result)
		}
	} else if strings.Contains(string(buf[:size]), "GET : ") {
		objName := strings.Split(string(buf[:size]), ":")
		logger.SuccessLogger.Printf("request recieve from %v for GET this %v object \n", con.RemoteAddr().String(), objName[1])

		sendFile(strings.TrimSpace(objName[1]), con)
	} else if strings.Contains(string(buf[:size]), "DELETE : ") {
		objName := strings.Split(string(buf[:size]), ":")
		logger.InfoLogger.Printf("request receive from %v for DELETE this %v object \n", con.RemoteAddr().String(), objName[1])

		if err := os.RemoveAll("/var/zminio/" + filepath.Base(objName[1])); err != nil {
			logger.ErrorLogger.Printf("error in delete this path %v , error = %v \n", objName[1], err)
			con.Write([]byte("ERROR"))
		} else {
			logger.SuccessLogger.Printf("object %v deleted successfully!", objName[1])
			con.Write([]byte("OK"))
		}
	} else {
		if _, err := con.Write([]byte("EOF")); err != nil {
			logger.ErrorLogger.Printf("error in write message to %v , err = %v \n", con.RemoteAddr().String(), err)
		}
	}
}

func splitFiles(filename string) ([]byte, error) {
	cmd := exec.Command("split", "-db", fmt.Sprintf("%vM", os.Getenv("SIDELOADER_SIZE")), "--suffix-length=3", "--additional-suffix="+filepath.Ext(filename), filename, fmt.Sprintf("/var/zminio/%v%s/split_%v", strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), filepath.Ext(filename), strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))))
	if result, err := cmd.CombinedOutput(); err != nil {
		logger.ErrorLogger.Printf("%s\n", result)
		return nil, err
	}

	list, err := filepath.Glob(fmt.Sprintf("/var/zminio/%v%s/split_*", strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), filepath.Ext(filename)))
	if err != nil {
		return nil, err
	}

	fileList := make([]string, 0)
	var wg sync.WaitGroup
	for cf := range slices.Chunk(list, 5) {
		for _, sf := range cf {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hash, err := utils.CalculateSHA1(sf)
				if err != nil {
					return
				}
	
				newpath := fmt.Sprintf("%v/%v#####%v", filepath.Dir(sf), hash, filepath.Base(sf))
				fileList = append(fileList, filepath.Dir(sf)+"/"+filepath.Base(newpath))
				if err := os.Rename(sf, newpath); err != nil {
					return
				}
			}()
		}
		wg.Wait()
	}

	fileListByte, err := json.Marshal(fileList)
	if err != nil {
		return nil, err
	}
	clear(fileList)

	return fileListByte, nil
}
