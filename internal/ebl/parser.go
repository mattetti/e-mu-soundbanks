package ebl

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// Parser handles reading and parsing EBL files
type Parser struct {
	debug     bool
	errorSave bool
}

// NewParser creates a new EBL parser
func NewParser(debug, errorSave bool) *Parser {
	return &Parser{
		debug:     debug,
		errorSave: errorSave,
	}
}

// Debug logs a message if debug mode is enabled
func (p *Parser) Debug(message string) {
	if p.debug {
		fmt.Println(message)
	}
}

// dumpHex returns a hexadecimal dump of the provided data
func (p *Parser) dumpHex(data []byte, maxLen int) string {
	if len(data) > maxLen {
		data = data[:maxLen]
	}
	return hex.Dump(data)
}

// decodeUTF16 decodes UTF-16 little endian bytes to a string, removing trailing nulls
func (p *Parser) decodeUTF16(b []byte) string {
	// Make sure we have an even number of bytes
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}

	// Convert bytes to uint16s
	u16s := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		u16s = append(u16s, binary.LittleEndian.Uint16(b[i:i+2]))
	}

	// Convert uint16s to runes
	runes := utf16.Decode(u16s)

	// Create string and trim null terminators
	return strings.TrimRight(string(runes), "\x00")
}

// ReadFile reads and parses an EBL file
func (p *Parser) ReadFile(inputFile string, errorDir string) (*EBLFile, error) {
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}

	eblFile := &EBLFile{
		Filename: filepath.Base(inputFile),
		Path:     inputFile,
		Size:     fileInfo.Size(),
		Read:     0,
	}

	// Read Header 1 (8 bytes)
	prefix := make([]byte, 4)
	if _, err := file.Read(prefix); err != nil {
		return nil, fmt.Errorf("error reading prefix: %w", err)
	}

	if p.debug {
		p.Debug(fmt.Sprintf("Header 1 prefix: %s (hex: %x)", string(prefix), prefix))
	}

	if string(prefix) != "FORM" {
		if p.debug {
			p.Debug(fmt.Sprintf("Invalid Header 1 prefix. Expected 'FORM', got:\n%s", p.dumpHex(prefix, 4)))
		}
		return nil, fmt.Errorf("invalid EBL file: expected FORM prefix, got %s (hex: %x)", string(prefix), prefix)
	}

	var filesize uint32
	if err := binary.Read(file, binary.BigEndian, &filesize); err != nil {
		return nil, fmt.Errorf("error reading file size: %w", err)
	}

	eblFile.Header1 = Header1{
		Prefix:   prefix,
		FileSize: int(filesize),
		Read:     8,
	}
	eblFile.Read = 8

	if p.debug {
		p.Debug(fmt.Sprintf("Header 1 filesize: %d", filesize))
	}

	// Read Header 2 (12 bytes)
	prefix2 := make([]byte, 8)
	if _, err := file.Read(prefix2); err != nil {
		return nil, fmt.Errorf("error reading header 2 prefix: %w", err)
	}

	if p.debug {
		p.Debug(fmt.Sprintf("Header 2 prefix: %s (hex: %x)", string(prefix2), prefix2))
	}

	if string(prefix2) != "E5B0TOC2" {
		if p.debug {
			p.Debug(fmt.Sprintf("Invalid Header 2 prefix. Expected 'E5B0TOC2', got:\n%s", p.dumpHex(prefix2, 8)))
		}
		return nil, fmt.Errorf("invalid EBL file: expected E5B0TOC2 prefix, got %s (hex: %x)", string(prefix2), prefix2)
	}

	var nextHeaderBytes uint32
	if err := binary.Read(file, binary.BigEndian, &nextHeaderBytes); err != nil {
		return nil, fmt.Errorf("error reading next header bytes: %w", err)
	}

	if p.debug {
		p.Debug(fmt.Sprintf("Header 2 nextHeaderBytes: %d", nextHeaderBytes))
	}

	eblFile.Header2 = Header2{
		Prefix:          prefix2,
		NextHeaderBytes: int(nextHeaderBytes),
		Read:            12,
	}
	eblFile.Read += 12

	// Read Header 3 (78 bytes)
	prefix3 := make([]byte, 4)
	if _, err := file.Read(prefix3); err != nil {
		return nil, fmt.Errorf("error reading header 3 prefix: %w", err)
	}

	if p.debug {
		p.Debug(fmt.Sprintf("Header 3 prefix: %s (hex: %x)", string(prefix3), prefix3))
	}

	if string(prefix3) != "E5S1" {
		if p.debug {
			p.Debug(fmt.Sprintf("Invalid Header 3 prefix. Expected 'E5S1', got:\n%s", p.dumpHex(prefix3, 4)))
		}
		return nil, fmt.Errorf("invalid EBL file: expected E5S1 prefix, got %s (hex: %x)", string(prefix3), prefix3)
	}

	var dataSize, data uint32
	if err := binary.Read(file, binary.BigEndian, &dataSize); err != nil {
		return nil, fmt.Errorf("error reading data size: %w", err)
	}
	if err := binary.Read(file, binary.BigEndian, &data); err != nil {
		return nil, fmt.Errorf("error reading data: %w", err)
	}

	if p.debug {
		p.Debug(fmt.Sprintf("Header 3 dataSize: %d, data: %d", dataSize, data))
	}

	zeros := make([]byte, 2)
	if _, err := file.Read(zeros); err != nil {
		return nil, fmt.Errorf("error reading zeros: %w", err)
	}

	filenameBytes := make([]byte, 64)
	if _, err := file.Read(filenameBytes); err != nil {
		return nil, fmt.Errorf("error reading filename: %w", err)
	}

	// Decode filename using UTF-16LE instead of UTF-8
	filename := p.decodeUTF16(filenameBytes)

	eblFile.Header3 = Header3{
		Prefix:   prefix3,
		DataSize: int(dataSize),
		Data:     int(data),
		Zeros:    zeros,
		Filename: filename,
		Read:     74,
	}
	eblFile.Read += 74

	if p.debug {
		p.Debug(fmt.Sprintf("Header 3 filename: %s", filename))
	}

	// Handle padding if necessary
	header3Padding := eblFile.Header3.Data - int(eblFile.Read)

	// Initialize our flags for header 4 detection
	foundHeader4InPadding := false
	var header4PrefixFromPadding []byte
	var header4SizeBytes []byte

	if header3Padding > 0 {
		eblFile.Padding = header3Padding
		padding := make([]byte, header3Padding)

		if p.debug {
			p.Debug(fmt.Sprintf("Reading %d bytes of padding after Header 3", header3Padding))
		}

		bytesRead, err := file.Read(padding)
		if err != nil {
			if p.debug {
				p.Debug(fmt.Sprintf("Error reading padding: %v (read %d of %d bytes)", err, bytesRead, header3Padding))
				if bytesRead > 0 {
					p.Debug(fmt.Sprintf("Partial padding data:\n%s", p.dumpHex(padding[:bytesRead], bytesRead)))
				}
			}
			return nil, fmt.Errorf("error reading padding: %w (read %d of %d bytes)", err, bytesRead, header3Padding)
		}

		eblFile.Read += int64(header3Padding)

		if p.debug && bytesRead > 0 {
			p.Debug(fmt.Sprintf("Padding data (first 16 bytes if available):\n%s", p.dumpHex(padding, 16)))
		}

		// Check if the padding already contains the Header 4 prefix
		if bytesRead >= 4 && string(padding[:4]) == "E5S1" {
			if p.debug {
				p.Debug("Found Header 4 prefix (E5S1) in the padding bytes")
			}
			foundHeader4InPadding = true
			header4PrefixFromPadding = padding[:4]

			// If we have enough bytes, also grab the size field
			if bytesRead >= 8 {
				header4SizeBytes = padding[4:8]
				if p.debug {
					p.Debug(fmt.Sprintf("Also found Header 4 size bytes in padding: %x", header4SizeBytes))
				}
			}
		}
	} else {
		eblFile.Padding = 0
		if p.debug {
			p.Debug(fmt.Sprintf("No padding needed (header3Padding = %d)", header3Padding))
		}
	}

	var prefix4 []byte
	var size uint32

	// If we already found the header 4 prefix in the padding, use that
	if foundHeader4InPadding {
		prefix4 = header4PrefixFromPadding

		// If we have the size bytes, use them
		if header4SizeBytes != nil {
			size = binary.BigEndian.Uint32(header4SizeBytes)
		} else {
			// Otherwise we need to read the size
			if err := binary.Read(file, binary.BigEndian, &size); err != nil {
				return nil, fmt.Errorf("error reading size after found header 4 prefix: %w", err)
			}
		}

		if p.debug {
			p.Debug(fmt.Sprintf("Using Header 4 prefix from padding: %s (hex: %x)", string(prefix4), prefix4))
			p.Debug(fmt.Sprintf("Header 4 size: %d", size))
		}
	} else {
		// Read Header 4 (14 bytes) - normal flow
		prefix4 = make([]byte, 4)
		bytesRead, err := file.Read(prefix4)
		if err != nil {
			if p.debug {
				p.Debug(fmt.Sprintf("Error reading header 4 prefix: %v (read %d of 4 bytes)", err, bytesRead))
				if bytesRead > 0 {
					p.Debug(fmt.Sprintf("Partial header 4 prefix:\n%s", p.dumpHex(prefix4[:bytesRead], bytesRead)))
				}
			}
			return nil, fmt.Errorf("error reading header 4 prefix: %w (read %d of 4 bytes)", err, bytesRead)
		}

		if p.debug {
			p.Debug(fmt.Sprintf("Header 4 prefix: %s (hex: %x)", string(prefix4), prefix4))
		}

		if string(prefix4) != "E5S1" {
			if p.debug {
				// Try to examine what's next in the file to aid in debugging
				remainingBytes := make([]byte, 20)
				remainingBytesRead, _ := file.Read(remainingBytes)

				p.Debug(fmt.Sprintf("Invalid Header 4 prefix. Expected 'E5S1', got '%s' (hex: %x)", string(prefix4), prefix4))
				p.Debug(fmt.Sprintf("Next %d bytes after invalid Header 4 prefix:\n%s",
					remainingBytesRead, p.dumpHex(remainingBytes[:remainingBytesRead], remainingBytesRead)))
			}
			return nil, fmt.Errorf("invalid EBL file: expected E5S1 prefix, got %s (hex: %x)", string(prefix4), prefix4)
		}

		if err := binary.Read(file, binary.BigEndian, &size); err != nil {
			return nil, fmt.Errorf("error reading size: %w", err)
		}

		if p.debug {
			p.Debug(fmt.Sprintf("Header 4 size: %d", size))
		}
	}

	data4 := make([]byte, 6)
	if _, err := file.Read(data4); err != nil {
		return nil, fmt.Errorf("error reading data: %w", err)
	}

	if p.debug {
		p.Debug(fmt.Sprintf("Header 4 data: %x", data4))
	}

	// If we found Header 4 in padding, we didn't actually read the prefix and size from the file
	header4ReadBytes := 14
	if foundHeader4InPadding {
		if header4SizeBytes != nil {
			header4ReadBytes = 6 // We only read the 6 bytes of data4
		} else {
			header4ReadBytes = 10 // We read 4 bytes of size + 6 bytes of data4
		}
	}

	eblFile.Header4 = Header4{
		Prefix: prefix4,
		Size:   int(size),
		Data:   data4,
		Read:   int64(header4ReadBytes),
	}
	eblFile.Read += int64(header4ReadBytes)

	// Read Header Data
	filenameBytes2 := make([]byte, 64)
	if _, err := file.Read(filenameBytes2); err != nil {
		return nil, fmt.Errorf("error reading filename: %w", err)
	}

	// This second filename should also be decoded as UTF-16LE
	filename2 := p.decodeUTF16(filenameBytes2)

	if p.debug && filename != filename2 {
		p.Debug(fmt.Sprintf("Filename mismatch: Header3=%s, HeaderData=%s", filename, filename2))
	}

	var v1, v2, v3, v4, v5, v6, v7, v8, v9, sampleRate, v11, v12 uint32
	if err := binary.Read(file, binary.LittleEndian, &v1); err != nil {
		return nil, fmt.Errorf("error reading v1: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v2); err != nil {
		return nil, fmt.Errorf("error reading v2: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v3); err != nil {
		return nil, fmt.Errorf("error reading v3: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v4); err != nil {
		return nil, fmt.Errorf("error reading v4: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v5); err != nil {
		return nil, fmt.Errorf("error reading v5: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v6); err != nil {
		return nil, fmt.Errorf("error reading v6: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v7); err != nil {
		return nil, fmt.Errorf("error reading v7: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v8); err != nil {
		return nil, fmt.Errorf("error reading v8: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v9); err != nil {
		return nil, fmt.Errorf("error reading v9: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &sampleRate); err != nil {
		return nil, fmt.Errorf("error reading frequency: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v11); err != nil {
		return nil, fmt.Errorf("error reading v11: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &v12); err != nil {
		return nil, fmt.Errorf("error reading v12: %w", err)
	}

	comment := make([]byte, 64)
	if _, err := file.Read(comment); err != nil {
		return nil, fmt.Errorf("error reading comment: %w", err)
	}

	// Also decode the comment as UTF-16LE
	commentStr := p.decodeUTF16(comment)

	if p.debug {
		p.Debug(fmt.Sprintf("HeaderData values: v1=%d, v2=%d, v3=%d, v4=%d, v5=%d, frequency=%d",
			v1, v2, v3, v4, v5, sampleRate))
		if commentStr != "" {
			p.Debug(fmt.Sprintf("Comment: %s", commentStr))
		}
	}

	eblFile.HeaderData = HeaderData{
		Filename:    filenameBytes2, // Keep the raw bytes
		FilenameStr: filename2,      // Store the decoded string
		V1:          int(v1),
		V2:          int(v2),
		V3:          int(v3),
		V4:          int(v4),
		V5:          int(v5),
		V6:          int(v6),
		V7:          int(v7),
		V8:          int(v8),
		V9:          int(v9),
		SampleRate:  int(sampleRate),
		V11:         int(v11),
		V12:         int(v12),
		Comment:     comment,    // Keep the raw bytes
		CommentStr:  commentStr, // Store the decoded string
		Read:        176,
	}
	eblFile.Read += 176

	// Calculate channel sizes
	eblFile.Channel1Size = eblFile.HeaderData.V3 - eblFile.HeaderData.V2
	eblFile.Channel2Size = eblFile.HeaderData.V5 - eblFile.HeaderData.V4

	if p.debug {
		p.Debug(fmt.Sprintf("Channel sizes: Channel1=%d, Channel2=%d",
			eblFile.Channel1Size, eblFile.Channel2Size))
	}

	if eblFile.Channel1Size == eblFile.Channel2Size {
		p.Debug("Channels same Length")
		if eblFile.Channel1Size == 0 {
			p.Debug("MONO DETECTED")
			eblFile.Channel1Size = eblFile.HeaderData.V4 - eblFile.HeaderData.V3 + 2
			eblFile.Channel2Size = 0
			p.Debug(fmt.Sprintf("Updated Channel1 size for mono: %d", eblFile.Channel1Size))
		}
	} else {
		if eblFile.Channel1Size*eblFile.Channel2Size != 0 {
			p.Debug(fmt.Sprintf("Error: Channels Different length. C1: %d, C2: %d", eblFile.Channel1Size, eblFile.Channel2Size))
		}
	}

	eblFile.DataSizeCalc = eblFile.Channel1Size + eblFile.Channel2Size

	// Handle data padding
	dataPadding := eblFile.HeaderData.V5 - eblFile.DataSizeCalc - 178
	if dataPadding > 0 {
		p.Debug(fmt.Sprintf("Reading %d bytes of data padding", dataPadding))
		padding := make([]byte, dataPadding)
		if _, err := file.Read(padding); err != nil {
			return nil, fmt.Errorf("error reading data padding: %w", err)
		}
		eblFile.Read += int64(dataPadding)
	}

	eblFile.HeaderRead = eblFile.Read
	eblFile.DataSizeEst = eblFile.Size - eblFile.HeaderRead

	if p.debug {
		p.Debug(fmt.Sprintf("About to read audio data: Channel1Size=%d, Channel2Size=%d",
			eblFile.Channel1Size, eblFile.Channel2Size))
	}

	// Read audio data
	eblFile.Channel1Data = make([]byte, eblFile.Channel1Size)
	bytesRead, err := io.ReadFull(file, eblFile.Channel1Data)
	if err != nil {
		if p.debug {
			p.Debug(fmt.Sprintf("Error reading channel 1 data: %v (read %d of %d bytes)",
				err, bytesRead, eblFile.Channel1Size))
		}
		return nil, fmt.Errorf("error reading channel 1 data: %w (read %d of %d bytes)",
			err, bytesRead, eblFile.Channel1Size)
	}
	eblFile.Read += int64(eblFile.Channel1Size)

	eblFile.Channel2Data = make([]byte, eblFile.Channel2Size)
	bytesRead, err = io.ReadFull(file, eblFile.Channel2Data)
	if err != nil {
		if p.debug {
			p.Debug(fmt.Sprintf("Error reading channel 2 data: %v (read %d of %d bytes)",
				err, bytesRead, eblFile.Channel2Size))
		}
		return nil, fmt.Errorf("error reading channel 2 data: %w (read %d of %d bytes)",
			err, bytesRead, eblFile.Channel2Size)
	}
	eblFile.Read += int64(eblFile.Channel2Size)

	// Check if we've reached the end of the file
	endOfData := eblFile.Read
	if endOfData != eblFile.Size {
		difference := eblFile.Size - endOfData

		// Many files have a 4-byte trailer at the end
		if difference == 4 {
			trailer := make([]byte, 4)
			bytesRead, err := file.Read(trailer)
			if err == nil && bytesRead == 4 {
				eblFile.Read += 4
				if p.debug {
					p.Debug(fmt.Sprintf("Read 4-byte trailer: %x", trailer))
				}
				// File is now fully read
				return eblFile, nil
			}
		}

		// Only show as an error if it's not a 40-byte or 4-byte difference
		if difference != 40 && difference != 4 {
			p.Debug(fmt.Sprintf("ERROR: Inconsistent filesize: Read: %d, Expected: %d, Difference: %d",
				endOfData, eblFile.Size, difference))
		} else if difference == 40 {
			p.Debug("WARN: Found 40 bytes. Additional data header.")
		}
	}

	return eblFile, nil
}
