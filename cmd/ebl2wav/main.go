package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mattetti/e-mu-soundbanks/internal/converter"
	"github.com/mattetti/e-mu-soundbanks/internal/flac"
)

var (
	inputPath  string
	outputPath string
	exbPath    string
	exbDirPath string
	debugMode  bool
	errorSave  bool
	flacMode   bool
	version    bool
)

func init() {
	flag.StringVar(&inputPath, "i", "", "Input file or directory (required if not using -exb or -exbdir)")
	flag.StringVar(&outputPath, "o", "", "Output directory (defaults to \"E-MU Sounds\")")
	flag.StringVar(&exbPath, "exb", "", "Path to an .exb file. Will process related .ebl files in SamplePool folder")
	flag.StringVar(&exbDirPath, "exbdir", "", "Path to a directory containing .exb files (will process recursively)")
	flag.BoolVar(&debugMode, "d", false, "Debug mode")
	flag.BoolVar(&errorSave, "e", false, "Save files with errors to output/errors/")
	flag.BoolVar(&flacMode, "flac", false, "Convert output to FLAC format (requires ffmpeg)")
	flag.BoolVar(&version, "version", false, "Display version information")
}

const VERSION = "1.0.0"

func main() {
	flag.Parse()

	// Display version if requested
	if version {
		fmt.Printf("ebl2wav version %s\n", VERSION)
		os.Exit(0)
	}

	// Process directory of EXB files if provided
	if exbDirPath != "" {
		processExbDirectory(exbDirPath)
		return
	}

	// Process EXB file if provided
	if exbPath != "" {
		if filepath.Ext(exbPath) != ".exb" {
			fmt.Println("Error: EXB path must point to an .exb file")
			os.Exit(1)
		}

		// Process the EXB file
		processExbFile(exbPath)
		return
	}

	// Check for required input path if not using EXB mode
	if inputPath == "" {
		fmt.Println("Error: Input path is required. Use -i flag, provide an .exb file with -exb, or specify a directory of EXB files with -exbdir.")
		printUsage()
		os.Exit(1)
	}

	// Set default output path if not provided
	if outputPath == "" {
		outputPath = "E-MU Sounds"
		fmt.Printf("No output directory selected - Defaulting to %s\n", outputPath)
	}

	// Create converter with options
	conv := converter.NewConverter(converter.Options{
		Debug:            debugMode,
		NoWrite:          false,
		PreserveFilename: false,
		ErrorSave:        errorSave,
		ExbName:          "", // No EXB name when using -i flag
	})

	// Process input path
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if errorSave {
		errorDir := filepath.Join(outputPath, "errors")
		if err := os.MkdirAll(errorDir, 0755); err != nil {
			fmt.Printf("Error creating error directory: %v\n", err)
			os.Exit(1)
		}
	}

	if debugMode {
		fmt.Printf("DEBUG MODE: %t, ERROR SAVE: %t, FLAC MODE: %t\n", debugMode, errorSave, flacMode)
		fmt.Printf("Using %d CPU cores\n", runtime.NumCPU())
	}

	// Convert EBL to WAV
	if inputInfo.IsDir() {
		// Process directory
		err = conv.ProcessDirectory(inputPath, outputPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Process single file
		if filepath.Ext(inputPath) != ".ebl" {
			fmt.Println("Input file must be an EBL file.")
			os.Exit(1)
		}
		success, err := conv.ConvertFile(inputPath, outputPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if success {
			fmt.Printf("Converted %s\n", filepath.Base(inputPath))
		} else {
			fmt.Printf("Failed to convert %s\n", filepath.Base(inputPath))
		}
	}

	// Convert WAV to FLAC if requested
	if flacMode {
		convertToFlac(outputPath)
	}
}

// processExbDirectory processes all EXB files in a directory and its subdirectories
func processExbDirectory(exbDirPath string) {
	// Verify the directory exists
	dirInfo, err := os.Stat(exbDirPath)
	if err != nil {
		fmt.Printf("Error accessing directory: %v\n", err)
		os.Exit(1)
	}

	if !dirInfo.IsDir() {
		fmt.Printf("Error: %s is not a directory\n", exbDirPath)
		os.Exit(1)
	}

	fmt.Printf("Scanning %s for EXB files...\n", exbDirPath)

	// Find all EXB files recursively
	var exbFiles []string
	err = filepath.Walk(exbDirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".exb" {
			exbFiles = append(exbFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning for EXB files: %v\n", err)
		os.Exit(1)
	}

	if len(exbFiles) == 0 {
		fmt.Println("No EXB files found.")
		os.Exit(1)
	}

	fmt.Printf("Found %d EXB files to process.\n", len(exbFiles))

	// Process each EXB file
	for i, exbFile := range exbFiles {
		fmt.Printf("[%d/%d] Processing %s\n", i+1, len(exbFiles), exbFile)

		// Save the current output path
		originalOutputPath := outputPath

		// Process the EXB file
		exbPath = exbFile
		processExbFile(exbFile)

		// Restore the output path
		outputPath = originalOutputPath
	}

	fmt.Printf("Successfully processed %d EXB files.\n", len(exbFiles))
}

// processExbFile processes an EXB file and its associated SamplePool folder
func processExbFile(exbPath string) {
	// Extract the base name without the .exb extension to use as prefix
	baseExbName := filepath.Base(exbPath)
	baseExbName = strings.TrimSuffix(baseExbName, filepath.Ext(baseExbName))

	// Check if SamplePool directory exists
	exbDir := filepath.Dir(exbPath)
	samplePoolDir := filepath.Join(exbDir, "SamplePool")

	if _, err := os.Stat(samplePoolDir); os.IsNotExist(err) {
		fmt.Printf("Error: SamplePool directory not found at %s\n", samplePoolDir)
		// Don't exit when processing multiple EXB files
		if exbDirPath == "" {
			os.Exit(1)
		} else {
			fmt.Println("Skipping this EXB file.")
			return
		}
	}

	// Set default output path if not provided
	thisOutputPath := outputPath
	if thisOutputPath == "" {
		// If processing a directory of EXB files, use a subdirectory structure that mirrors the input
		if exbDirPath != "" {
			relPath, err := filepath.Rel(exbDirPath, exbDir)
			if err == nil && relPath != "." {
				thisOutputPath = filepath.Join("E-MU Sounds", relPath, baseExbName)
			} else {
				thisOutputPath = filepath.Join("E-MU Sounds", baseExbName)
			}
		} else {
			thisOutputPath = filepath.Join("E-MU Sounds", baseExbName)
		}
		fmt.Printf("No output directory selected - Defaulting to %s\n", thisOutputPath)
	}

	// Create output directory
	if err := os.MkdirAll(thisOutputPath, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		if exbDirPath == "" {
			os.Exit(1)
		} else {
			fmt.Println("Skipping this EXB file.")
			return
		}
	}

	// Create error directory if needed
	if errorSave {
		errorDir := filepath.Join(thisOutputPath, "errors")
		if err := os.MkdirAll(errorDir, 0755); err != nil {
			fmt.Printf("Error creating error directory: %v\n", err)
			if exbDirPath == "" {
				os.Exit(1)
			} else {
				fmt.Println("Continuing without error directory.")
			}
		}
	}

	// Create converter with options, including the EXB name
	conv := converter.NewConverter(converter.Options{
		Debug:            debugMode,
		NoWrite:          false,
		PreserveFilename: false,
		ErrorSave:        errorSave,
		ExbName:          baseExbName, // Use the EXB name for prefixing WAV files
	})

	// Find all .ebl files in the SamplePool directory
	fmt.Printf("Processing EXB file: %s\n", baseExbName)
	fmt.Printf("Scanning %s for .ebl files...\n", samplePoolDir)

	// Process the SamplePool directory
	err := conv.ProcessDirectory(samplePoolDir, thisOutputPath)
	if err != nil {
		fmt.Printf("Error processing SamplePool directory: %v\n", err)
		if exbDirPath == "" {
			os.Exit(1)
		} else {
			fmt.Println("Skipping to the next EXB file.")
			return
		}
	}

	// Convert WAV to FLAC if requested
	if flacMode {
		convertToFlac(thisOutputPath)
	}
}

// convertToFlac converts all WAV files in the output directory to FLAC
func convertToFlac(outputDir string) {
	// Initialize FLAC converter
	flacConverter, err := flac.NewConverter(debugMode)
	if err != nil {
		fmt.Printf("Error initializing FLAC converter: %v\n", err)
		fmt.Println("WAV files were not converted to FLAC.")
		return
	}

	fmt.Println("Converting WAV files to FLAC format (using parallel processing)...")
	startTime := time.Now()

	// Convert all WAV files in the output directory
	err = flacConverter.ConvertDirectory(outputDir)
	if err != nil {
		fmt.Printf("Error converting to FLAC: %v\n", err)
		fmt.Println("Some WAV files may not have been converted.")
		return
	}

	elapsed := time.Since(startTime)
	fmt.Printf("FLAC conversion completed successfully in %.2f seconds.\n", elapsed.Seconds())
}

func printUsage() {
	fmt.Println("Usage: ebl2wav -i <input> [options] or ebl2wav -exb <exbfile> [options] or ebl2wav -exbdir <directory> [options]")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  ebl2wav -i /path/to/input/                # Process directory of .ebl files")
	fmt.Println("  ebl2wav -i file.ebl -o .                  # Process single file")
	fmt.Println("  ebl2wav -exb Sample.exb                   # Process .ebl files in SamplePool folder")
	fmt.Println("  ebl2wav -exbdir /path/to/soundbanks/      # Process all .exb files recursively")
	fmt.Println("  ebl2wav -exbdir /path/to/soundbanks/ -flac # Convert all soundbanks to FLAC")
	fmt.Println("  ebl2wav -i /path/to/input/ -d -e          # Process with debug mode and error saving")
}
