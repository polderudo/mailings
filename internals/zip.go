package internals

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"time"
)

// AppendBytesToZip appends byte data as a new file to an existing or new zip archive.
// If the zip file doesn't exist, it creates a new one.
func AppendBytesToZip(zipPath, fileName string, data []byte) error {
	// Check if the zip file exists.
	_, err := os.Stat(zipPath)
	if os.IsNotExist(err) { // Zip file doesn't exist, create a new one
		zipFile, err := os.Create(zipPath)
		if err != nil {
			return fmt.Errorf("error creating new zip file: %w", err)
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()
		return writeBytesToZip(zipWriter, fileName, data)

	} else if err != nil { // Some other error occurred
		return fmt.Errorf("error checking zip file status: %w", err)
	}

	// Zip file exists, append to it.
	tempZipFile, err := os.CreateTemp("", "temp_zip_*.zip") // Use a temp file
	if err != nil {
		return fmt.Errorf("error creating temp zip file: %w", err)
	}
	defer os.Remove(tempZipFile.Name()) // Ensure temp file closure

	newZipWriter := zip.NewWriter(tempZipFile)
	defer newZipWriter.Close()

	// Copy existing files from original zip to the new zip.
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("error opening existing zip: %w", err)
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		err := addFileToZip(newZipWriter, f) //addFileToZip function from previous examples
		if err != nil {
			return err
		}
	}

	// Add the new file with byte data.
	if err := writeBytesToZip(newZipWriter, fileName, data); err != nil {
		return err
	}

	if err := newZipWriter.Close(); err != nil {
		return err
	}

	tempZipFile.Close()

	if err := os.Rename(tempZipFile.Name(), zipPath); err != nil {
		return err
	}

	return nil

}

func writeBytesToZip(zipWriter *zip.Writer, fileName string, data []byte) error {

	header := &zip.FileHeader{
		Name:     fileName,
		Modified: time.Now(),
		Method:   zip.Deflate, // You can change the compression method if needed

	}

	writer, err := zipWriter.CreateHeader(header)

	if err != nil {
		return err
	}

	_, err = writer.Write(data)

	return err

}

func addFileToZip(zipWriter *zip.Writer, file *zip.File) error {

	reader, err := file.Open()
	if err != nil {
		return err
	}

	defer reader.Close()

	header, err := zip.FileInfoHeader(file.FileInfo())

	if err != nil {
		return err
	}

	header.Name = file.Name
	header.Method = file.Method

	writer, err := zipWriter.CreateHeader(header)

	if err != nil {
		return err
	}

	_, err = io.Copy(writer, reader)

	return err

}
