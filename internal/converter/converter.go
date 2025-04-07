package converter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattetti/em-u-soundbanks/internal/ebl"
	"github.com/mattetti/em-u-soundbanks/internal/wav"
)

// Options represents the conversion options
type Options struct {
	Debug            bool
	NoWrite          bool
	PreserveFilename bool
	ErrorSave        bool
	ExbName          string // The name of the EXB file (for prefixing WAV files)
}

// Converter handles the conversion process
type Converter struct {
	options Options
	parser  *ebl.Parser
	encoder *wav.Encoder
}

// NewConverter creates a new converter
func NewConverter(options Options) *Converter {
	return &Converter{
		options: options,
		parser:  ebl.NewParser(options.Debug, options.ErrorSave),
		encoder: wav.NewEncoder(options.Debug, options.NoWrite, options.PreserveFilename, options.ExbName),
	}
}

// ConvertFile converts a single EBL file to WAV
func (c *Converter) ConvertFile(inputFile, outputDir string) (bool, error) {
	errorDir := filepath.Join(outputDir, "errors")

	// Parse EBL file
	eblFile, err := c.parser.ReadFile(inputFile, errorDir)
	if err != nil {
		fmt.Printf("EBL READ ERROR: %s\n", filepath.Base(inputFile))
		if c.options.ErrorSave {
			c.saveErrorFile(inputFile, errorDir)
		}
		return false, err
	}

	// Encode to WAV
	_, err = c.encoder.WriteWAV(eblFile, outputDir)
	if err != nil {
		fmt.Printf("WAV WRITE ERROR: %s\n", filepath.Base(inputFile))
		if c.options.ErrorSave {
			c.saveErrorFile(inputFile, errorDir)
		}
		return false, err
	}

	return true, nil
}

// ProcessDirectory processes all EBL files in a directory and its subdirectories
func (c *Converter) ProcessDirectory(inputDir, outputDir string) error {
	fmt.Printf("Scanning %s/ ...", inputDir)

	// Find all .ebl files recursively
	var files []string
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".ebl" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error scanning directory: %w", err)
	}

	fmt.Printf("Done.\nPlanning to process %d EBL files in %s/\n", len(files), inputDir)

	// Group files by directory
	dirMap := make(map[string][]string)
	for _, file := range files {
		relPath, err := filepath.Rel(inputDir, filepath.Dir(file))
		if err != nil {
			return fmt.Errorf("error calculating relative path: %w", err)
		}
		if relPath == "." {
			relPath = ""
		}
		if relPath == "" {
			relPath = "/"
		}
		dirMap[relPath] = append(dirMap[relPath], file)
	}

	// Process files by directory
	totalConverted := 0
	startTime := time.Now()

	for dir, dirFiles := range dirMap {
		fmt.Printf("%s - %d file(s).\n", dir, len(dirFiles))

		// Create output directory if necessary
		dirOutputPath := filepath.Join(outputDir, dir)
		if !c.options.NoWrite {
			if err := os.MkdirAll(dirOutputPath, 0755); err != nil {
				return fmt.Errorf("error creating output directory: %w", err)
			}
		}

		// Convert files
		converted := 0
		for _, file := range dirFiles {
			success, err := c.ConvertFile(file, dirOutputPath)
			if err != nil && c.options.Debug {
				fmt.Printf("Error converting %s: %v\n", file, err)
				continue
			}
			if success {
				converted++
				totalConverted++
			}
		}
		fmt.Printf("Converted %d files in folder.\n", converted)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Converted %d/%d files. Duration: %.2fs\n", totalConverted, len(files), elapsed.Seconds())

	return nil
}

// saveErrorFile saves a copy of a file that caused an error
func (c *Converter) saveErrorFile(inputFile, errorDir string) {
	// Only save if ErrorSave is enabled and NoWrite is disabled
	if !c.options.ErrorSave || c.options.NoWrite {
		return
	}

	// Create error directory if it doesn't exist
	if err := os.MkdirAll(errorDir, 0755); err != nil {
		fmt.Printf("Error creating error directory: %v\n", err)
		return
	}

	// Open input file
	inFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening file for error copy: %v\n", err)
		return
	}
	defer inFile.Close()

	// Create output file
	errorFilePath := filepath.Join(errorDir, filepath.Base(inputFile))
	outFile, err := os.Create(errorFilePath)
	if err != nil {
		fmt.Printf("Error creating error file: %v\n", err)
		return
	}
	defer outFile.Close()

	// Copy content
	if _, err := io.Copy(outFile, inFile); err != nil {
		fmt.Printf("Error copying file content: %v\n", err)
	}
}
