package jj

import (
	"bufio"
	"io"
	"slices"
	"strings"
	"unicode"
)

// ConnectionType defines the types of connections in the input
type ConnectionType string

const (
	SPACE              = "  "
	HORIZONTAL         = "──"
	VERTICAL           = "│ "
	MERGE_LEFT         = "╯ "
	MERGE_RIGHT        = "╰─"
	MERGE_BOTH         = "┴─"
	FORK_LEFT          = "╮ "
	FORK_RIGHT         = "╭─"
	FORK_BOTH          = "┬─"
	JOIN_LEFT          = "┤ "
	JOIN_RIGHT         = "├─"
	JOIN_BOTH          = "┼─"
	TERMINATION        = "~ "
	GLYPH              = "○ "
	GLYPH_IMMUTABLE    = "◆ "
	GLYPH_CONFLICT     = "× "
	GLYPH_WORKING_COPY = "@ "
)

type Parser struct {
	reader     *bufio.Reader
	firstRune  rune
	secondRune rune
}

type GraphRow struct {
	Connections [][]ConnectionType
	Commit      *Commit
	IsSelected  bool
	IsAffected  bool
}

func NewParser(reader io.Reader) *Parser {
	p := &Parser{
		reader: bufio.NewReader(reader),
	}
	return p
}

func (p *Parser) Parse() []GraphRow {
	ret := make([]GraphRow, 0)
	for p.advance() {
		if p.firstRune == '\n' {
			break
		}
		var connections []ConnectionType
		for p.firstRune != 0 {
			connectionType := p.isConnectionType()
			if connectionType != "" {
				connections = append(connections, connectionType)
				p.advance()
				if p.firstRune == '\n' {
					break
				}
				p.advance()
			} else {
				break
			}
		}
		if p.firstRune != '\n' {
			p.skipWhiteSpace()
		}
		content := p.parseText()
		var commit Commit
		if slices.ContainsFunc(connections, func(c ConnectionType) bool {
			return c == GLYPH_IMMUTABLE || c == GLYPH_WORKING_COPY || c == GLYPH_CONFLICT || c == GLYPH
		}) {
			commit = p.parseCommit(content)
			r := GraphRow{Connections: [][]ConnectionType{connections}, Commit: &commit}
			ret = append(ret, r)
		} else if len(ret) > 0 {
			previousLine := &ret[len(ret)-1]
			previousLine.Connections = append(previousLine.Connections, connections)
		} else {
			// jj log might print information before the first commit, hence we skip it
			// e.g. if commits after the working copy are rebased
			continue
		}
	}
	return ret
}

func (p *Parser) parseCommit(content string) Commit {
	parts := strings.Split(content, ";")
	commit := Commit{
		ChangeIdShort: parts[0],
	}
	if len(parts) > 1 {
		commit.ChangeId = parts[1]
	}
	if len(parts) > 2 && parts[2] != "." {
		commit.Bookmarks = strings.Split(parts[2], ",")
	}
	if len(parts) > 3 {
		commit.IsWorkingCopy = parts[3] == "true"
	}
	if len(parts) > 4 {
		commit.Immutable = parts[4] == "true"
	}
	if len(parts) > 5 {
		commit.Conflict = parts[5] == "true"
	}
	if len(parts) > 6 {
		commit.Empty = parts[6] == "true"
	}
	if len(parts) > 7 {
		commit.Hidden = parts[7] == "true"
	}
	if len(parts) > 8 {
		commit.Author = parts[8]
	}
	if len(parts) > 9 {
		commit.Timestamp = parts[9]
	}
	if len(parts) > 10 {
		commit.CommitIdShort = parts[10]
	}
	if len(parts) > 11 {
		commit.CommitId = parts[11]
	}
	if len(parts) > 12 {
		commit.Description = parts[12]
	}
	if commit.IsRoot() {
		commit.Conflict = false
		commit.Immutable = false
		commit.Author = ""
		commit.Bookmarks = nil
		commit.Description = ""
	}
	return commit
}

func (p *Parser) parseText() string {
	var buffer strings.Builder
	for p.firstRune != 0 && p.firstRune != '\n' && p.firstRune != '\r' {
		buffer.WriteRune(p.firstRune)
		p.advance()
	}
	return buffer.String()
}

func (p *Parser) isConnectionType() ConnectionType {
	if p.secondRune == 0 || p.secondRune == '\n' {
		p.secondRune = ' '
	}

	twoRune := string([]rune{p.firstRune, p.secondRune})

	switch twoRune {
	case SPACE:
		return SPACE
	case HORIZONTAL:
		return HORIZONTAL
	case VERTICAL:
		return VERTICAL
	case MERGE_LEFT:
		return MERGE_LEFT
	case MERGE_RIGHT:
		return MERGE_RIGHT
	case MERGE_BOTH:
		return MERGE_BOTH
	case FORK_LEFT:
		return FORK_LEFT
	case FORK_RIGHT:
		return FORK_RIGHT
	case FORK_BOTH:
		return FORK_BOTH
	case JOIN_LEFT:
		return JOIN_LEFT
	case JOIN_RIGHT:
		return JOIN_RIGHT
	case JOIN_BOTH:
		return JOIN_BOTH
	case GLYPH:
		return GLYPH
	case GLYPH_IMMUTABLE:
		return GLYPH_IMMUTABLE
	case GLYPH_CONFLICT:
		return GLYPH_CONFLICT
	case GLYPH_WORKING_COPY:
		return GLYPH_WORKING_COPY
	case TERMINATION:
		return TERMINATION
	default:
		return ""
	}
}

func (p *Parser) advance() bool {
	if r, _, err := p.reader.ReadRune(); err == nil {
		p.firstRune = r
		if s, _, err := p.reader.ReadRune(); err == nil {
			p.secondRune = s
			_ = p.reader.UnreadRune()
		} else {
			p.secondRune = 0
		}
	} else {
		p.firstRune = 0
		return false
	}
	return true
}

func (p *Parser) skipWhiteSpace() {
	for unicode.IsSpace(p.firstRune) {
		p.advance()
	}
}
