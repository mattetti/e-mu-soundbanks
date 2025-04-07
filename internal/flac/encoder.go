package flac

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Converter handles converting WAV files to FLAC
type Converter struct {
	ffmpegPath string
	debug      bool
}

// NewConverter creates a new FLAC converter
func NewConverter(debug bool) (*Converter, error) {
	// Find ffmpeg in the system
	ffmpegPath, err := findFFmpeg()
	if err != nil {
		return nil, err
	}

	return &Converter{
		ffmpegPath: ffmpegPath,
		debug:      debug,
	}, nil
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

// ConvertDirectory converts all WAV files in a directory to FLAC
func (c *Converter) ConvertDirectory(dir string) error {
	// Find all WAV files in the directory
	wavFiles, err := filepath.Glob(filepath.Join(dir, "*.wav"))
	if err != nil {
		return fmt.Errorf("error finding WAV files: %w", err)
	}

	// Convert each WAV file to FLAC
	for _, wavFile := range wavFiles {
		if c.debug {
			fmt.Printf("Converting %s to FLAC\n", wavFile)
		}

		err := c.ConvertToFlac(wavFile)
		if err != nil {
			fmt.Printf("Error converting %s: %v\n", wavFile, err)
			// Continue with other files even if one fails
			continue
		}
	}

	// Process subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			err := c.ConvertDirectory(subdir)
			if err != nil {
				fmt.Printf("Error processing subdirectory %s: %v\n", subdir, err)
				// Continue with other directories even if one fails
				continue
			}
		}
	}

	return nil
}
