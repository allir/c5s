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

// diffPrefix extracts the diff marker, line number prefix, and code content
// from a formatted diff line like "- 42  code here" or "  code here".
func diffPrefix(line string) (marker byte, prefix, code string) {
	if len(line) < 2 {
		return ' ', line, ""
	}
	marker = line[0]
	// Find where the code content starts (after "X NNN  " or "X ")
	// The format is: marker + space + optional "NNN  " + code
	rest := line[1:]
	// Skip the space after the marker
	if len(rest) > 0 && rest[0] == ' ' {
		rest = rest[1:]
	}
	// Check for line number pattern: digits followed by two spaces
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i > 0 && i+2 <= len(rest) && rest[i] == ' ' && rest[i+1] == ' ' {
		prefix = line[:len(line)-len(rest)+i+2]
		code = rest[i+2:]
	} else {
		prefix = line[:2] // just "X " (marker + space)
		code = line[2:]
	}
	return marker, prefix, code
}

// renderDiffBlock takes a slice of consecutive diff entries (all sharing the
// same FilePath) and returns syntax-highlighted, styled lines.
func renderDiffBlock(entries []claude.TranscriptEntry) []string {
	if len(entries) == 0 {
		return nil
	}

	// Extract code lines for tokenization
	type parsed struct {
		marker byte
		prefix string
		code   string
	}
	parts := make([]parsed, len(entries))
	codeLines := make([]string, len(entries))
	for i, e := range entries {
		marker, pfx, code := diffPrefix(e.Content)
		parts[i] = parsed{marker, pfx, code}
		codeLines[i] = code
	}

	// Tokenize the code block as a whole
	codeBlock := strings.Join(codeLines, "\n")
	highlighted := highlightCode(codeBlock, entries[0].FilePath)

	// Build styled output lines
	lines := make([]string, len(entries))
	for i, p := range parts {
		var fg, bg color.Color
		switch p.marker {
		case '-':
			fg = theme.ColorDiffRemoveFg
			bg = theme.ColorDiffRemoveBg
		case '+':
			fg = theme.ColorDiffAddFg
			bg = theme.ColorDiffAddBg
		default:
			fg = theme.ColorMuted
			bg = nil
		}

		// Render prefix (marker + line number) with the diff color
		prefixStyled := lipgloss.NewStyle().Foreground(fg).Render("  " + p.prefix)

		var codePart string
		if i < len(highlighted) && highlighted[i] != "" {
			// Overlay syntax colors on diff background
			codePart = overlayBackground(highlighted[i], bg)
		} else {
			// Fallback: plain diff color
			style := lipgloss.NewStyle().Foreground(fg)
			if bg != nil {
				style = style.Background(bg)
			}
			codePart = style.Render(p.code)
		}

		lines[i] = prefixStyled + codePart
	}
	return lines
}

// highlightCode tokenizes code using Chroma and returns one ANSI-styled string
// per line. Returns nil if the language can't be detected.
func highlightCode(code, filePath string) []string {
	lexer := lexers.Match(filepath.Base(filePath))
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil
	}

	// Map Chroma token types to palette colors
	var result []string
	var current strings.Builder

	for _, token := range iterator.Tokens() {
		style := tokenStyle(token.Type)
		// Split multi-line tokens into per-line segments
		parts := strings.Split(token.Value, "\n")
		for j, part := range parts {
			if j > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			if part != "" {
				current.WriteString(style.Render(part))
			}
		}
	}
	// Flush last line
	result = append(result, current.String())

	return result
}

// tokenStyle returns a lipgloss style for a Chroma token type, using the
// active palette colors.
func tokenStyle(t chroma.TokenType) lipgloss.Style {
	p := theme.P
	s := lipgloss.NewStyle()

	switch t { //nolint:exhaustive // only styling known token types, default handles the rest
	// Comments
	case chroma.Comment, chroma.CommentSingle, chroma.CommentMultiline,
		chroma.CommentSpecial:
		return s.Foreground(lipgloss.Color(p.Comment))
	case chroma.CommentPreproc:
		return s.Foreground(lipgloss.Color(p.Cyan))

	// Keywords
	case chroma.Keyword, chroma.KeywordReserved, chroma.KeywordNamespace,
		chroma.KeywordDeclaration, chroma.KeywordConstant:
		return s.Foreground(lipgloss.Color(p.Pink))
	case chroma.KeywordType:
		return s.Foreground(lipgloss.Color(p.Cyan))

	// Operators
	case chroma.Operator, chroma.OperatorWord:
		return s.Foreground(lipgloss.Color(p.Pink))

	// Names
	case chroma.NameFunction, chroma.NameClass, chroma.NameDecorator,
		chroma.NameAttribute:
		return s.Foreground(lipgloss.Color(p.Green))
	case chroma.NameBuiltin, chroma.NameBuiltinPseudo:
		return s.Foreground(lipgloss.Color(p.Cyan))
	case chroma.NameTag:
		return s.Foreground(lipgloss.Color(p.Pink))
	case chroma.NameConstant:
		return s.Foreground(lipgloss.Color(p.Purple))

	// Literals
	case chroma.LiteralString, chroma.LiteralStringDouble, chroma.LiteralStringSingle,
		chroma.LiteralStringBacktick, chroma.LiteralStringHeredoc,
		chroma.LiteralStringAffix, chroma.LiteralStringInterpol:
		return s.Foreground(lipgloss.Color(p.Yellow))
	case chroma.LiteralStringEscape, chroma.LiteralStringRegex,
		chroma.LiteralStringSymbol:
		return s.Foreground(lipgloss.Color(p.Purple))
	case chroma.LiteralNumber, chroma.LiteralNumberFloat,
		chroma.LiteralNumberHex, chroma.LiteralNumberInteger,
		chroma.LiteralNumberOct, chroma.LiteralNumberBin:
		return s.Foreground(lipgloss.Color(p.Purple))

	// Punctuation
	case chroma.Punctuation:
		return s.Foreground(lipgloss.Color(p.Fg))

	// Generic diff tokens (if Chroma emits them)
	case chroma.GenericDeleted:
		return s.Foreground(lipgloss.Color(p.Pink))
	case chroma.GenericInserted:
		return s.Foreground(lipgloss.Color(p.Green))
	case chroma.GenericEmph:
		return s.Italic(true)
	case chroma.GenericStrong:
		return s.Bold(true)
	case chroma.GenericSubheading:
		return s.Foreground(lipgloss.Color(p.Comment))

	default:
		return s.Foreground(lipgloss.Color(p.Fg))
	}
}

// overlayBackground takes an already ANSI-styled string and wraps each segment
// with a background color. Since we can't easily modify existing ANSI sequences,
// we set the background on the overall style and let the terminal handle layering.
func overlayBackground(styledText string, bg color.Color) string {
	if bg == nil {
		return styledText
	}
	// Wrap the pre-styled text in a background color span.
	// lipgloss will add the background escape codes around the content.
	return lipgloss.NewStyle().Background(bg).Render(styledText)
}
