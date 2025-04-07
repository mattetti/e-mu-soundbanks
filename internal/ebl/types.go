package ebl

// EBLFile represents the structure of an EBL file
type EBLFile struct {
	Filename     string
	Path         string
	Size         int64
	Read         int64
	Header1      Header1
	Header2      Header2
	Header3      Header3
	Padding      int
	Header4      Header4
	HeaderData   HeaderData
	Channel1Size int
	Channel2Size int
	DataSizeCalc int
	HeaderRead   int64
	DataSizeEst  int64
	Channel1Data []byte
	Channel2Data []byte
}

// Header1 represents the first header section of an EBL file
type Header1 struct {
	Prefix   []byte // "FORM"
	FileSize int    // FileSize - 8 (how many bytes are left)
	Read     int64
}

// Header2 represents the second header section of an EBL file
type Header2 struct {
	Prefix          []byte // "E5B0TOC2"
	NextHeaderBytes int    // Length of the next Chunk (78)
	Read            int64
}

// Header3 represents the third header section of an EBL file
type Header3 struct {
	Prefix   []byte // "E5S1"
	DataSize int    // Size after "header_4_data" below, i.e byte >= 108
	Data     int    // 98
	Zeros    []byte // 2 bytes of zeros
	Filename string // Decoded UTF-16 string
	Read     int64
}

// Header4 represents the fourth header section of an EBL file
type Header4 struct {
	Prefix []byte // "E5S1"
	Size   int
	Data   []byte // 6 bytes
	Read   int64
}

// HeaderData represents the data chunk header
type HeaderData struct {
	Filename    []byte // 64 bytes of filename data (UTF-16LE)
	FilenameStr string // Decoded UTF-16 filename
	V1          int    // Unknown. 301 le
	V2          int    // Data Offset. 184 le
	V3          int    // Data size (including offset)
	V4          int    // Data size - 2
	V5          int    // Close to the end of file
	V6          int    // Channel 1 Data Offset
	V7          int    // Data size (including offset)
	V8          int    // Start of Audio Data?
	V9          int    // End of data for this channel?
	SampleRate  int    // typically 44100 Hz
	V11         int    // Unknown, 0
	V12         int    // Unknown
	Comment     []byte // 64 bytes of comment data (UTF-16LE)
	CommentStr  string // Decoded UTF-16 comment
	Read        int64
}
