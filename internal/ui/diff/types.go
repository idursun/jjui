package diff

// FileStatus represents the status of a file in the diff
type FileStatus int

const (
	FileModified FileStatus = iota
	FileAdded
	FileDeleted
	FileRenamed
	FileCopied
)

// String returns a single-character representation of the file status
func (s FileStatus) String() string {
	switch s {
	case FileModified:
		return "M"
	case FileAdded:
		return "A"
	case FileDeleted:
		return "D"
	case FileRenamed:
		return "R"
	case FileCopied:
		return "C"
	default:
		return "?"
	}
}

// LineType represents the type of a diff line
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

// DiffFile represents a single file in a diff
type DiffFile struct {
	OldPath  string
	NewPath  string
	Status   FileStatus
	IsBinary bool
	Hunks    []Hunk
}

// Path returns the display path for the file
func (f *DiffFile) Path() string {
	if f.NewPath != "" {
		return f.NewPath
	}
	return f.OldPath
}

// Hunk represents a single hunk in a diff
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Header   string
	Lines    []DiffLine
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type      LineType
	OldLineNo int // 0 means no line number (for added lines)
	NewLineNo int // 0 means no line number (for removed lines)
	Content   string
	Segments  []Segment // For word-level diff highlighting
}

// Segment represents a portion of a line with a specific highlight status
type Segment struct {
	Text      string
	Highlight bool // true if this segment represents a change
}

// ParsedDiff contains all files from a parsed diff
type ParsedDiff struct {
	Files []*DiffFile
}

// TotalLines returns the total number of lines across all files and hunks
func (d *ParsedDiff) TotalLines() int {
	total := 0
	for _, file := range d.Files {
		total++ // File header line
		for _, hunk := range file.Hunks {
			total++ // Hunk header line
			total += len(hunk.Lines)
		}
	}
	return total
}
