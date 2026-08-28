package mageutil

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/pgzip"
	"github.com/openimsdk/gomake/internal/util"
)

type ArchiveOptions struct {
	ProjectName *string
}

func (opt *ArchiveOptions) GetProjectName() string {
	projectName := strings.TrimSpace(util.NilAsZero(util.NilAsZero(opt).ProjectName))
	if projectName == "" {
		return ""
	}
	return strings.NewReplacer("/", "_", "\\", "_").Replace(projectName)
}

func archive(archivePath string, mappingPaths map[string]string) error {
	archivePath = fmt.Sprintf("%s.tar.gz", archivePath)
	PrintBlue(fmt.Sprintf("Creating archive: %s", archivePath))
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file %s: %v", archivePath, err)
	}
	defer archiveFile.Close()
	gzipWriter, err := pgzip.NewWriterLevel(archiveFile, pgzip.BestCompression)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %v", err)
	}
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for in, out := range mappingPaths {
		err := util.CheckExist(in)
		if err != nil {
			return err
		}

		PrintBlue(fmt.Sprintf("Adding %s to archive", in))
		if err := util.AddToTar(tarWriter, in, out); err != nil {
			return fmt.Errorf("failed to add %s to archive: %v", in, err)
		}
	}

	PrintGreen(fmt.Sprintf("Archive created successfully: %s", archivePath))
	return nil
}

func ArchiveProject(archiveOptions *ArchiveOptions) error {
	archiveDir := Paths.OutputArchive
	PrintBlue(fmt.Sprintf("Using archive directory: %s", archiveDir))

	allFiles, err := GetAllRootFilesExcludeIgnore()
	if err != nil {
		return err
	}
	mappingPaths, err := EnsureRootRelPaths(allFiles...)
	if err != nil {
		return err
	}

	archiveName := "archived"
	projectName := archiveOptions.GetProjectName()
	if projectName != "" {
		archiveName = fmt.Sprintf("archived_%s", projectName)
	}
	return archive(filepath.Join(archiveDir, archiveName), mappingPaths)
}
