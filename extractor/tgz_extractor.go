package extractor

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type tgzExtractor struct{}

func NewTgz() Extractor {
	return &tgzExtractor{}
}

func (e *tgzExtractor) Extract(src, dest string) error {
	srcType, err := mimeType(src)
	if err != nil {
		return err
	}

	switch srcType {
	case "application/x-gzip":
		err := extractTgz(src, dest)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s is not a tgz archive: %s", src, srcType)
	}

	return nil
}

func extractTgz(src, dest string) error {
	cmd, err := tarFromShell(src, dest)
	if err == nil {
		return cmd.Run()
	}

	fd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fd.Close()

	gReader, err := gzip.NewReader(fd)
	if err != nil {
		return err
	}
	defer gReader.Close()

	tarReader := tar.NewReader(gReader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if hdr.Name == "." {
			continue
		}

		err = extractTarArchiveFile(hdr, dest, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractTarArchiveFile(header *tar.Header, dest string, input io.Reader) error {
	filePath := filepath.Join(dest, header.Name)
	fileInfo := header.FileInfo()

	if fileInfo.IsDir() {
		err := os.MkdirAll(filePath, fileInfo.Mode())
		if err != nil {
			return err
		}
	} else {
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return err
		}

		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return os.Symlink(header.Linkname, filePath)
		}

		fileCopy, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}
		defer fileCopy.Close()

		_, err = io.Copy(fileCopy, input)
		if err != nil {
			return err
		}
	}

	return nil
}
