package flac

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Converter handles converting WAV files to FLAC
type Converter struct {
	ffmpegPath string
	debug      bool
	maxWorkers int
}

// NewConverter creates a new FLAC converter
func NewConverter(debug bool) (*Converter, error) {
	// Find ffmpeg in the system
	ffmpegPath, err := findFFmpeg()
	if err != nil {
		return nil, err
	}

	// Determine the number of workers based on available CPU cores
	// Use 75% of available cores (minimum 2, maximum 12)
	numCPU := runtime.NumCPU()
	maxWorkers := max(2, min(numCPU*3/4, 12))

	return &Converter{
		ffmpegPath: ffmpegPath,
		debug:      debug,
		maxWorkers: maxWorkers,
	}, nil
}

// Helper functions for min/max operations
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// findFFmpeg locates the ffmpeg binary on the system
func findFFmpeg() (string, error) {
	// Try to find ffmpeg in PATH
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("where", "ffmpeg")
	} else {
		cmd = exec.Command("which", "ffmpeg")
	}

	output, err := cmd.Output()
	if err == nil {
		// Found ffmpeg in PATH
		return strings.TrimSpace(string(output)), nil
	}

	// Check common installation locations based on OS
	commonPaths := []string{}

	if runtime.GOOS == "windows" {
		commonPaths = []string{
			`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
			`C:\Program Files (x86)\ffmpeg\bin\ffmpeg.exe`,
		}
	} else if runtime.GOOS == "darwin" {
		commonPaths = []string{
			"/usr/local/bin/ffmpeg",
			"/opt/homebrew/bin/ffmpeg",
			"/opt/local/bin/ffmpeg",
		}
	} else {
		// Linux/Unix
		commonPaths = []string{
			"/usr/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/opt/ffmpeg/bin/ffmpeg",
		}
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("ffmpeg not found. Please install ffmpeg to use the FLAC conversion feature")
}

// ConvertToFlac converts a WAV file to FLAC format
func (c *Converter) ConvertToFlac(wavFile string) error {
	// Check if input file exists
	if _, err := os.Stat(wavFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", wavFile)
	}

	// Create output filename
	flacFile := strings.TrimSuffix(wavFile, ".wav") + ".flac"

	// Build ffmpeg command with appropriate options
	cmd := exec.Command(
		c.ffmpegPath,
		"-i", wavFile, // Input file
		"-c:a", "flac", // Use FLAC codec
		"-compression_level", "8", // Maximum compression
		"-y",     // Overwrite output file if it exists
		flacFile, // Output file
	)

	// If debug mode is on, show the ffmpeg output
	if c.debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("Running: %s\n", cmd.String())
	} else {
		// Suppress ffmpeg output otherwise
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error converting to FLAC: %w", err)
	}

	// Delete the original WAV file
	err = os.Remove(wavFile)
	if err != nil {
		return fmt.Errorf("error removing original WAV file: %w", err)
	}

	return nil
}

// ConvertDirectory converts all WAV files in a directory to FLAC using multiple workers
func (c *Converter) ConvertDirectory(dir string) error {
	// Find all WAV files in the directory and subdirectories
	var wavFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".wav" {
			wavFiles = append(wavFiles, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error finding WAV files: %w", err)
	}

	if len(wavFiles) == 0 {
		// No WAV files found
		return nil
	}

	// Create a channel to send jobs to workers
	jobs := make(chan string, len(wavFiles))

	// Create a channel to receive results
	results := make(chan error, len(wavFiles))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Determine number of workers
	numWorkers := min(c.maxWorkers, len(wavFiles))

	// Print info about parallelization
	if c.debug {
		fmt.Printf("Converting %d WAV files to FLAC using %d parallel workers\n",
			len(wavFiles), numWorkers)
	}

	// Start workers
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Process jobs until the channel is closed
			for wavFile := range jobs {
				if c.debug {
					fmt.Printf("Worker %d: Converting %s to FLAC\n", id, wavFile)
				}

				err := c.ConvertToFlac(wavFile)
				results <- err

				if err != nil {
					if c.debug {
						fmt.Printf("Worker %d: Error converting %s: %v\n", id, wavFile, err)
					}
				} else if c.debug {
					fmt.Printf("Worker %d: Successfully converted %s\n", id, wavFile)
				}
			}
		}(w)
	}

	// Send jobs to workers
	for _, wavFile := range wavFiles {
		jobs <- wavFile
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect errors
	var errorCount int
	for err := range results {
		if err != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("%d files failed to convert", errorCount)
	}

	return nil
}
