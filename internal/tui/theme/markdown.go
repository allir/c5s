package theme

import "charm.land/glamour/v2/ansi"

// MarkdownStyleConfig builds a glamour style config from the active palette.
// This is a function (not a var) so it picks up the current value of P.
func MarkdownStyleConfig() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       ptr(P.Fg),
			},
			Margin: ptr(uint(0)),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(P.Comment),
			},
			Indent:      ptr(uint(1)),
			IndentToken: ptr("│ "),
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: ptr(P.Fg),
				},
			},
			LevelIndent: 2,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       ptr(P.Pink),
				Bold:        ptr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold: ptr(true),
			},
		},
		H2: ansi.StyleBlock{},
		H3: ansi.StyleBlock{},
		H4: ansi.StyleBlock{},
		H5: ansi.StyleBlock{},
		H6: ansi.StyleBlock{},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: ptr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: ptr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold: ptr(true),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  ptr(P.Comment),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
			Color:       ptr(P.Purple),
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "[✓] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     ptr(P.Cyan),
			Underline: ptr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: ptr(P.Green),
		},
		Image: ansi.StylePrimitive{
			Color:     ptr(P.Cyan),
			Underline: ptr(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  ptr(P.Green),
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ptr(P.Cyan),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: ptr(P.Fg),
				},
				Margin: ptr(uint(2)),
			},
			Chroma: &ansi.Chroma{
				Text:                ansi.StylePrimitive{Color: ptr(P.Fg)},
				Error:               ansi.StylePrimitive{Color: ptr(P.Fg), BackgroundColor: ptr(P.Pink)},
				Comment:             ansi.StylePrimitive{Color: ptr(P.Comment)},
				CommentPreproc:      ansi.StylePrimitive{Color: ptr(P.Cyan)},
				Keyword:             ansi.StylePrimitive{Color: ptr(P.Pink)},
				KeywordReserved:     ansi.StylePrimitive{Color: ptr(P.Pink)},
				KeywordNamespace:    ansi.StylePrimitive{Color: ptr(P.Pink)},
				KeywordType:         ansi.StylePrimitive{Color: ptr(P.Cyan)},
				Operator:            ansi.StylePrimitive{Color: ptr(P.Pink)},
				Punctuation:         ansi.StylePrimitive{Color: ptr(P.Fg)},
				Name:                ansi.StylePrimitive{Color: ptr(P.Green)},
				NameConstant:        ansi.StylePrimitive{Color: ptr(P.Purple)},
				NameBuiltin:         ansi.StylePrimitive{Color: ptr(P.Cyan)},
				NameTag:             ansi.StylePrimitive{Color: ptr(P.Pink)},
				NameAttribute:       ansi.StylePrimitive{Color: ptr(P.Green)},
				NameClass:           ansi.StylePrimitive{Color: ptr(P.Green)},
				NameDecorator:       ansi.StylePrimitive{Color: ptr(P.Green)},
				NameFunction:        ansi.StylePrimitive{Color: ptr(P.Green)},
				LiteralNumber:       ansi.StylePrimitive{Color: ptr(P.Purple)},
				LiteralString:       ansi.StylePrimitive{Color: ptr(P.Yellow)},
				LiteralStringEscape: ansi.StylePrimitive{Color: ptr(P.Purple)},
				GenericDeleted:      ansi.StylePrimitive{Color: ptr(P.Pink)},
				GenericEmph:         ansi.StylePrimitive{Italic: ptr(true)},
				GenericInserted:     ansi.StylePrimitive{Color: ptr(P.Green)},
				GenericStrong:       ansi.StylePrimitive{Bold: ptr(true)},
				GenericSubheading:   ansi.StylePrimitive{Color: ptr(P.Comment)},
				Background:          ansi.StylePrimitive{BackgroundColor: ptr(P.Bg)},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n🠶 ",
		},
	}
}
