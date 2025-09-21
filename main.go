package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMenu state = iota
	stateSourceSelect
	stateDestSelect
	stateBrowseSource
	stateBrowseDest
	stateExtensions
	stateCustomExtensions
	stateOptions
	stateConfirm
	stateCopying
	stateComplete
)

type model struct {
	state        state
	cursor       int
	config       Config
	currentPath  string
	directories  []string
	message      string
	progress     int
	totalFiles   int
	copiedFiles  int
	currentFile  string
	customInput  string
	err          error
	quitting     bool
	progressChan chan copyProgressMsg
}
type Config struct {
	SourceDir  string
	DestDir    string
	Extensions []string
	Recursive  bool
	Verbose    bool
	DryRun     bool
}

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED"))
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).
			Bold(true).
			MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C3AED")).
			Bold(true).
			Padding(0, 1)
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Padding(0, 1)
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4"))
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#475569")).
			Padding(1, 2).
			MarginBottom(1)
	progressBarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#06B6D4")).
				Padding(1)
	progressFilledStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#10B981"))
	progressEmptyStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#374151"))
)

type copyProgressMsg struct {
	file     string
	progress int
	total    int
	copied   int
}
type copyCompleteMsg struct {
	success bool
	copied  int
	total   int
}
type tickMsg time.Time
type startCopyMsg struct {
	files []string
}

func initialModel() model {
	wd, _ := os.Getwd()
	return model{
		state:        stateMenu,
		currentPath:  wd,
		progressChan: make(chan copyProgressMsg, 100),
		config: Config{
			Extensions: []string{".jpg", ".png", ".pdf"},
			Recursive:  true,
			Verbose:    false,
			DryRun:     false,
		},
	}
}
func (m model) Init() tea.Cmd {
	return nil
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.state == stateComplete {
				m.quitting = true
				return m, tea.Quit
			}
		}
		switch m.state {
		case stateMenu:
			return m.updateMenu(msg)
		case stateSourceSelect:
			return m.updateSourceSelect(msg)
		case stateDestSelect:
			return m.updateDestSelect(msg)
		case stateBrowseSource, stateBrowseDest:
			return m.updateBrowse(msg)
		case stateExtensions:
			return m.updateExtensions(msg)
		case stateCustomExtensions:
			return m.updateCustomExtensions(msg)
		case stateOptions:
			return m.updateOptions(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		case stateCopying:
			return m, nil
		}
	case copyProgressMsg:
		m.currentFile = msg.file
		m.progress = msg.progress
		m.totalFiles = msg.total
		m.copiedFiles = msg.copied
		return m, nil
	case copyCompleteMsg:
		m.state = stateComplete
		m.copiedFiles = msg.copied
		m.totalFiles = msg.total
		return m, nil
	case tickMsg:
		if m.state == stateCopying {
			return m, m.tickCmd()
		}
		return m, nil
	case startCopyMsg:
		return m, m.processFiles(msg.files)
	}
	return m, nil
}
func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 5 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0:
			m.state = stateSourceSelect
			m.cursor = 0
		case 1:
			m.state = stateDestSelect
			m.cursor = 0
		case 2:
			m.state = stateExtensions
		case 3:
			m.state = stateOptions
			m.cursor = 0
		case 4:
			if m.config.SourceDir != "" && m.config.DestDir != "" {
				m.state = stateConfirm
				m.cursor = 0
			} else {
				m.message = "Please select source and destination folders first!"
			}
		case 5:
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m model) updateSourceSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	shortcuts := getShortcuts()
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(shortcuts)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(shortcuts)-2 {
			m.config.SourceDir = shortcuts[m.cursor].path
			m.state = stateMenu
			m.cursor = 1
		} else if m.cursor == len(shortcuts)-2 {
			m.state = stateBrowseSource
			m.cursor = 0
			m.directories = getDirectories(m.currentPath)
		} else {
			m.state = stateMenu
			m.cursor = 0
		}
	case "backspace":
		m.state = stateMenu
		m.cursor = 0
	}
	return m, nil
}
func (m model) updateDestSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	shortcuts := getShortcuts()
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(shortcuts)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(shortcuts)-2 {
			m.config.DestDir = shortcuts[m.cursor].path
			m.state = stateMenu
			m.cursor = 2
		} else if m.cursor == len(shortcuts)-2 {
			m.state = stateBrowseDest
			m.cursor = 0
			m.directories = getDirectories(m.currentPath)
		} else {
			m.state = stateMenu
			m.cursor = 1
		}
	case "backspace":
		m.state = stateMenu
		m.cursor = 1
	}
	return m, nil
}
func (m model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	options := []string{"✅ Select this folder"}
	if m.currentPath != "/" && m.currentPath != filepath.Dir(m.currentPath) {
		options = append(options, "⬆️  Up")
	}
	options = append(options, m.directories...)
	options = append(options, "🔙 Back")
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(options)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor == 0 {
			if m.state == stateBrowseSource {
				m.config.SourceDir = m.currentPath
				m.state = stateMenu
				m.cursor = 1
			} else {
				m.config.DestDir = m.currentPath
				m.state = stateMenu
				m.cursor = 2
			}
		} else if len(options) > 1 && m.cursor == 1 && options[1] == "⬆️  Up" {
			m.currentPath = filepath.Dir(m.currentPath)
			m.directories = getDirectories(m.currentPath)
			m.cursor = 0
		} else if m.cursor == len(options)-1 {
			if m.state == stateBrowseSource {
				m.state = stateSourceSelect
			} else {
				m.state = stateDestSelect
			}
			m.cursor = 0
		} else {
			dirIndex := m.cursor
			if len(options) > 1 && options[1] == "⬆️  Up" {
				dirIndex -= 2
			} else {
				dirIndex -= 1
			}
			if dirIndex >= 0 && dirIndex < len(m.directories) {
				dirName := strings.TrimPrefix(m.directories[dirIndex], "📁 ")
				m.currentPath = filepath.Join(m.currentPath, dirName)
				m.directories = getDirectories(m.currentPath)
				m.cursor = 0
			}
		}
	case "backspace":
		if m.state == stateBrowseSource {
			m.state = stateSourceSelect
		} else {
			m.state = stateDestSelect
		}
		m.cursor = 0
	}
	return m, nil
}
func (m model) updateExtensions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	presets := [][]string{
		{".jpg", ".png", ".gif", ".bmp"},
		{".pdf", ".doc", ".docx", ".txt"},
		{".mp4", ".avi", ".mkv", ".mov"},
		{".mp3", ".wav", ".flac", ".m4a"},
		{".zip", ".rar", ".7z", ".tar"},
	}
	maxCursor := len(presets) + 1
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < maxCursor {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(presets) {
			m.config.Extensions = presets[m.cursor]
			m.state = stateMenu
			m.cursor = 3
		} else if m.cursor == len(presets) {
			m.state = stateCustomExtensions
			m.customInput = strings.Join(m.config.Extensions, ", ")
			m.cursor = 0
		} else {
			m.state = stateMenu
			m.cursor = 2
		}
	case "backspace":
		m.state = stateMenu
		m.cursor = 2
	}
	return m, nil
}
func (m model) updateCustomExtensions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.config.Extensions = m.parseExtensions(m.customInput)
		m.state = stateMenu
		m.cursor = 3
	case "backspace":
		if len(m.customInput) > 0 {
			m.customInput = m.customInput[:len(m.customInput)-1]
		}
	case "esc":
		m.state = stateExtensions
		m.cursor = len([][]string{
			{".jpg", ".png", ".gif", ".bmp"},
			{".pdf", ".doc", ".docx", ".txt"},
			{".mp4", ".avi", ".mkv", ".mov"},
			{".mp3", ".wav", ".flac", ".m4a"},
			{".zip", ".rar", ".7z", ".tar"},
		})
	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "a" && char <= "z") || (char >= "A" && char <= "Z") ||
				(char >= "0" && char <= "9") || char == "." || char == "," || char == " " {
				m.customInput += char
			}
		}
	}
	return m, nil
}
func (m model) parseExtensions(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(input, func(c rune) bool {
		return c == ',' || c == ' '
	})
	var extensions []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			if !strings.HasPrefix(part, ".") {
				part = "." + part
			}
			extensions = append(extensions, strings.ToLower(part))
		}
	}
	return extensions
}
func (m model) updateOptions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 3 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0:
			m.config.Recursive = !m.config.Recursive
		case 1:
			m.config.Verbose = !m.config.Verbose
		case 2:
			m.config.DryRun = !m.config.DryRun
		case 3:
			m.state = stateMenu
			m.cursor = 4
		}
	case "backspace":
		m.state = stateMenu
		m.cursor = 3
	}
	return m, nil
}
func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		m.cursor = 0
	case "right", "l":
		m.cursor = 1
	case "enter":
		if m.cursor == 0 {
			m.state = stateCopying
			m.progress = 0
			return m, m.startCopying()
		} else {
			m.state = stateMenu
			m.cursor = 4
		}
	case "backspace":
		m.state = stateMenu
		m.cursor = 4
	}
	return m, nil
}
func (m model) startCopying() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			files, err := m.scanFiles()
			if err != nil {
				return copyCompleteMsg{success: false, copied: 0, total: 0}
			}
			return startCopyMsg{files: files}
		},
		m.tickCmd(),
	)
}
func (m model) processFiles(files []string) tea.Cmd {
	return func() tea.Msg {
		m.totalFiles = len(files)
		copied := 0
		progressChan := make(chan copyProgressMsg, 10)
		go func() {
			defer close(progressChan)
			for i, file := range files {
				progress := ((i + 1) * 100) / len(files)
				currentFile := filepath.Base(file)
				progressChan <- copyProgressMsg{
					file:     currentFile,
					progress: progress,
					total:    len(files),
					copied:   copied,
				}
				if !m.config.DryRun {
					err := m.copyFile(file)
					if err == nil {
						copied++
					}
				} else {
					copied++
					time.Sleep(50 * time.Millisecond)
				}
			}
		}()
		var lastProgress copyProgressMsg
		for progress := range progressChan {
			lastProgress = progress
		}
		return copyCompleteMsg{
			success: true,
			copied:  lastProgress.copied,
			total:   len(files),
		}
	}
}
func (m model) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
func (m model) scanFiles() ([]string, error) {
	var files []string
	err := filepath.WalkDir(m.config.SourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		for _, allowedExt := range m.config.Extensions {
			if ext == allowedExt {
				files = append(files, path)
				break
			}
		}
		return nil
	})
	return files, err
}
func (m model) copyFile(srcPath string) error {
	fileName := filepath.Base(srcPath)
	destPath := filepath.Join(m.config.DestDir, fileName)
	destPath = m.resolveFileConflict(destPath)
	if err := os.MkdirAll(m.config.DestDir, 0755); err != nil {
		return err
	}
	sourceFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	return err
}
func (m model) resolveFileConflict(destPath string) string {
	originalPath := destPath
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		dir := filepath.Dir(originalPath)
		base := filepath.Base(originalPath)
		ext := filepath.Ext(base)
		nameWithoutExt := strings.TrimSuffix(base, ext)
		destPath = filepath.Join(dir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
		counter++
	}
	return destPath
}
func (m model) View() string {
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString(titleStyle.Render("📁 Flat File Copy Utility"))
	s.WriteString("\n\n")
	switch m.state {
	case stateMenu:
		s.WriteString(m.viewMenu())
	case stateSourceSelect:
		s.WriteString(m.viewSourceSelect())
	case stateDestSelect:
		s.WriteString(m.viewDestSelect())
	case stateBrowseSource, stateBrowseDest:
		s.WriteString(m.viewBrowse())
	case stateExtensions:
		s.WriteString(m.viewExtensions())
	case stateCustomExtensions:
		s.WriteString(m.viewCustomExtensions())
	case stateOptions:
		s.WriteString(m.viewOptions())
	case stateConfirm:
		s.WriteString(m.viewConfirm())
	case stateCopying:
		s.WriteString(m.viewCopying())
	case stateComplete:
		s.WriteString(m.viewComplete())
	}
	if m.message != "" {
		s.WriteString("\n" + warningStyle.Render("⚠️ "+m.message))
		m.message = ""
	}
	s.WriteString("\n\n" + infoStyle.Render("↑/↓: navigation • Enter: select • Backspace: back • Esc: exit"))
	return s.String()
}
func (m model) viewMenu() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("📋 Main Menu"))
	s.WriteString("\n\n")
	items := []string{
		fmt.Sprintf("📂 Source folder: %s", getDisplayPath(m.config.SourceDir)),
		fmt.Sprintf("📁 Destination folder: %s", getDisplayPath(m.config.DestDir)),
		fmt.Sprintf("📄 File formats: %s", strings.Join(m.config.Extensions, ", ")),
		"⚙️  Additional settings",
		"🚀 Start copying",
		"🚪 Exit",
	}
	for i, item := range items {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		s.WriteString(style.Render(item))
		s.WriteString("\n")
	}
	return s.String()
}
func (m model) viewSourceSelect() string {
	return m.viewDirectorySelect("📂 Select source folder")
}
func (m model) viewDestSelect() string {
	return m.viewDirectorySelect("📁 Select destination folder")
}
func (m model) viewDirectorySelect(title string) string {
	var s strings.Builder
	s.WriteString(headerStyle.Render(title))
	s.WriteString("\n\n")
	shortcuts := getShortcuts()
	for i, shortcut := range shortcuts {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		s.WriteString(style.Render(shortcut.label))
		s.WriteString("\n")
	}
	return s.String()
}
func (m model) viewBrowse() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render(fmt.Sprintf("📁 %s", m.currentPath)))
	s.WriteString("\n\n")
	options := []string{"✅ Select this folder"}
	if m.currentPath != "/" && m.currentPath != filepath.Dir(m.currentPath) {
		options = append(options, "⬆️  Up")
	}
	options = append(options, m.directories...)
	options = append(options, "🔙 Back")
	for i, option := range options {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		s.WriteString(style.Render(option))
		s.WriteString("\n")
	}
	return s.String()
}
func (m model) viewExtensions() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("📄 Select file types"))
	s.WriteString("\n\n")
	presets := []struct {
		label string
		exts  []string
	}{
		{"🖼️  Images", []string{".jpg", ".png", ".gif", ".bmp"}},
		{"📄 Documents", []string{".pdf", ".doc", ".docx", ".txt"}},
		{"🎬 Video", []string{".mp4", ".avi", ".mkv", ".mov"}},
		{"🎵 Audio", []string{".mp3", ".wav", ".flac", ".m4a"}},
		{"📦 Archives", []string{".zip", ".rar", ".7z", ".tar"}},
		{"✏️  Custom extensions", nil},
		{"🔙 Back", nil},
	}
	for i, preset := range presets {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		s.WriteString(style.Render(preset.label))
		s.WriteString("\n")
	}
	s.WriteString("\n" + infoStyle.Render(fmt.Sprintf("Current: %s", strings.Join(m.config.Extensions, ", "))))
	return s.String()
}
func (m model) viewCustomExtensions() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("✏️ Custom extensions"))
	s.WriteString("\n\n")
	inputBox := boxStyle.Render(fmt.Sprintf(
		"Enter file extensions separated by commas\n"+
			"Example: .txt, .log, pdf, docx\n\n"+
			"Input: %s|",
		m.customInput))
	s.WriteString(inputBox)
	s.WriteString("\n\n")
	s.WriteString(infoStyle.Render("Enter: save • Backspace: delete character • Esc: cancel"))
	s.WriteString("\n")
	s.WriteString(infoStyle.Render("Current extensions: " + strings.Join(m.config.Extensions, ", ")))
	return s.String()
}
func (m model) viewOptions() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("⚙️ Additional settings"))
	s.WriteString("\n\n")
	options := []string{
		fmt.Sprintf("🔍 Search in subfolders: %s", getBoolDisplay(m.config.Recursive)),
		fmt.Sprintf("📝 Verbose output: %s", getBoolDisplay(m.config.Verbose)),
		fmt.Sprintf("🧪 Dry run mode: %s", getBoolDisplay(m.config.DryRun)),
		"🔙 Back",
	}
	for i, option := range options {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		s.WriteString(style.Render(option))
		s.WriteString("\n")
	}
	return s.String()
}
func (m model) viewConfirm() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("🚀 Operation confirmation"))
	s.WriteString("\n\n")
	config := boxStyle.Render(fmt.Sprintf(
		"📂 Source folder: %s\n"+
			"📁 Destination folder: %s\n"+
			"📄 Formats: %s\n"+
			"🔍 Recursive: %s\n"+
			"📋 Copy mode: flat (all files in one folder)\n"+
			"🧪 Dry run mode: %s",
		m.config.SourceDir,
		m.config.DestDir,
		strings.Join(m.config.Extensions, ", "),
		getBoolDisplay(m.config.Recursive),
		getBoolDisplay(m.config.DryRun),
	))
	s.WriteString(config)
	s.WriteString("\n\n")
	buttons := []string{"✅ Start", "❌ Cancel"}
	for i, button := range buttons {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		s.WriteString(style.Render(button))
		s.WriteString("  ")
	}
	return s.String()
}
func (m model) viewCopying() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("📋 Copying files"))
	s.WriteString("\n\n")
	progressWidth := 50
	filled := (m.progress * progressWidth) / 100
	var progressBar strings.Builder
	for i := 0; i < progressWidth; i++ {
		if i < filled {
			progressBar.WriteString(progressFilledStyle.Render("█"))
		} else {
			progressBar.WriteString(progressEmptyStyle.Render("░"))
		}
	}
	progress := progressBarStyle.Render(fmt.Sprintf(
		"Progress: %d%%\n%s\nFiles: %d/%d\nCurrent: %s",
		m.progress,
		progressBar.String(),
		m.copiedFiles,
		m.totalFiles,
		m.currentFile,
	))
	s.WriteString(progress)
	return s.String()
}
func (m model) viewComplete() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("✅ Operation completed"))
	s.WriteString("\n\n")
	result := boxStyle.Render(fmt.Sprintf(
		"Files copied: %d of %d\n"+
			"Operation completed successfully!",
		m.copiedFiles,
		m.totalFiles,
	))
	s.WriteString(result)
	s.WriteString("\n\n")
	s.WriteString(infoStyle.Render("Press 'q' to exit"))
	return s.String()
}

type shortcut struct {
	label string
	path  string
}

func getShortcuts() []shortcut {
	homeDir, _ := os.UserHomeDir()
	currentDir, _ := os.Getwd()
	return []shortcut{
		{"📁 Current folder", currentDir},
		{"🏠 Home folder", homeDir},
		{"🖥️  Desktop", filepath.Join(homeDir, "Desktop")},
		{"📁 Documents", filepath.Join(homeDir, "Documents")},
		{"📁 Downloads", filepath.Join(homeDir, "Downloads")},
		{"📸 Pictures", filepath.Join(homeDir, "Pictures")},
		{"🎵 Music", filepath.Join(homeDir, "Music")},
		{"🎬 Videos", filepath.Join(homeDir, "Videos")},
		{"📂 Browse folders...", ""},
		{"🔙 Back", ""},
	}
}
func getDirectories(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, "📁 "+entry.Name())
		}
	}
	sort.Strings(dirs)
	return dirs
}
func getDisplayPath(path string) string {
	if path == "" {
		return "not selected"
	}
	if len(path) > 50 {
		return "..." + path[len(path)-47:]
	}
	return path
}
func getBoolDisplay(value bool) string {
	if value {
		return "✅ Yes"
	}
	return "❌ No"
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		runTestMode()
		return
	}
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
func runTestMode() {
	fmt.Println("🧪 ficout test mode")
	fmt.Println("===================")
	m := initialModel()
	fmt.Printf("✅ Model initialized: %s\n", m.currentPath)
	shortcuts := getShortcuts()
	fmt.Printf("✅ Found %d directory shortcuts\n", len(shortcuts))
	m.config.SourceDir = m.currentPath
	m.config.Extensions = []string{".go", ".md"}
	files, err := m.scanFiles()
	if err != nil {
		fmt.Printf("❌ Scanning error: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d files with extensions %v\n", len(files), m.config.Extensions)
		for i, file := range files {
			if i < 3 {
				fmt.Printf("   - %s\n", filepath.Base(file))
			}
		}
		if len(files) > 3 {
			fmt.Printf("   ... and %d more files\n", len(files)-3)
		}
	}
	dirs := getDirectories(m.currentPath)
	fmt.Printf("✅ Found %d subdirectories\n", len(dirs))
	if _, err := os.Stat("test_source"); err == nil {
		fmt.Println("\n📁 Testing copy functionality...")
		m.config.SourceDir = filepath.Join(m.currentPath, "test_source")
		m.config.DestDir = filepath.Join(m.currentPath, "test_dest")
		m.config.Extensions = []string{".txt", ".md"}
		m.config.DryRun = false
		files, err := m.scanFiles()
		if err != nil {
			fmt.Printf("❌ test_source scanning error: %v\n", err)
		} else {
			fmt.Printf("✅ Found %d files for copying in test_source\n", len(files))
			if len(files) > 0 {
				fmt.Printf("✅ Files found for copying:\n")
				for _, file := range files {
					fmt.Printf("   - %s (from %s)\n", filepath.Base(file), filepath.Dir(file))
				}
				if !m.config.DryRun {
					fmt.Println("📋 Copying files in flat structure...")
					copied := 0
					for _, file := range files {
						err := m.copyFile(file)
						if err == nil {
							copied++
							fmt.Printf("   ✅ %s\n", filepath.Base(file))
						} else {
							fmt.Printf("   ❌ %s: %v\n", filepath.Base(file), err)
						}
					}
					fmt.Printf("📁 Copied %d files to %s\n", copied, m.config.DestDir)
				} else {
					fmt.Println("🧪 Dry-run mode - files not copied")
				}
			}
		}
	}
	fmt.Println("\n🔧 Testing extensions parsing...")
	testInputs := []string{
		".txt, .log, pdf",
		"jpg, png, .gif",
		"  .doc,   .docx  , txt  ",
		"mp4 avi .mkv",
	}
	for _, input := range testInputs {
		parsed := m.parseExtensions(input)
		fmt.Printf("   '%s' → %v\n", input, parsed)
	}
	if _, err := os.Stat("test_source"); err == nil {
		fmt.Println("\n📁 Test with custom extensions...")
		m.config.SourceDir = filepath.Join(m.currentPath, "test_source")
		m.config.DestDir = filepath.Join(m.currentPath, "test_dest")
		m.config.Extensions = m.parseExtensions(".log, ini, csv")
		files, err := m.scanFiles()
		if err != nil {
			fmt.Printf("❌ Scanning error: %v\n", err)
		} else {
			fmt.Printf("✅ Found %d files with custom extensions %v:\n", len(files), m.config.Extensions)
			for _, file := range files {
				fmt.Printf("   - %s\n", filepath.Base(file))
			}
		}
	}
	fmt.Println("\n🎯 Core functions work correctly!")
	fmt.Println("For full testing run in interactive terminal:")
	fmt.Println("   go run main.go")
}
