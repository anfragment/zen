package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip reader: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("close zip reader: %v", err)
		}
	}()

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip file: %w", err)
		}
		defer func() {
			if err := rc.Close(); err != nil {
				log.Printf("close zip file: %v", err)
			}
		}()

		path := filepath.Join(dest, f.Name) // #nosec G305

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
				return fmt.Errorf("open file: %w", err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					log.Printf("close file: %v", err)
				}
			}()

			const maxDecompressedSize int64 = 100 << 20 // 100 MB

			// Limit the size of the file to prevent zip bombs. G110 gosec
			limitedReader := &io.LimitedReader{R: rc, N: maxDecompressedSize}
			_, err = io.Copy(f, limitedReader)
			if err != nil {
				return fmt.Errorf("copy file: %w", err)
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return fmt.Errorf("extract and write file: %w", err)
		}
	}

	return nil
}

func untarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		path := filepath.Join(dest, header.Name) // #nosec G305

		// check for ZipSlip (Directory traversal).
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(path, 0755)
			if err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		case tar.TypeReg:
			err := writeTarFile(tarReader, path, os.FileMode(header.Mode)) // #nosec G115
			if err != nil {
				return fmt.Errorf("write file: %w", err)
			}
		}
	}
	return nil
}

func writeTarFile(tarReader *tar.Reader, path string, mode os.FileMode) error {
	if mode > 0o777 {
		return fmt.Errorf("invalid file mode: %d", mode)
	}

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, tarReader)
	return err
}

func unarchive(src, dest string) error {
	switch {
	case strings.HasSuffix(src, ".zip"):
		return unzip(src, dest)
	case strings.HasSuffix(src, ".tar.gz"), strings.HasSuffix(src, ".tgz"):
		return untarGz(src, dest)
	default:
		return fmt.Errorf("unsupported archive format: %s", src)
	}
}
