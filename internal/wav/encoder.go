package wav

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattetti/e-mu-soundbanks/internal/ebl"
)

// Encoder handles encoding EBL audio data to WAV format
type Encoder struct {
	debug            bool
	noWrite          bool
	preserveFilename bool
	exbName          string // Name of the EXB file, used as a prefix for WAV filenames
}

// NewEncoder creates a new WAV encoder
func NewEncoder(debug, noWrite, preserveFilename bool, exbName string) *Encoder {
	return &Encoder{
		debug:            debug,
		noWrite:          noWrite,
		preserveFilename: preserveFilename,
		exbName:          exbName,
	}
}

// Debug logs a message if debug mode is enabled
func (e *Encoder) Debug(message string) {
	if e.debug {
		fmt.Println(message)
	}
}

// WriteWAV writes the EBL audio data to a WAV file
func (e *Encoder) WriteWAV(eblFile *ebl.EBLFile, outputDir string) (string, error) {
	// Constants
	wavHeaderLength := uint32(16) // Standard PCM header length
	wavPCMMode := uint16(1)       // PCM format
	wavBPS := uint16(16)          // 16 bits per sample

	// Determine channels (mono or stereo)
	numChannels := uint16(2) // default to stereo
	if eblFile.Channel2Size == 0 {
		numChannels = 1 // mono
	}

	// Get sample rate from EBL file
	sampleRate := uint32(eblFile.HeaderData.SampleRate)

	// Calculate derived fields
	byteRate := sampleRate * uint32(numChannels) * uint32(wavBPS) / 8
	blockAlign := numChannels * wavBPS / 8

	// Prepare audio data
	var wavData []byte
	var dataSize uint32

	if numChannels == 1 {
		// Mono: just use channel 1 data
		wavData = eblFile.Channel1Data
		dataSize = uint32(len(wavData))
	} else {
		// Stereo: interleave the channels (LRLRLR...)
		wavData = interleaveChannels(eblFile.Channel1Data, eblFile.Channel2Data)
		dataSize = uint32(len(wavData))
	}

	// Calculate file size
	fileSize := 36 + dataSize // 4 + (8 + 16) + (8 + DataSize)

	// Determine output filename
	var outputFilename string
	var baseName string

	if e.preserveFilename {
		baseName = strings.TrimSuffix(eblFile.Filename, ".ebl")
	} else {
		// Use the decoded UTF-16 filename from header
		baseName = cleanFilename(eblFile.HeaderData.FilenameStr)
		if baseName == "" {
			// Fallback to Header3 filename if HeaderData filename is empty
			baseName = cleanFilename(eblFile.Header3.Filename)
		}
		if baseName == "" {
			// Ultimate fallback: use the original filename
			baseName = strings.TrimSuffix(eblFile.Filename, ".ebl")
		}
	}

	// Add the EXB prefix if available
	if e.exbName != "" {
		outputFilename = fmt.Sprintf("%s - %s.wav", e.exbName, baseName)
	} else {
		outputFilename = baseName + ".wav"
	}

	outputPath := filepath.Join(outputDir, outputFilename)

	// If we're in no-write mode, just return
	if e.noWrite {
		return outputFilename, nil
	}

	// Create the WAV file
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("error creating output file: %w", err)
	}
	defer file.Close()

	// Write WAV header
	header := WAVHeader{
		RiffID:        [4]byte{'R', 'I', 'F', 'F'},
		FileSize:      fileSize,
		WaveID:        [4]byte{'W', 'A', 'V', 'E'},
		FmtID:         [4]byte{'f', 'm', 't', ' '},
		FmtSize:       wavHeaderLength,
		AudioFormat:   wavPCMMode,
		NumChannels:   numChannels,
		SampleRate:    sampleRate,
		ByteRate:      byteRate,
		BlockAlign:    blockAlign,
		BitsPerSample: wavBPS,
		DataID:        [4]byte{'d', 'a', 't', 'a'},
		DataSize:      dataSize,
	}

	// Write header
	if err := binary.Write(file, binary.LittleEndian, &header); err != nil {
		return "", fmt.Errorf("error writing WAV header: %w", err)
	}

	// Write audio data
	if _, err := file.Write(wavData); err != nil {
		return "", fmt.Errorf("error writing audio data: %w", err)
	}

	return outputFilename, nil
}

// interleaveChannels interleaves the left and right channel data for stereo WAV
// EBL format stores channels as LLLL...RRRR... but WAV needs LRLRLR...
func interleaveChannels(channel1, channel2 []byte) []byte {
	// For 16-bit samples, we need to work with 2 bytes at a time
	numSamples := len(channel1) / 2
	result := make([]byte, numSamples*4) // 2 channels * 2 bytes per sample

	for i := 0; i < numSamples; i++ {
		// Left channel (2 bytes)
		result[i*4] = channel1[i*2]
		result[i*4+1] = channel1[i*2+1]

		// Right channel (2 bytes)
		if i*2+1 < len(channel2) {
			result[i*4+2] = channel2[i*2]
			result[i*4+3] = channel2[i*2+1]
		}
	}

	return result
}

// cleanFilename removes invalid characters from a filename (Windows-safe)
func cleanFilename(filename string) string {
	// Replace non-alphanumeric characters (except specific ones) with underscores
	re := regexp.MustCompile(`[^0-9a-zA-Z\.,:%\-_#]+`)
	return re.ReplaceAllString(filename, "_")
}
