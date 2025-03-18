package sideloader

import (
	"Zminio/console"
	logger "Zminio/log"
	"Zminio/prometheus"
	"Zminio/utils"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
)

func Client(server, kind, object, srcBucket, dstBucket string, connection *minio.Client) error {
	con, err := net.Dial(kind, server)
	if err != nil {
		return err
	}

	if console.Prometheus != "" {
		prometheus.SetSideloader(fmt.Sprintf("%v | %v", strings.Split(object, "/-/")[1], srcBucket))
	}

	buf := make([]byte, 50*1024*1024)
	msg := fmt.Sprintf("FILE : %v", object)
	if _, err := con.Write([]byte(msg)); err != nil {
		return err
	}

	size, err := con.Read(buf)
	if err != nil {
		return err
	}

	res, err := utils.DecompressZlib(buf[:size])
	if err != nil {
		return err
	}

	FILE_GLOG := make([]string, 0)
	if err := json.Unmarshal(res, &FILE_GLOG); err != nil {
		return err
	}

	logger.SuccessLogger.Println("Response from server was successful; starting to download splited files")
	con.Close()

	var wg sync.WaitGroup
	var numw int
	if console.NumberOfWorker <= 3 {
		numw = 1
	} else {
		numw = console.NumberOfWorker / 3
	}

	for file := range slices.Chunk(FILE_GLOG, numw) {
		for _, f := range file {
			conF, err := net.Dial(kind, server)
			if err != nil {
				return err
			}
			defer conF.Close()

			msg := fmt.Sprintf("GET : %v", f)
			if _, err := conF.Write([]byte(msg)); err != nil {
				return err
			}

			wg.Add(1)
			go handleIncomingFileRequest(conF, &wg)
		}
		wg.Wait()
	}

	logger.SuccessLogger.Println("all splited files transfer successfully!")
	logger.InfoLogger.Println("we are starting to merge transfered files.")

	output, err := mergeFilesToUpload("/var/zminio/")
	if err != nil {
		logger.ErrorLogger.Printf("error in merge transfered files, error = %v\n", err)
	}

	if _, errp := connection.FPutObject(context.Background(), dstBucket, strings.TrimSpace(strings.Split(object, "/-/")[1]), output, minio.PutObjectOptions{}); errp != nil {
		return errp
	}
	logger.SuccessLogger.Printf("The object with name {%v} moved from this bucket {%v} to the this bucket {%v} successfully.", strings.TrimSpace(strings.Split(object, "/-/")[1]), srcBucket, dstBucket)

	if !console.SaveObjects {
		if err := os.Remove("/var/zminio/" + strings.TrimSpace(strings.Split(object, "/-/")[1])); err != nil {
			return err
		} else {
			logger.SuccessLogger.Printf("file { %v } deleted after uploaded in the minio.", strings.TrimSpace(strings.Split(object, "/-/")[1]))
		}
	}

	conD, errD := net.Dial(kind, server)
	if errD != nil {
		return errD
	}

	msg = fmt.Sprintf("DELETE : %v", object)
	if _, errD := conD.Write([]byte(msg)); errD != nil {
		return errD
	}

	sizeD, errD := conD.Read(buf)
	if errD != nil {
		return errD
	}

	if strings.Contains(strings.TrimSpace(string(buf[:sizeD])), "OK") {
		logger.SuccessLogger.Printf("Ok response receive from server for deleting this file { %v }\n", strings.TrimSpace(strings.Split(object, "/-/")[1]))
	} else {
		logger.WarningLogger.Printf("Server did not remove this object { %v } \n", strings.TrimSpace(strings.Split(object, "/-/")[1]))
	}

	if console.Prometheus != "" {
		prometheus.ResetSideloader()
	}

	return nil
}

func handleIncomingFileRequest(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.InfoLogger.Printf("Received a header file request from : %v \n", conn.RemoteAddr().String())
	headerBuffer := make([]byte, 1024)

	_, err := conn.Read(headerBuffer)
	if err != nil {
		logger.ErrorLogger.Println(err)
	}

	var name string
	var reps uint32

	if headerBuffer[0] == byte(1) && headerBuffer[1023] == byte(0) {
		reps = binary.BigEndian.Uint32(headerBuffer[1:5])
		lengthOfName := binary.BigEndian.Uint32(headerBuffer[5:9])
		name = string(headerBuffer[9 : 9+lengthOfName])
	} else {
		logger.ErrorLogger.Println("Invalid header")
	}

	conn.Write([]byte("Header Received"))

	dataBuffer := make([]byte, 1024)

	filename := fmt.Sprintf("/var/zminio/%v", strings.Split(name, "#####")[1])
	dirPath := filepath.Dir(filename)
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		logger.ErrorLogger.Printf("Error creating directories: %v\n", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		logger.ErrorLogger.Printf("error creating file %v: %v", filename, err)
	}

	logger.SuccessLogger.Printf("[ INFO ] : starting to download this object %v.\n", filepath.Base(filename))

	for i := 0; i < int(reps); i++ {
		_, err := conn.Read(dataBuffer)
		if err != nil {
			logger.ErrorLogger.Fatal(err)
		}

		if dataBuffer[0] == byte(0) && dataBuffer[1023] == byte(1) {
			length := binary.BigEndian.Uint32(dataBuffer[5:9])
			file.Write(dataBuffer[9 : 9+length])
		} else {
			logger.ErrorLogger.Println("Invalid Segment")
		}

		conn.Write([]byte("OK"))
	}

	hash, err := utils.CalculateSHA1(filename)
	if err != nil {
		logger.ErrorLogger.Printf("the hash of the file %v is incorrect, file transfered unsuccessfully!", filepath.Base(filename))
	}
	if hash == filepath.Base(strings.TrimSpace(strings.Split(name, "#####")[0])) {
		conn.Write([]byte("DONE"))
		logger.SuccessLogger.Printf("file with name %v transfered successfully!", filename)
	} else {
		conn.Write([]byte("WRONG"))
		logger.ErrorLogger.Printf("file with name %v transfered unsuccessfully!", filename)
	}

	logger.SuccessLogger.Printf("[ DONE ] : received bytes have been written into file %v.\n", filepath.Base(filename))

	file.Close()
	conn.Close()
}

func mergeFilesToUpload(path string) (string, error) {
	var output string
	splitedFiles, err := filepath.Glob(path + "split_*")
	if err != nil {
		return "", err
	}

	output = path + strings.Split(utils.RemoveLastTwoDigits(splitedFiles[0]), "split_")[1]

	cmd := exec.Command("sh", "-c", fmt.Sprintf("cat %vsplit_* > '%v'", path, output))
	if result, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s | %v", result, err)
	}

	for _, f := range splitedFiles {
		if err := os.Remove(f); err != nil {
			logger.ErrorLogger.Printf("error in remove this file %v , err = %v", f, err)
		}
	}

	logger.SuccessLogger.Println("all splited files after merge removed successfully")

	return output, nil
}
