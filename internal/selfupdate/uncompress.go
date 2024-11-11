package selfupdate

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

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// check for ZipSlip (Directory traversal).
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func untarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(dest, header.Name)

		// check for ZipSlip (Directory traversal).
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(path, 0755)
		case tar.TypeReg:
			err := writeTarFile(tarReader, path, header.Mode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func writeTarFile(tarReader *tar.Reader, path string, mode int64) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, tarReader)
	return err
}

func Unarchive(src, dest string) error {
	switch {
	case strings.HasSuffix(src, ".zip"):
		return unzip(src, dest)
	case strings.HasSuffix(src, ".tar.gz"), strings.HasSuffix(src, ".tgz"):
		return untarGz(src, dest)
	default:
		return fmt.Errorf("unsupported archive format: %s", src)
	}
}
