package wav

// WAVHeader represents the structure of a WAV file header
type WAVHeader struct {
	// RIFF header
	RiffID   [4]byte // "RIFF"
	FileSize uint32  // 4 + (8 + SubChunk1Size) + (8 + SubChunk2Size)
	WaveID   [4]byte // "WAVE"

	// fmt sub-chunk
	FmtID         [4]byte // "fmt "
	FmtSize       uint32  // 16 for PCM
	AudioFormat   uint16  // 1 for PCM
	NumChannels   uint16  // 1 for mono, 2 for stereo
	SampleRate    uint32  // e.g., 44100
	ByteRate      uint32  // SampleRate * NumChannels * BitsPerSample/8
	BlockAlign    uint16  // NumChannels * BitsPerSample/8
	BitsPerSample uint16  // 8, 16, etc.

	// data sub-chunk
	DataID   [4]byte // "data"
	DataSize uint32  // NumSamples * NumChannels * BitsPerSample/8
}
