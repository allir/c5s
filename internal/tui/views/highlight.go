package views

import (
	"image/color"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

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

	// Tokenize the code block as a whole
	codeBlock := strings.Join(codeLines, "\n")
	highlighted := highlightCodeWithBg(codeBlock, entries[0].FilePath, entries, parts)

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
// appropriate diff background. This avoids the ANSI reset problem of wrapping
// pre-styled text with a background color.
func highlightCodeWithBg(code, filePath string, entries []claude.TranscriptEntry, parts []diffParts) []string {
	lexer := lexers.Match(filepath.Base(filePath))
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil
	}

	// Determine background color for each line
	lineBgs := make([]color.Color, len(entries))
	for i := range parts {
		switch parts[i].marker {
		case '-':
			lineBgs[i] = theme.ColorDiffRemoveBg
		case '+':
			lineBgs[i] = theme.ColorDiffAddBg
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

	for _, token := range iterator.Tokens() {
		fg := tokenColor(token.Type)
		tokenParts := strings.Split(token.Value, "\n")
		for j, part := range tokenParts {
			if j > 0 {
				result = append(result, current.String())
				current.Reset()
				lineIdx++
			}
			if part != "" {
				s := lipgloss.NewStyle().Foreground(fg)
				if lineIdx < len(lineBgs) && lineBgs[lineIdx] != nil {
					s = s.Background(lineBgs[lineIdx])
				}
				current.WriteString(s.Render(part))
			}
		}
	}
	result = append(result, current.String())

	return result
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
