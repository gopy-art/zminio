package sideloader

import (
	logger "Zminio/log"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type MetaData struct {
	name     string
	fileSize uint32
	reps     uint32
}

func prepareMetadata(file *os.File) MetaData {
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	size := fileInfo.Size()
	header := MetaData{
		name:     file.Name(),
		fileSize: uint32(size),
		reps:     uint32(size/1014) + 1,
	}
	return header
}

func sendFile(path string, conn net.Conn) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		logger.ErrorLogger.Println(err)
	}

	header := prepareMetadata(file)
	dataBuffer := make([]byte, 1014)
	headerBuffer := []byte{1}
	segmentBuffer := []byte{0}
	temp := make([]byte, 4)
	received := make([]byte, 100)
	for i := 0; i < int(header.reps); i++ {
		n, _ := file.ReadAt(dataBuffer, int64(i*1014))

		if i == 0 {
			// Number of segments
			binary.BigEndian.PutUint32(temp, header.reps)
			headerBuffer = append(headerBuffer, temp...)

			// Length of name
			binary.BigEndian.PutUint32(temp, uint32(len(header.name)))
			headerBuffer = append(headerBuffer, temp...)

			// Name
			headerBuffer = append(headerBuffer, []byte(header.name)...)

			headerBuffer = append(headerBuffer, 0)

			_, err := conn.Write(headerBuffer)

			if err != nil {
				logger.ErrorLogger.Println(err)
			}

			_, err = conn.Read(received)

			if err != nil {
				logger.ErrorLogger.Println(err)
			}
			clear(received)
		}

		// Segment number
		binary.BigEndian.PutUint32(temp, uint32(i))
		segmentBuffer = append(segmentBuffer, temp...)

		// Length of data
		binary.BigEndian.PutUint32(temp, uint32(n))
		segmentBuffer = append(segmentBuffer, temp...)

		// Data
		segmentBuffer = append(segmentBuffer, dataBuffer...)

		segmentBuffer = append(segmentBuffer, 1)

		_, err = conn.Write(segmentBuffer)

		if err != nil {
			logger.ErrorLogger.Println(err)
		}

		_, err = conn.Read(received)
		if err != nil || !strings.Contains(string(received), "OK") {
			if err == io.EOF {
				logger.ErrorLogger.Printf("EOF error from %v \n", conn.RemoteAddr().String())
				conn.Close()
				break
			}
			logger.ErrorLogger.Printf("error received from %v with this payload : %v \n", conn.RemoteAddr().String(), err)
		}

		if strings.Contains(string(received), "DONE") {
			logger.ErrorLogger.Printf("file %v has been transfered successfully", filepath.Base(path))
		} else if strings.Contains(string(received), "WRONG") {
			logger.ErrorLogger.Printf("file %v has been transfered unsuccessfully", filepath.Base(path))
		}

		// Reset segment buffer
		segmentBuffer = []byte{0}
	}
}
