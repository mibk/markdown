// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

import "strings"

// A Callout is a [Block] representing a block quote
// whose first line is a [!type] marker,
// as in a [GitHub alert].
//
// [GitHub alert]: https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#alerts
type Callout struct {
	Position
	Type   string  // letters between [! and ], as written
	Title  string  // rest of the marker line, verbatim
	Blocks []Block // content under the marker line
}

func (*Callout) Block() {}

func (b *Callout) printHTML(p *printer) {
	p.html(`<div class="markdown-alert markdown-alert-`, strings.ToLower(b.Type), `">`, "\n")
	p.html(`<p class="markdown-alert-title">`)
	if b.Title != "" {
		p.text(b.Title)
	} else {
		p.text(b.Type)
	}
	p.html("</p>\n")
	for _, c := range b.Blocks {
		c.printHTML(p)
	}
	p.html("</div>\n")
}

func (b *Callout) printMarkdown(p *printer) {
	if p.loose+p.tight == 0 || p.curLoose {
		p.maybeNL()
	}
	p.maybeQuoteNL('>')
	p.WriteString("> ")
	defer p.pop(p.push("> "))
	p.md("[!", b.Type, "]")
	if b.Title != "" {
		p.md(" ", b.Title)
	}
	if len(b.Blocks) > 0 {
		p.nl()
		p.markerEnd = p.buf.Len()
		printMarkdownBlocks(b.Blocks, p)
	}
}

// A calloutBuilder is a [blockBuilder] for a [Callout].
type calloutBuilder struct {
	typ   string
	title string
}

// startCallout is a [starter] for a [Callout].
func startCallout(p *parser, s line) (line, bool) {
	if !p.Callout {
		return s, false
	}
	t, ok := trimQuote(s)
	if !ok {
		return s, false
	}
	typ, title, ok := parseCalloutMarker(t.string())
	if !ok {
		return s, false
	}
	p.addBlock(&calloutBuilder{typ, title})
	t.spaces = 0
	t.skip(len(t.text) - t.i)
	return t, true
}

// parseCalloutMarker parses the [!type] title line that opens a callout.
func parseCalloutMarker(s string) (typ, title string, ok bool) {
	if !strings.HasPrefix(s, "[!") {
		return "", "", false
	}
	i := 2
	for i < len(s) && isLetter(s[i]) {
		i++
	}
	if i == 2 || i == len(s) || s[i] != ']' {
		return "", "", false
	}
	rest := s[i+1:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", "", false
	}
	return s[2:i], trimSpaceTab(rest), true
}

func (b *calloutBuilder) extend(p *parser, s line) (line, bool) {
	return trimQuote(s)
}

func (b *calloutBuilder) build(p *parser) Block {
	return &Callout{p.pos(), b.typ, b.title, p.blocks()}
}
