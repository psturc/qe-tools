package oci

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	extractTimeout = 1 * time.Minute
)

// GzFileInfo holds information about a gzipped file and its directory.
type GzFileInfo struct {
	// FilePath represents the full path to the gzipped file.
	FilePath string

	// DirPath represents the directory where the gzipped file are stored.
	DirPath string
}

// GetGzFilesFromDir retrieves all .gz files in the specified directory
func (c *Controller) GetGzFilesFromDir(dir string) ([]GzFileInfo, error) {
	var gzFiles []GzFileInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".gz") {
			gzFiles = append(gzFiles, GzFileInfo{
				FilePath: path,
				DirPath:  filepath.Dir(path),
			})
		}
		return nil
	})

	return gzFiles, err
}

// ExtractGzFile extracts a .gz file to the specified output directory.
// It decompresses the .gz file into its original file.
func (c *Controller) ExtractGzFile(gzFilePath, destDir string) error {
	// #nosec G304
	gzFile, err := os.Open(gzFilePath)
	if err != nil {
		return fmt.Errorf("failed to open .gz file: %w", err)
	}
	defer gzFile.Close()

	gzFileStats, err := os.Stat(gzFilePath)
	if err != nil {
		return fmt.Errorf("failed to get stat for .gz file: %w", err)
	}

	// If the file size is 0, it's considered empty. Return nil to skip extraction
	if gzFileStats.Size() == 0 {
		return nil
	}

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	outputFileName := gzReader.Name
	if outputFileName == "" {
		outputFileName = strings.TrimSuffix(filepath.Base(gzFilePath), ".gz")
	}

	outputFilePath := filepath.Join(destDir, outputFileName)
	// #nosec G304
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// #nosec G110
	_, err = io.Copy(outputFile, gzReader)
	if err != nil {
		// Check for EOF error, and treat it as normal if it occurs.
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("failed to write decompressed data to file: %w", err)
	}

	return nil
}

// HandleBlob handles the extraction of individual blobs.
// It manages concurrency with WaitGroup and semaphore for blob processing.
func (c *Controller) HandleBlob(blobPath, outputDir string, wg *sync.WaitGroup, errors chan<- error, sem chan struct{}) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()

	// Process the blob file for extraction
	if err := c.processBlob(blobPath, outputDir); err != nil {
		errors <- err
	}
}

// Extracts tar.gz files to a specified destination.
// It takes an io.Reader for the gzip stream and the destination path.
func (c *Controller) extractTarGz(gzipStream io.Reader, dest string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	// Iterate through the tar entries
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		// #nosec G305
		destPath := filepath.Join(dest, header.Name)
		if err := c.handleTarEntry(header, tarReader, destPath); err != nil {
			return err
		}
	}
	return nil
}

// Handles individual entries in the tar archive.
// It creates directories or files as specified in the tar header.
func (c *Controller) handleTarEntry(header *tar.Header, tarReader *tar.Reader, destPath string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(destPath, 0o750)
	case tar.TypeReg:
		if _, err := os.Stat(destPath); err == nil {
			return nil
		}
		return c.createFileFromTar(tarReader, destPath)
	default:
		return fmt.Errorf("unsupported tar entry: %c", header.Typeflag)
	}
}

// Creates a file from the tar reader.
// It writes the contents of the tar entry to a newly created file.
func (c *Controller) createFileFromTar(tarReader *tar.Reader, destPath string) error {
	cleanDestPath := filepath.Clean(destPath)

	outFile, err := os.Create(cleanDestPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", cleanDestPath, err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return fmt.Errorf("failed to write file %s: %w", cleanDestPath, err)
	}
	return nil
}

// Processes the blob file for extraction.
// It checks for file existence, size, and identifies if it's a tar.gz blob.
func (c *Controller) processBlob(blobPath, outputDir string) error {
	// Normalize the path to prevent directory traversal
	cleanBlobPath := filepath.Clean(blobPath)

	// Check file existence and size
	fileInfo, err := os.Stat(cleanBlobPath)
	if err != nil {
		return fmt.Errorf("failed to stat blob %s: %w", cleanBlobPath, err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("blob %s is empty, skipping", cleanBlobPath)
	}

	// Open the file safely
	file, err := os.Open(cleanBlobPath)
	if err != nil {
		return fmt.Errorf("failed to open blob %s: %w", cleanBlobPath, err)
	}
	defer file.Close()

	// Check if the blob is a tar.gz and process it if so
	if isTarGzBlob(cleanBlobPath, file) {
		return c.extractBlob(cleanBlobPath, file, outputDir)
	}

	return nil
}

// Determines if the blob is a tar.gz file.
// It checks the file extension and the header bytes for gzip format.
func isTarGzBlob(blobPath string, file *os.File) bool {
	buf := make([]byte, 2)
	if _, err := file.Read(buf); err != nil {
		return false
	}
	if _, err := file.Seek(0, 0); err != nil {
		return false
	}
	return strings.HasSuffix(blobPath, ".tar.gz") || (len(buf) == 2 && buf[0] == 0x1F && buf[1] == 0x8B)
}

// Extracts a tar.gz blob to the specified output directory.
// It handles timeouts during extraction.
func (c *Controller) extractBlob(blobPath string, file *os.File, outputDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), extractTimeout)
	defer cancel()

	extractErr := make(chan error, 1)
	go func() {
		extractErr <- c.extractTarGz(file, outputDir)
	}()

	select {
	case err := <-extractErr:
		if err != nil {
			return fmt.Errorf("failed to extract tar.gz blob %s: %w", blobPath, err)
		}
	case <-ctx.Done():
		return fmt.Errorf("timeout while extracting blob %s", blobPath)
	}

	return nil
}
