package util

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

func AddToTar(tarWriter *tar.Writer, filePath, archivePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return addDirToTar(tarWriter, filePath, archivePath)
	}
	return addFileToTar(tarWriter, filePath, archivePath)
}

func addFileToTar(tarWriter *tar.Writer, filePath, archivePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    archivePath,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	return err
}

func addDirToTar(tarWriter *tar.Writer, dirPath, archiveDirName string) error {
	if archiveDirName != "" {
		rootHeader := &tar.Header{
			Name:     archiveDirName + "/",
			Mode:     0755,
			ModTime:  time.Now(),
			Typeflag: tar.TypeDir,
		}
		if err := tarWriter.WriteHeader(rootHeader); err != nil {
			return err
		}
	}

	return filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, filePath)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		archivePath := path.Join(archiveDirName, relPath)

		if info.IsDir() {
			header := &tar.Header{
				Name:     archivePath + "/",
				Mode:     int64(info.Mode()),
				Typeflag: tar.TypeDir,
			}
			return tarWriter.WriteHeader(header)
		}

		return addFileToTar(tarWriter, filePath, archivePath)
	})
}
