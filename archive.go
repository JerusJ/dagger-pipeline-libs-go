package pipeline

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CreateArchive(globPattern string, archiveFile string) error {
	switch ext := filepath.Ext(archiveFile); ext {
	case ".zip":
		return createZip(globPattern, archiveFile)
	case ".gz":
		return createTarGz(globPattern, archiveFile)
	default:
		return fmt.Errorf("Unsupported archive extension: '%s'", ext)
	}
}

func createZip(globPattern string, zipFile string) error {
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return err
	}

	zipf, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer zipf.Close()

	zipw := zip.NewWriter(zipf)
	defer zipw.Close()

	for _, match := range matches {
		err = filepath.Walk(match, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				header, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}
				header.Name = strings.TrimPrefix(path, filepath.Dir(globPattern)+"/")
				header.Method = zip.Deflate

				writer, err := zipw.CreateHeader(header)
				if err != nil {
					return err
				}

				_, err = io.Copy(writer, file)
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func createTarGz(globPattern string, tarGzFile string) error {
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return err
	}

	file, err := os.Create(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, match := range matches {
		err = filepath.Walk(match, func(path string, info os.FileInfo, err error) error {
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = strings.TrimPrefix(path, filepath.Dir(globPattern)+"/")

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(tw, file)
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Usage
func main() {
	if err := CreateArchive("path/to/directory/*", "archive.zip"); err != nil {
		panic(err)
	}
	if err := CreateArchive("path/to/directory/*", "archive.tar.gz"); err != nil {
		panic(err)
	}
}
