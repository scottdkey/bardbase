// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package parser

import (
	"encoding/xml"
	"strings"
)

// XMLNode is a lightweight DOM node for parsing mixed-content XML like Schmidt lexicon entries.
type XMLNode struct {
	Name     string
	Attrs    map[string]string
	Children []*XMLNode
	Text     string // direct text content (before first child)
	Tail     string // text after closing tag (belongs to parent)
}

// ParseXML parses XML bytes into a tree of XMLNodes.
func ParseXML(data []byte) (*XMLNode, error) {
	decoder := newXMLDecoder(data)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	root := &XMLNode{Name: "root", Attrs: make(map[string]string)}
	stack := []*XMLNode{root}

	for {
		tok, err := decoder.Token()
		if err != nil {
			break // EOF or error
		}

		switch t := tok.(type) {
		case xml.StartElement:
			node := &XMLNode{
				Name:  t.Name.Local,
				Attrs: make(map[string]string),
			}
			for _, attr := range t.Attr {
				key := attr.Name.Local
				if attr.Name.Space != "" {
					key = attr.Name.Space + ":" + key
				}
				node.Attrs[key] = attr.Value
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
			stack = append(stack, node)

		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			text := string(t)
			current := stack[len(stack)-1]
			if len(current.Children) == 0 {
				current.Text += text
			} else {
				// Text after a child element = tail of last child
				lastChild := current.Children[len(current.Children)-1]
				lastChild.Tail += text
			}
		}
	}

	return root, nil
}

// newXMLDecoder creates an xml.Decoder from byte data.
func newXMLDecoder(data []byte) *xml.Decoder {
	return xml.NewDecoder(strings.NewReader(string(data)))
}

// GetText returns all text content from a node and its descendants, recursively.
func (n *XMLNode) GetText() string {
	var b strings.Builder
	n.collectText(&b)
	return b.String()
}

func (n *XMLNode) collectText(b *strings.Builder) {
	b.WriteString(n.Text)
	for _, child := range n.Children {
		child.collectText(b)
		b.WriteString(child.Tail)
	}
}

// GetTextExcluding returns text content, skipping elements with names in the exclude set.
// Tail text of excluded elements is still included (it belongs to the parent context).
func (n *XMLNode) GetTextExcluding(excludeNames ...string) string {
	exclude := make(map[string]bool, len(excludeNames))
	for _, name := range excludeNames {
		exclude[name] = true
	}
	var b strings.Builder
	n.collectTextExcluding(&b, exclude)
	return b.String()
}

func (n *XMLNode) collectTextExcluding(b *strings.Builder, exclude map[string]bool) {
	b.WriteString(n.Text)
	for _, child := range n.Children {
		if exclude[child.Name] {
			// Skip the element's content AND its tail (trailing punctuation between refs)
		} else {
			child.collectTextExcluding(b, exclude)
			b.WriteString(child.Tail)
		}
	}
}

// Attr returns the value of the named attribute, or empty string if not found.
func (n *XMLNode) Attr(key string) string {
	return n.Attrs[key]
}

// Find returns the first descendant with the given element name, or nil.
func (n *XMLNode) Find(name string) *XMLNode {
	for _, child := range n.Children {
		if child.Name == name {
			return child
		}
		if found := child.Find(name); found != nil {
			return found
		}
	}
	return nil
}

// FindAll returns all descendants with the given element name.
func (n *XMLNode) FindAll(name string) []*XMLNode {
	var results []*XMLNode
	n.findAll(name, &results)
	return results
}

func (n *XMLNode) findAll(name string, results *[]*XMLNode) {
	for _, child := range n.Children {
		if child.Name == name {
			*results = append(*results, child)
		}
		child.findAll(name, results)
	}
}

// ContainsChild checks if a node is a descendant of this node.
func (n *XMLNode) ContainsChild(target *XMLNode) bool {
	for _, child := range n.Children {
		if child == target {
			return true
		}
		if child.ContainsChild(target) {
			return true
		}
	}
	return false
}
