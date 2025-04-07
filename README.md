# E-MU SoundBanks

Tools for recovering and using legacy E-mu sound samples and banks. This repository provides utilities to help musicians and producers access vintage E-MU sample libraries that might otherwise be inaccessible due to outdated file formats.

## EBL to WAV Converter

The first tool in this collection is an EBL to WAV converter that transforms proprietary E-mu Emulator X files (`.ebl`) to the standard WAV format. This makes these high-quality samples accessible in any modern DAW or sampler.

The [E-mu Emulator X](https://en.wikipedia.org/wiki/E-mu_Emulator_X) was a popular software sampler released in the early 2000s that used proprietary formats for storing sound data. Many sample libraries from this era are only available in these legacy formats.

### Finding Sample Libraries

Hundreds of E-MU sound banks are available for free on Archive.org, for instance:
- [E-mu EXB Sound Banks Collection](https://archive.org/details/emuexbsoundbanks)

These collections contain thousands of od school samples that can be converted with this tool.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mattetti/e-mu-soundbanks.git
cd e-MU-soundbanks

# Build the binary
go build ./cmd/ebl2wav

# Optional: Install to your GOPATH
go install ./cmd/ebl2wav
```

## Usage

The tool supports both individual file conversion and batch processing:

### Examples

Convert every `.ebl` file in `/path/to/input/` recursively. Outputs to `./E-MU Sounds/`:

```bash
ebl2wav -i /path/to/input/
```

Convert a single `.ebl` file:

```bash
ebl2wav -i file.ebl -o ./output
```

Process an `.exb` file and convert all `.ebl` files in its associated `SamplePool` directory:

```bash
ebl2wav -exb ./data/PROcussion/PROcussion.exb
```

Process with debug mode and error saving:

```bash
ebl2wav -i /path/to/input/ -d -e
```

### Command Line Options

- `-i`: Input file or directory. Required unless `-exb` is used.
- `-o`: Output Directory. Resultant output directory. Defaults to `./E-MU Sounds/`.
- `-exb`: Path to an .exb file. Will process related .ebl files in the SamplePool folder.
- `-d`: Debug - Prints debug messages, mostly EBL file read warnings.
- `-e`: Error Save. Writes files which can't be read to /output/errors/.
- `--version`: Display the version information.

## How It Works

This tool reads proprietary E-MU Emulator X-3 EBL files and converts them to the more open and accessible WAV format. No encoding is performed - EBL files store channel data in a similar format to WAV, although channels are split in EBL.

Original files are not modified in any way. Output filenames are taken from Emulator X-3 specified filenames encoded in the file header.

## Features

- Convert individual EBL files or process directories recursively
- Process EXB files and their associated SamplePool directories
- Preserve original directory structure in output
- Debug mode for detailed processing information
- Option to save files with errors for further investigation
- Support for both mono and stereo audio files