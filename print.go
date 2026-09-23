// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

import "bytes"

const (
	writeMarkdown = iota
	writeHTML
	writeText
)

type printer struct {
	writeMode   int
	buf         bytes.Buffer
	prefix      []byte
	prefixOld   []byte
	prefixOlder []byte
	trimLimit   int
	// prevBlock is the sibling block printed just before the one
	// being printed, within the current container; nil for the first.
	prevBlock Block
	// refEnd is the offset just past the ] of a shortcut reference link,
	// valid only while nothing else has been printed after it.
	refEnd int
	// markerEnd is the offset just past the prefix of the line
	// after a callout marker, which no block there can run into.
	markerEnd int
	listOut
	footnotes    map[*Footnote]*printedNote
	footnotelist []*printedNote
}

type listOut struct {
	bullet rune
	num    int
	loose  int
	tight  int
	// curLoose is the looseness of the innermost list being printed.
	// The loose/tight counters aggregate the whole stack, but block
	// separation within an item is governed by the item's own list:
	// a blank line inside a tight list nested in a loose one would
	// make the tight list loose, changing the document.
	curLoose bool
}

func (w *printer) WriteStrings(list ...string) {
	for _, s := range list {
		w.WriteString(s)
	}
}

func cutLastNL(text []byte) (prefix, last []byte) {
	i := bytes.LastIndexByte(text, '\n')
	if i < 0 {
		return nil, text
	}
	return text[:i], text[i+1:]
}

func (b *printer) noTrim() {
	b.trimLimit = len(b.buf.Bytes())
}

func (b *printer) nl() {
	text := b.buf.Bytes()
	for len(text) > b.trimLimit && text[len(text)-1] == ' ' {
		text = text[:len(text)-1]
	}
	b.buf.Truncate(len(text))

	b.buf.WriteByte('\n')
	b.buf.Write(b.prefix)
	b.prefixOlder, b.prefixOld = b.prefixOld, b.prefix
}

func (b *printer) maybeNL() bool {
	// Starting a new block that may need a blank line before it
	// to avoid being mixed into a previous block
	// as paragraph continuation text.
	//
	// If the prefix on the current line (all of cur)
	// is the same as the current continuation prefix
	// (not first line of a list item)
	// and the previous line started with the same prefix,
	// then we need a blank line to avoid looking like
	// paragraph continuation text.
	before, cur := cutLastNL(b.buf.Bytes())
	before, prev := cutLastNL(before)
	if b.buf.Len() > 0 && b.buf.Len() != b.markerEnd && bytes.Equal(cur, b.prefix) && bytes.HasPrefix(prev, b.prefix) {
		b.nl()
		return true
	}
	return true
}

func ToHTML(b Block) string {
	var p printer
	p.writeMode = writeHTML
	b.printHTML(&p)
	printFootnoteHTML(&p)
	return p.buf.String()
}

func Format(b Block) string {
	var p printer
	b.printMarkdown(&p)
	return p.buf.String()
}

var closeP = []byte("</p>\n")

func (b *printer) eraseCloseP() bool {
	if bytes.HasSuffix(b.buf.Bytes(), closeP) {
		b.buf.Truncate(b.buf.Len() - len(closeP))
		return true
	}
	return false
}

func (b *printer) maybeQuoteNL(quote byte) bool {
	// Starting a new quote block.
	// Make sure it doesn't look like it is part of a preceding quote block.
	before, cur := cutLastNL(b.buf.Bytes())
	before, prev := cutLastNL(before)
	if len(prev) >= len(cur)+1 && bytes.HasPrefix(prev, cur) && prev[len(cur)] == quote {
		b.nl()
		return true
	}
	return false
}

// attaches reports whether writing c next would attach to the ]
// of a shortcut reference link just printed
// and change what the text parses back as.
func (p *printer) attaches(c byte) bool {
	if p.refEnd == 0 || p.refEnd != p.buf.Len() {
		return false
	}
	switch c {
	case '(', '[':
		// [a](b) and [a][b] would swallow the text as a destination or a label.
		return true
	case ':':
		return p.labelLine()
	}
	return false
}

// labelLine reports whether all that is printed on the current line
// is a link label, which a : written next would turn into
// a link reference definition.
func (p *printer) labelLine() bool {
	_, line := cutLastNL(p.buf.Bytes())
	line = bytes.TrimPrefix(line, p.prefix)
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	if i > 3 || i == len(line) || line[i] != '[' {
		return false
	}
	for i++; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case ']':
			return i == len(line)-1
		}
	}
	return false
}

func (b *printer) WriteByte(c byte) error {
	if c == '\n' {
		panic("Write \\n")
	}
	return b.buf.WriteByte(c)
}

func (p *printer) Write(text []byte) (int, error) {
	if p.writeMode == writeMarkdown {
		for i := range text {
			if text[i] == '\n' {
				panic("Write \\n")
			}
		}
	}
	return p.buf.Write(text)
}

func (p *printer) html(list ...string) {
	if p.writeMode != writeHTML {
		panic("raw HTML in non-HTML output")
	}
	for _, s := range list {
		p.buf.WriteString(s)
	}
}

func (p *printer) text(list ...string) {
	if p.writeMode == writeHTML {
		for _, s := range list {
			htmlEscaper.WriteString(&p.buf, s)
		}
		return
	}
	for _, s := range list {
		p.buf.WriteString(s)
	}

}

func (p *printer) md(list ...string) {
	if p.writeMode != writeMarkdown {
		panic("markdown in non-markdown output")
	}
	for _, s := range list {
		p.buf.WriteString(s)
	}
}

func (b *printer) WriteString(s string) (int, error) {
	if b.writeMode == writeMarkdown {
		for i := 0; i < len(s); i++ {
			if s[i] == '\n' {
				panic("Write \\n")
			}
		}
	}
	return b.buf.WriteString(s)
}

func (b *printer) push(s string) int {
	n := len(b.prefix)
	b.prefix = append(b.prefix, s...)
	return n
}

func (b *printer) pop(n int) {
	b.prefix = b.prefix[:n]
}
