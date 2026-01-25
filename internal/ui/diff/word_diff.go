package diff

import (
	"strings"
	"unicode"
)

// ComputeWordDiff computes word-level differences between an old and new line
// and updates the Segments field of both lines
func ComputeWordDiff(oldLine, newLine *DiffLine) {
	if oldLine == nil || newLine == nil {
		return
	}

	oldTokens := tokenize(oldLine.Content)
	newTokens := tokenize(newLine.Content)

	// Compute LCS (Longest Common Subsequence)
	lcs := computeLCS(oldTokens, newTokens)

	// Build segments for old line (removed line)
	oldLine.Segments = buildSegments(oldTokens, lcs, true)

	// Build segments for new line (added line)
	newLine.Segments = buildSegments(newTokens, lcs, false)
}

// tokenize splits a string into tokens (words and whitespace/punctuation)
func tokenize(s string) []string {
	if s == "" {
		return nil
	}

	var tokens []string
	var current strings.Builder
	var lastType tokenType = tokenNone

	for _, r := range s {
		currType := getTokenType(r)

		if lastType != tokenNone && currType != lastType {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}

		current.WriteRune(r)
		lastType = currType
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

type tokenType int

const (
	tokenNone tokenType = iota
	tokenWord
	tokenSpace
	tokenPunct
)

func getTokenType(r rune) tokenType {
	if unicode.IsSpace(r) {
		return tokenSpace
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return tokenWord
	}
	return tokenPunct
}

// computeLCS computes the longest common subsequence of two token slices
// Returns a map of indices from both slices that are part of the LCS
func computeLCS(old, new []string) map[string]bool {
	m := len(old)
	n := len(new)

	if m == 0 || n == 0 {
		return make(map[string]bool)
	}

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if old[i-1] == new[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS elements
	lcs := make(map[string]bool)
	i, j := m, n
	for i > 0 && j > 0 {
		if old[i-1] == new[j-1] {
			// Format key as "old:i,new:j" to track positions
			lcs[formatKey("old", i-1)] = true
			lcs[formatKey("new", j-1)] = true
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

func formatKey(prefix string, index int) string {
	return prefix + ":" + strings.Repeat("x", index+1) // Simple unique key
}

// buildSegments creates segments from tokens, marking non-LCS tokens as highlighted
func buildSegments(tokens []string, lcs map[string]bool, isOld bool) []Segment {
	if len(tokens) == 0 {
		return nil
	}

	prefix := "new"
	if isOld {
		prefix = "old"
	}

	segments := make([]Segment, 0, len(tokens))
	var currentText strings.Builder
	currentHighlight := false

	for i, token := range tokens {
		key := formatKey(prefix, i)
		inLCS := lcs[key]
		highlight := !inLCS

		if i > 0 && highlight != currentHighlight {
			if currentText.Len() > 0 {
				segments = append(segments, Segment{
					Text:      currentText.String(),
					Highlight: currentHighlight,
				})
				currentText.Reset()
			}
		}

		currentText.WriteString(token)
		currentHighlight = highlight
	}

	if currentText.Len() > 0 {
		segments = append(segments, Segment{
			Text:      currentText.String(),
			Highlight: currentHighlight,
		})
	}

	return segments
}

// ComputeWordDiffForHunk processes a hunk and computes word-level diffs
// for adjacent removed/added line pairs
func ComputeWordDiffForHunk(hunk *Hunk) {
	if hunk == nil || len(hunk.Lines) == 0 {
		return
	}

	// Find sequences of removed lines followed by added lines
	i := 0
	for i < len(hunk.Lines) {
		// Skip context lines
		if hunk.Lines[i].Type == LineContext {
			i++
			continue
		}

		// Find a block of removed lines
		removedStart := i
		for i < len(hunk.Lines) && hunk.Lines[i].Type == LineRemoved {
			i++
		}
		removedEnd := i

		// Find a block of added lines
		addedStart := i
		for i < len(hunk.Lines) && hunk.Lines[i].Type == LineAdded {
			i++
		}
		addedEnd := i

		removedCount := removedEnd - removedStart
		addedCount := addedEnd - addedStart

		// Pair up removed and added lines for word diff
		pairCount := min(removedCount, addedCount)
		for j := 0; j < pairCount; j++ {
			ComputeWordDiff(&hunk.Lines[removedStart+j], &hunk.Lines[addedStart+j])
		}

		// Remaining unpaired lines get full highlight
		for j := pairCount; j < removedCount; j++ {
			line := &hunk.Lines[removedStart+j]
			if line.Content != "" {
				line.Segments = []Segment{{Text: line.Content, Highlight: true}}
			}
		}
		for j := pairCount; j < addedCount; j++ {
			line := &hunk.Lines[addedStart+j]
			if line.Content != "" {
				line.Segments = []Segment{{Text: line.Content, Highlight: true}}
			}
		}
	}
}

// ComputeWordDiffForFile processes all hunks in a file
func ComputeWordDiffForFile(file *DiffFile) {
	if file == nil {
		return
	}
	for i := range file.Hunks {
		ComputeWordDiffForHunk(&file.Hunks[i])
	}
}

// ComputeWordDiffForDiff processes all files in a diff
func ComputeWordDiffForDiff(diff *ParsedDiff) {
	if diff == nil {
		return
	}
	for _, file := range diff.Files {
		ComputeWordDiffForFile(file)
	}
}
