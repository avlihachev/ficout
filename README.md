# Ficout - Flat File Copy Utility

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)

A terminal user interface (TUI) application for copying files from multiple directories into a single flat structure.

## Features

- **Flat File Copying**: Copies all files from source directory (including subdirectories) to destination folder without preserving directory structure
- **Interactive File Browser**: Navigate filesystem with shortcuts for common directories
- **File Type Filtering**: Choose from predefined sets (Images, Documents, Video, Audio, Archives) or define custom extensions
- **Custom Extensions**: Input your own file extensions separated by commas
- **Progress Tracking**: Real-time progress bar during file operations
- **Dry Run Mode**: Preview operations without actually copying files
- **File Conflict Resolution**: Automatically handles duplicate filenames by adding numbers

## Installation

### Prerequisites
- Go 1.25.1 or later

### Build from source
```bash
git clone <repository-url>
cd ficout
go build
```

## Usage

### Interactive Mode
```bash
./ficout
```

### Test Mode
```bash
./ficout --test
```

## Navigation

- **↑/↓ or j/k**: Navigate menu items
- **Enter**: Select item
- **Backspace**: Go back
- **Esc**: Exit application
- **q**: Quit (in completion screen)

## File Type Examples

### Predefined Sets
- **Images**: .jpg, .png, .gif, .bmp
- **Documents**: .pdf, .doc, .docx, .txt
- **Video**: .mp4, .avi, .mkv, .mov
- **Audio**: .mp3, .wav, .flac, .m4a
- **Archives**: .zip, .rar, .7z, .tar

### Custom Extensions
Enter extensions like: `.txt, .log, pdf, docx`
- Automatic dot prefix addition
- Case insensitive
- Comma or space separated

## Example Workflow

1. Select source folder (where files are located)
2. Select destination folder (where files will be copied)
3. Choose file types or define custom extensions
4. Configure additional settings (recursive search, dry run, etc.)
5. Review and start copying

## Copy Behavior

All files matching the selected extensions will be copied to a single destination folder:

```
Source structure:
project/
├── file1.txt
├── docs/
│   ├── readme.txt
│   └── guide.pdf
└── images/
    └── logo.png

Destination (flat):
backup/
├── file1.txt
├── readme_1.txt  (renamed to avoid conflict)
├── guide.pdf
└── logo.png
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

## Development

### Build
```bash
go build -o ficout
```

### Test
```bash
go run main.go --test
```

### Format
```bash
go fmt
```

### Vet
```bash
go vet
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
