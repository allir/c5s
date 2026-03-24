package theme

import "charm.land/glamour/v2/ansi"

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func uintPtr(v uint) *uint       { return &v }

// MonokaiStyleConfig is a Monokai-inspired glamour style for markdown rendering.
var MonokaiStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
			Color:       stringPtr("#F8F8F2"),
		},
		Margin: uintPtr(0),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: stringPtr("#75715E"),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr("│ "),
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("#F8F8F2"),
			},
		},
		LevelIndent: 2,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       stringPtr("#F92672"),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "# ",
			Bold:   boolPtr(true),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "###### ",
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
	},
	Emph: ansi.StylePrimitive{
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold: boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  stringPtr("#75715E"),
		Format: "\n--------\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       stringPtr("#AE81FF"),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:     stringPtr("#66D9EF"),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: stringPtr("#A6E22E"),
	},
	Image: ansi.StylePrimitive{
		Color:     stringPtr("#66D9EF"),
		Underline: boolPtr(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  stringPtr("#A6E22E"),
		Format: "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: stringPtr("#66D9EF"),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("#F8F8F2"),
			},
			Margin: uintPtr(2),
		},
		Chroma: &ansi.Chroma{
			Text: ansi.StylePrimitive{
				Color: stringPtr("#F8F8F2"),
			},
			Error: ansi.StylePrimitive{
				Color:           stringPtr("#F8F8F2"),
				BackgroundColor: stringPtr("#F92672"),
			},
			Comment: ansi.StylePrimitive{
				Color: stringPtr("#75715E"),
			},
			CommentPreproc: ansi.StylePrimitive{
				Color: stringPtr("#66D9EF"),
			},
			Keyword: ansi.StylePrimitive{
				Color: stringPtr("#F92672"),
			},
			KeywordReserved: ansi.StylePrimitive{
				Color: stringPtr("#F92672"),
			},
			KeywordNamespace: ansi.StylePrimitive{
				Color: stringPtr("#F92672"),
			},
			KeywordType: ansi.StylePrimitive{
				Color: stringPtr("#66D9EF"),
			},
			Operator: ansi.StylePrimitive{
				Color: stringPtr("#F92672"),
			},
			Punctuation: ansi.StylePrimitive{
				Color: stringPtr("#F8F8F2"),
			},
			Name: ansi.StylePrimitive{
				Color: stringPtr("#A6E22E"),
			},
			NameConstant: ansi.StylePrimitive{
				Color: stringPtr("#AE81FF"),
			},
			NameBuiltin: ansi.StylePrimitive{
				Color: stringPtr("#66D9EF"),
			},
			NameTag: ansi.StylePrimitive{
				Color: stringPtr("#F92672"),
			},
			NameAttribute: ansi.StylePrimitive{
				Color: stringPtr("#A6E22E"),
			},
			NameClass: ansi.StylePrimitive{
				Color: stringPtr("#A6E22E"),
			},
			NameDecorator: ansi.StylePrimitive{
				Color: stringPtr("#A6E22E"),
			},
			NameFunction: ansi.StylePrimitive{
				Color: stringPtr("#A6E22E"),
			},
			LiteralNumber: ansi.StylePrimitive{
				Color: stringPtr("#AE81FF"),
			},
			LiteralString: ansi.StylePrimitive{
				Color: stringPtr("#E6DB74"),
			},
			LiteralStringEscape: ansi.StylePrimitive{
				Color: stringPtr("#AE81FF"),
			},
			GenericDeleted: ansi.StylePrimitive{
				Color: stringPtr("#F92672"),
			},
			GenericEmph: ansi.StylePrimitive{
				Italic: boolPtr(true),
			},
			GenericInserted: ansi.StylePrimitive{
				Color: stringPtr("#A6E22E"),
			},
			GenericStrong: ansi.StylePrimitive{
				Bold: boolPtr(true),
			},
			GenericSubheading: ansi.StylePrimitive{
				Color: stringPtr("#75715E"),
			},
			Background: ansi.StylePrimitive{
				BackgroundColor: stringPtr("#272822"),
			},
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
