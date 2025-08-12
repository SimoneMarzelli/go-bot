package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			log.Fatalf("Error closing zip reader: %v", err)
		}
	}(r)

	for _, f := range r.File {
		filePath := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(filePath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		// Make sure directory exists
		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		if s, c := strings.CutSuffix(filePath, ".txt"); c {
			filePath = s + ".csv"
		}

		// Create destination file
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		HandleFileClose(outFile)
		HandleReaderClose(&rc)

		if err != nil {
			return err
		}
	}
	return nil
}

func DownloadFile(url string, outFile string) error {
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error downloading %v", outFile)
	}

	defer HandleReaderClose(&response.Body)

	out, err := os.Create(outFile)
	if err != nil {
		log.Fatal("Could not fetch live data", err)
	}
	defer HandleFileClose(out)

	_, err = io.Copy(out, response.Body)
	return err
}

func ReadRemoteFile(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}

	responseData := make([]byte, 10)

	_, err = response.Body.Read(responseData)
	if err != nil {
		return "", fmt.Errorf("error reading remote file: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return "", fmt.Errorf("error closing body: %w", err)
	}

	return string(responseData), nil

}

func HandleFileClose(filePtr *os.File) {
	err := filePtr.Close()
	if err != nil {
		log.Fatalf("Error closing file %v: %v", filePtr.Name(), err)
	}
}

func HandleReaderClose(closer *io.ReadCloser) {
	err := (*closer).Close()
	if err != nil {
		log.Fatalf("Error closing reade:: %v", err)
	}
}
