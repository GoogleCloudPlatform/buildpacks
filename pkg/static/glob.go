// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package static

import (
	"fmt"
	"regexp"
	"strings"
)

// GlobToRegex converts a Firebase source glob pattern into an Nginx-compatible regex.
// It supports **, *, ?, @(a|b), [a-z], and extglobs *(a|b), +(a|b), ?(a|b), !(a|b).
func GlobToRegex(glob string) (string, error) {
	p := &globParser{
		runes: []rune(glob),
	}
	p.result.WriteString("^")

	// Prepend / if the glob doesn't start with / or **
	if !strings.HasPrefix(glob, "/") && !strings.HasPrefix(glob, "**") {
		p.result.WriteString("/")
	}

	for p.pos < len(p.runes) {
		p.step()
	}

	if len(p.activeGroups) > 0 {
		return "", fmt.Errorf("unbalanced glob group in pattern: %s", glob)
	}

	p.result.WriteString("$")
	return p.result.String(), nil
}

type globParser struct {
	runes        []rune
	pos          int
	result       strings.Builder
	activeGroups []rune
}

func (p *globParser) peek(offset int) rune {
	if p.pos+offset < len(p.runes) {
		return p.runes[p.pos+offset]
	}
	return 0
}

func (p *globParser) consume() rune {
	r := p.runes[p.pos]
	p.pos++
	return r
}

func (p *globParser) step() {
	r := p.peek(0)

	// Lookahead for /**/ in the middle of a pattern
	if r == '/' && p.peek(1) == '*' && p.peek(2) == '*' && p.peek(3) == '/' {
		p.result.WriteString("/(?:.*/)?")
		p.pos += 4
		return
	}
	// Lookahead for /** at the end of the string
	if r == '/' && p.peek(1) == '*' && p.peek(2) == '*' && p.pos+3 == len(p.runes) {
		p.result.WriteString("(?:/.*)?")
		p.pos += 3
		return
	}

	switch r {
	case '\\':
		p.parseEscape()
	case '@', '*', '+', '?', '!':
		p.parseWildcardOrGroup(r)
	case '|':
		p.parsePipe()
	case ')':
		p.parseGroupClose()
	case '[':
		p.parseBracketClass()
	case ':':
		if !p.parseColonParam() {
			p.result.WriteRune(p.consume())
		}
	case '.', '^', '$', ']', '{', '}', '(':
		p.result.WriteString(regexp.QuoteMeta(string(p.consume())))
	default:
		p.result.WriteRune(p.consume())
	}
}

func (p *globParser) parseColonParam() bool {
	start := p.pos + 1
	i := start
	for i < len(p.runes) && isIdentRune(p.runes[i]) {
		i++
	}
	if i == start {
		return false
	}
	name := string(p.runes[start:i])
	p.pos = i
	isSplat := false
	if p.peek(0) == '*' || p.peek(0) == '+' {
		isSplat = true
		p.pos++
	}
	if isSplat {
		p.result.WriteString(fmt.Sprintf("(?P<%s>.+)", name))
	} else {
		p.result.WriteString(fmt.Sprintf("(?P<%s>[^/]+)", name))
	}
	return true
}

func isIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func (p *globParser) parseEscape() {
	p.consume() // consume '\'
	if p.pos < len(p.runes) {
		p.result.WriteString(regexp.QuoteMeta(string(p.consume())))
	} else {
		p.result.WriteString(`\\`)
	}
}

func (p *globParser) parseWildcardOrGroup(r rune) {
	p.consume() // consume r
	if p.peek(0) == '(' {
		p.consume() // consume '('
		p.activeGroups = append(p.activeGroups, r)
		if r == '!' {
			p.result.WriteString("(?!(?:")
		} else {
			p.result.WriteString("(?:")
		}
		return
	}

	switch r {
	case '*':
		if p.peek(0) == '*' {
			p.consume()
			p.result.WriteString(".*")
		} else {
			p.result.WriteString("[^/]*")
		}
	case '?':
		p.result.WriteString("[^/]")
	case '@':
		p.result.WriteString("@")
	case '+':
		p.result.WriteString(`\+`)
	case '!':
		p.result.WriteString("!")
	}
}

func (p *globParser) parsePipe() {
	p.consume()
	if len(p.activeGroups) > 0 {
		p.result.WriteString("|")
	} else {
		p.result.WriteString(`\|`)
	}
}

func (p *globParser) parseGroupClose() {
	p.consume()
	if len(p.activeGroups) == 0 {
		p.result.WriteString(`\)`)
		return
	}

	last := p.activeGroups[len(p.activeGroups)-1]
	p.activeGroups = p.activeGroups[:len(p.activeGroups)-1]
	switch last {
	case '!':
		p.result.WriteString(`)(?:/|$))[^/]*`)
	default:
		p.result.WriteRune(')')
		switch last {
		case '*':
			p.result.WriteRune('*')
		case '+':
			p.result.WriteRune('+')
		case '?':
			p.result.WriteRune('?')
		}
	}
}

func (p *globParser) parseBracketClass() {
	p.consume() // consume '['
	closeIdx := findClosingBracket(p.runes, p.pos)
	if closeIdx == -1 || closeIdx == p.pos {
		// empty [] or unbalanced [ is treated as literal
		p.result.WriteString(`\[`)
		return
	}

	p.result.WriteRune('[')
	content := p.runes[p.pos:closeIdx]
	if len(content) > 0 && (content[0] == '!' || content[0] == '^') {
		p.result.WriteString(`^/`)
		content = content[1:]
	}
	p.result.WriteString(string(content))
	p.result.WriteRune(']')
	p.pos = closeIdx + 1
}

func findClosingBracket(runes []rune, start int) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			i++
			continue
		}
		if runes[i] == ']' {
			return i
		}
	}
	return -1
}
