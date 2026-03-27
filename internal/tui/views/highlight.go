package views

import (
	"image/color"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	udiff "github.com/aymanbagabas/go-udiff"

	"github.com/allir/c5s/internal/claude"
	"github.com/allir/c5s/internal/tui/theme"
)

// diffParts holds the diffParts components of a formatted diff line.
type diffParts struct {
	marker byte
	prefix string
	code   string
}

// diffPrefix extracts the diff marker, line number prefix, and code content.
// formatEditDiff produces fixed-width formats:
//
//	With line numbers (6-char prefix): "%3d + %s", "%3d - %s", "%3d   %s"
//	Without line numbers (4-char prefix): "  + %s", "  - %s", "    %s"
func diffPrefix(line string) (marker byte, prefix, code string) {
	// With line numbers: marker is at index 4 (e.g., " 29 - code")
	// Detect by checking if any of the first 3 chars is a digit.
	if len(line) >= 6 {
		hasDigit := false
		for _, c := range line[:3] {
			if c >= '0' && c <= '9' {
				hasDigit = true
				break
			}
		}
		if hasDigit {
			m := line[4]
			if m == '+' || m == '-' {
				return m, line[:6], line[6:]
			}
			return ' ', line[:6], line[6:]
		}
	}
	// Without line numbers: marker is at index 2 (e.g., "  + code")
	if len(line) >= 4 {
		m := line[2]
		if m == '+' || m == '-' {
			return m, line[:4], line[4:]
		}
		return ' ', line[:4], line[4:]
	}
	return ' ', line, ""
}

// charRegion marks a byte range within a line as changed or unchanged.
type charRegion struct {
	start, end int  // byte offsets into the code string
	changed    bool // true = this region was modified
}

// lineBg holds the normal and inline (brighter) backgrounds for a diff line.
type lineBg struct {
	normal color.Color // base diff bg
	inline color.Color // brighter bg for changed chars
}

// inlineRegions computes character-level diff regions for paired delete/insert
// lines. Returns a map from line index → regions.
func inlineRegions(parts []diffParts) map[int][]charRegion {
	pairs := pairChangedLines(parts)
	if len(pairs) == 0 {
		return nil
	}

	regions := make(map[int][]charRegion, len(pairs)*2)
	for _, p := range pairs {
		oldLine := parts[p[0]].code
		newLine := parts[p[1]].code
		if oldLine == newLine {
			continue
		}
		oldR, newR := charDiff(oldLine, newLine)
		// Skip inline highlighting if most of the line changed — it's just noise
		if changedRatio(oldR, len(oldLine)) > 0.7 && changedRatio(newR, len(newLine)) > 0.7 {
			continue
		}
		if len(oldR) > 0 {
			regions[p[0]] = oldR
		}
		if len(newR) > 0 {
			regions[p[1]] = newR
		}
	}
	return regions
}

// pairChangedLines finds adjacent delete/insert runs and pairs them by
// similarity. Returns pairs as [2]int{deleteIdx, insertIdx}.
func pairChangedLines(parts []diffParts) [][2]int {
	var pairs [][2]int
	i := 0
	for i < len(parts) {
		// Find run of deletes
		delStart := i
		for i < len(parts) && parts[i].marker == '-' {
			i++
		}
		delEnd := i
		// Find subsequent run of inserts
		insStart := i
		for i < len(parts) && parts[i].marker == '+' {
			i++
		}
		insEnd := i

		// Match each delete to the most similar insert (greedy, forward-only)
		used := make([]bool, insEnd-insStart)
		searchFrom := 0 // only search forward in the insert run
		for d := delStart; d < delEnd; d++ {
			bestIdx := -1
			bestScore := 0
			for k := searchFrom; k < insEnd-insStart; k++ {
				if used[k] {
					continue
				}
				score := lineSimilarity(parts[d].code, parts[insStart+k].code)
				if score > bestScore {
					bestScore = score
					bestIdx = k
				}
			}
			// Only pair if at least 50% similar
			minLen := max(len(parts[d].code), 1)
			if bestIdx >= 0 && bestScore*2 >= minLen {
				pairs = append(pairs, [2]int{d, insStart + bestIdx})
				used[bestIdx] = true
				// Advance search window past matched inserts
				for searchFrom < insEnd-insStart && used[searchFrom] {
					searchFrom++
				}
			}
		}

		// Skip context lines
		if i < len(parts) && parts[i].marker != '-' && parts[i].marker != '+' {
			i++
		}
	}
	return pairs
}

// changedRatio returns the fraction of bytes in the regions that are marked changed.
func changedRatio(regions []charRegion, lineLen int) float64 {
	if lineLen == 0 {
		return 0
	}
	changed := 0
	for _, r := range regions {
		if r.changed {
			changed += r.end - r.start
		}
	}
	return float64(changed) / float64(lineLen)
}

// lineSimilarity returns the length of the common prefix + common suffix
// between two strings, capped at the shorter string's length.
func lineSimilarity(a, b string) int {
	if a == b {
		return len(a)
	}
	n := min(len(a), len(b))

	// Common prefix
	pfx := 0
	for pfx < n && a[pfx] == b[pfx] {
		pfx++
	}

	// Common suffix (don't overlap with prefix)
	sfx := 0
	for sfx < n-pfx && a[len(a)-1-sfx] == b[len(b)-1-sfx] {
		sfx++
	}

	return min(pfx+sfx, n)
}

// charDiff computes character-level regions for a pair of old/new lines
// using Myers' algorithm via go-udiff.
func charDiff(oldLine, newLine string) (oldRegions, newRegions []charRegion) {
	edits := udiff.Strings(oldLine, newLine)
	if len(edits) == 0 {
		return nil, nil
	}

	// Build old-line regions from edit spans
	pos := 0
	for _, e := range edits {
		if e.Start > pos {
			oldRegions = append(oldRegions, charRegion{pos, e.Start, false})
		}
		if e.End > e.Start {
			oldRegions = append(oldRegions, charRegion{e.Start, e.End, true})
		}
		pos = e.End
	}
	if pos < len(oldLine) {
		oldRegions = append(oldRegions, charRegion{pos, len(oldLine), false})
	}

	// Build new-line regions by applying edits to track positions
	newPos := 0
	oldPos := 0
	for _, e := range edits {
		// Unchanged gap before this edit
		gap := e.Start - oldPos
		if gap > 0 {
			newRegions = append(newRegions, charRegion{newPos, newPos + gap, false})
			newPos += gap
		}
		// The replacement text in the new string
		if len(e.New) > 0 {
			newRegions = append(newRegions, charRegion{newPos, newPos + len(e.New), true})
			newPos += len(e.New)
		}
		oldPos = e.End
	}
	// Trailing unchanged text
	trailing := len(oldLine) - oldPos
	if trailing > 0 {
		newRegions = append(newRegions, charRegion{newPos, newPos + trailing, false})
	}

	return oldRegions, newRegions
}

// renderDiffBlock takes a slice of consecutive diff entries (all sharing the
// same FilePath) and returns syntax-highlighted, styled lines.
func renderDiffBlock(entries []claude.TranscriptEntry) []string {
	if len(entries) == 0 {
		return nil
	}

	// Pre-build styles — prefix and fallback include the diff background
	// so the entire row is highlighted, not just the code part.
	removePfx := lipgloss.NewStyle().Foreground(theme.ColorDiffRemoveFg).Background(theme.ColorDiffRemoveBg)
	addPfx := lipgloss.NewStyle().Foreground(theme.ColorDiffAddFg).Background(theme.ColorDiffAddBg)
	ctxPfx := lipgloss.NewStyle().Foreground(theme.ColorMuted)

	// Extract code lines for tokenization
	parts := make([]diffParts, len(entries))
	codeLines := make([]string, len(entries))
	for i, e := range entries {
		marker, pfx, code := diffPrefix(e.Content)
		parts[i] = diffParts{marker, pfx, code}
		codeLines[i] = code
	}

	// Compute inline (character-level) diff regions for paired lines
	inline := inlineRegions(parts)

	// Tokenize the code block as a whole
	codeBlock := strings.Join(codeLines, "\n")
	highlighted := highlightCodeWithBg(codeBlock, entries[0].FilePath, parts, inline)

	// Build styled output lines — whole row gets the diff background
	lines := make([]string, len(entries))
	for i, p := range parts {
		var pfxStyle lipgloss.Style
		switch p.marker {
		case '-':
			pfxStyle = removePfx
		case '+':
			pfxStyle = addPfx
		default:
			pfxStyle = ctxPfx
		}

		prefixStyled := "  " + pfxStyle.Render(p.prefix)

		var codePart string
		if i < len(highlighted) && highlighted[i] != "" {
			codePart = highlighted[i]
		} else {
			// Fallback: plain diff color (same style as prefix for full-row bg)
			codePart = pfxStyle.Render(p.code)
		}

		lines[i] = prefixStyled + codePart
	}
	return lines
}

// highlightCodeWithBg tokenizes code using Chroma and returns one ANSI-styled
// string per line, with each token rendered using both its syntax color AND the
// appropriate diff background. For lines with inline regions, changed characters
// get a brighter background to highlight the specific modification.
func highlightCodeWithBg(code, filePath string, parts []diffParts, inline map[int][]charRegion) []string {
	lexer := lexers.Match(filepath.Base(filePath))
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil
	}

	// Determine background colors for each line
	lineBgs := make([]lineBg, len(parts))
	for i := range parts {
		switch parts[i].marker {
		case '-':
			lineBgs[i] = lineBg{theme.ColorDiffRemoveBg, theme.ColorDiffRemoveInlineBg}
		case '+':
			lineBgs[i] = lineBg{theme.ColorDiffAddBg, theme.ColorDiffAddInlineBg}
		}
	}

	// Build token style cache to avoid per-token allocations
	styleCache := make(map[chroma.TokenType]color.Color)
	tokenColor := func(t chroma.TokenType) color.Color {
		if c, ok := styleCache[t]; ok {
			return c
		}
		c := tokenFgColor(t)
		styleCache[t] = c
		return c
	}

	// Render tokens with per-token foreground + line background
	var result []string
	var current strings.Builder
	lineIdx := 0
	lineOffset := 0 // byte offset within current line's code

	for _, token := range iterator.Tokens() {
		fg := tokenColor(token.Type)
		tokenParts := strings.Split(token.Value, "\n")
		for j, part := range tokenParts {
			if j > 0 {
				result = append(result, current.String())
				current.Reset()
				lineIdx++
				lineOffset = 0
			}
			if part == "" {
				continue
			}

			regions := inline[lineIdx]
			if len(regions) > 0 && lineIdx < len(lineBgs) {
				// Render with inline highlighting — split token at region boundaries
				renderTokenWithRegions(&current, part, fg, lineBgs[lineIdx], regions, lineOffset)
			} else {
				// Simple render — uniform background
				s := lipgloss.NewStyle().Foreground(fg)
				if lineIdx < len(lineBgs) && lineBgs[lineIdx].normal != nil {
					s = s.Background(lineBgs[lineIdx].normal)
				}
				current.WriteString(s.Render(part))
			}
			lineOffset += len(part)
		}
	}
	result = append(result, current.String())

	return result
}

// renderTokenWithRegions renders a token fragment, splitting it at inline
// region boundaries to apply the appropriate background (normal or inline)
// for each segment.
func renderTokenWithRegions(buf *strings.Builder, text string, fg color.Color, bg lineBg, regions []charRegion, offset int) {
	end := offset + len(text)
	pos := 0 // position within text

	for _, r := range regions {
		// Skip regions entirely before this token
		if r.end <= offset {
			continue
		}
		// Stop if we've passed this token
		if r.start >= end {
			break
		}

		// Clamp region to token bounds
		rStart := max(r.start-offset, pos)
		rEnd := min(r.end-offset, len(text))

		// Gap before this region (shouldn't happen with contiguous regions, but safe)
		if rStart > pos {
			s := lipgloss.NewStyle().Foreground(fg)
			if bg.normal != nil {
				s = s.Background(bg.normal)
			}
			buf.WriteString(s.Render(text[pos:rStart]))
		}

		// Render the region segment
		s := lipgloss.NewStyle().Foreground(fg)
		if r.changed && bg.inline != nil {
			s = s.Background(bg.inline)
		} else if bg.normal != nil {
			s = s.Background(bg.normal)
		}
		buf.WriteString(s.Render(text[rStart:rEnd]))
		pos = rEnd
	}

	// Remainder after last region
	if pos < len(text) {
		s := lipgloss.NewStyle().Foreground(fg)
		if bg.normal != nil {
			s = s.Background(bg.normal)
		}
		buf.WriteString(s.Render(text[pos:]))
	}
}

// tokenFgColor returns the foreground color for a Chroma token type.
func tokenFgColor(t chroma.TokenType) color.Color {
	p := theme.P

	switch t { //nolint:exhaustive // only styling known token types, default handles the rest
	// Comments
	case chroma.Comment, chroma.CommentSingle, chroma.CommentMultiline,
		chroma.CommentSpecial:
		return lipgloss.Color(p.Comment)
	case chroma.CommentPreproc:
		return lipgloss.Color(p.Cyan)

	// Keywords
	case chroma.Keyword, chroma.KeywordReserved, chroma.KeywordNamespace,
		chroma.KeywordDeclaration, chroma.KeywordConstant:
		return lipgloss.Color(p.Pink)
	case chroma.KeywordType:
		return lipgloss.Color(p.Cyan)

	// Operators
	case chroma.Operator, chroma.OperatorWord:
		return lipgloss.Color(p.Pink)

	// Names
	case chroma.NameFunction, chroma.NameClass, chroma.NameDecorator,
		chroma.NameAttribute:
		return lipgloss.Color(p.Green)
	case chroma.NameBuiltin, chroma.NameBuiltinPseudo:
		return lipgloss.Color(p.Cyan)
	case chroma.NameTag:
		return lipgloss.Color(p.Pink)
	case chroma.NameConstant:
		return lipgloss.Color(p.Purple)

	// Literals
	case chroma.LiteralString, chroma.LiteralStringDouble, chroma.LiteralStringSingle,
		chroma.LiteralStringBacktick, chroma.LiteralStringHeredoc,
		chroma.LiteralStringAffix, chroma.LiteralStringInterpol:
		return lipgloss.Color(p.Yellow)
	case chroma.LiteralStringEscape, chroma.LiteralStringRegex,
		chroma.LiteralStringSymbol:
		return lipgloss.Color(p.Purple)
	case chroma.LiteralNumber, chroma.LiteralNumberFloat,
		chroma.LiteralNumberHex, chroma.LiteralNumberInteger,
		chroma.LiteralNumberOct, chroma.LiteralNumberBin:
		return lipgloss.Color(p.Purple)

	// Punctuation
	case chroma.Punctuation:
		return lipgloss.Color(p.Fg)

	// Generic diff tokens
	case chroma.GenericDeleted:
		return lipgloss.Color(p.Pink)
	case chroma.GenericInserted:
		return lipgloss.Color(p.Green)
	case chroma.GenericSubheading:
		return lipgloss.Color(p.Comment)

	default:
		return lipgloss.Color(p.Fg)
	}
}
