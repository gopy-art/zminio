package utils

import (
	"bytes"
	"compress/zlib"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func RemoveLastTwoDigits(filename string) string {
    base := filepath.Base(filename)
    ext := filepath.Ext(filename)
    nameWithoutExt := base[:len(base)-len(ext)]

    re := regexp.MustCompile(`\d{3}$`)
    newName := re.ReplaceAllString(nameWithoutExt, "")

    return newName + ext
}

func CalculateSHA1(filename string) (string, error) {
	cmd := exec.Command("sha1sum", filename)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Extract the hash from the output
	hash := strings.Split(string(output), " ")[0]
	return hash, nil
}

func CompressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	w.Close()
	return buf.Bytes(), nil
}

func DecompressZlib(compressedData []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var out bytes.Buffer
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}