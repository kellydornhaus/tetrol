package xmlui

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Node is a parsed XML element node.
type Node struct {
	Name     string
	Attrs    map[string]string
	Text     string // direct text content (trimmed)
	Children []*Node
	Parent   *Node

	baseAttrs map[string]string
	classSet  map[string]struct{}
}

// parseXML parses the XML reader into a Node tree.
func parseXML(r io.Reader) (*Node, error) {
	root, _, err := parseXMLWithStyles(r)
	return root, err
}

func parseXMLWithStyles(r io.Reader) (*Node, []string, error) {
	dec := xml.NewDecoder(r)
	var stack []*Node
	var root *Node
	var styles []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &Node{Name: t.Name.Local, Attrs: make(map[string]string)}
			for _, a := range t.Attr {
				n.Attrs[a.Name.Local] = a.Value
			}
			n.baseAttrs = cloneAttrs(n.Attrs)
			n.initClassSet()
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, nil, fmt.Errorf("unexpected end element: %s", t.Name.Local)
			}
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				root = n
			} else {
				parent := stack[len(stack)-1]
				n.Parent = parent
				parent.Children = append(parent.Children, n)
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			s := string([]byte(t))
			if strings.TrimSpace(s) != "" {
				cur := stack[len(stack)-1]
				cur.Text += s
			}
		case xml.ProcInst:
			if strings.EqualFold(t.Target, "xml-stylesheet") {
				attrs := parsePIAttributes(string(t.Inst))
				typ := strings.ToLower(attrs["type"])
				if typ == "" || typ == "text/css" {
					if href := strings.TrimSpace(attrs["href"]); href != "" {
						styles = append(styles, href)
					}
				}
			}
		}
	}
	if root == nil {
		return nil, nil, fmt.Errorf("empty XML")
	}
	// Trim text fields
	var trimFn func(*Node)
	trimFn = func(n *Node) {
		n.Text = strings.TrimSpace(n.Text)
		for _, c := range n.Children {
			trimFn(c)
		}
	}
	trimFn(root)
	return root, styles, nil
}

func cloneAttrs(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func parsePIAttributes(s string) map[string]string {
	out := make(map[string]string)
	rest := strings.TrimSpace(s)
	for len(rest) > 0 {
		eq := strings.IndexByte(rest, '=')
		if eq <= 0 {
			break
		}
		key := strings.TrimSpace(rest[:eq])
		rest = strings.TrimSpace(rest[eq+1:])
		if len(rest) == 0 {
			break
		}
		var val string
		if rest[0] == '"' || rest[0] == '\'' {
			quote := rest[0]
			rest = rest[1:]
			end := strings.IndexByte(rest, quote)
			if end < 0 {
				val = rest
				rest = ""
			} else {
				val = rest[:end]
				rest = rest[end+1:]
			}
		} else {
			end := strings.IndexAny(rest, " \t\r\n")
			if end < 0 {
				val = rest
				rest = ""
			} else {
				val = rest[:end]
				rest = rest[end+1:]
			}
		}
		if key != "" {
			out[strings.ToLower(key)] = val
		}
		rest = strings.TrimSpace(rest)
	}
	return out
}

func (n *Node) initClassSet() {
	if n.classSet == nil {
		n.classSet = make(map[string]struct{})
	}
	if cls, ok := n.Attrs["class"]; ok {
		for _, c := range strings.Fields(cls) {
			if c != "" {
				n.classSet[c] = struct{}{}
			}
		}
	}
}

func (n *Node) resetAttributes() {
	n.Attrs = make(map[string]string, len(n.baseAttrs)+1)
	for k, v := range n.baseAttrs {
		if k == "class" {
			continue
		}
		n.Attrs[k] = v
	}
	if cls := n.classString(); cls != "" {
		n.Attrs["class"] = cls
	}
}

func (n *Node) resetRecursive() {
	n.resetAttributes()
	for _, ch := range n.Children {
		ch.resetRecursive()
	}
}

func (n *Node) classString() string {
	if len(n.classSet) == 0 {
		return ""
	}
	var classes []string
	for c := range n.classSet {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	return strings.Join(classes, " ")
}

func (n *Node) updateClassAttr() {
	if cls := n.classString(); cls != "" {
		n.Attrs["class"] = cls
	} else {
		delete(n.Attrs, "class")
	}
}

func cloneNode(src *Node) *Node {
	if src == nil {
		return nil
	}
	dst := &Node{
		Name:      src.Name,
		Attrs:     cloneAttrs(src.Attrs),
		Text:      src.Text,
		baseAttrs: cloneAttrs(src.baseAttrs),
	}
	if len(src.classSet) > 0 {
		dst.classSet = make(map[string]struct{}, len(src.classSet))
		for cls := range src.classSet {
			dst.classSet[cls] = struct{}{}
		}
	} else {
		dst.classSet = make(map[string]struct{})
	}
	if len(src.Children) > 0 {
		dst.Children = make([]*Node, len(src.Children))
		for i, ch := range src.Children {
			childClone := cloneNode(ch)
			if childClone != nil {
				childClone.Parent = dst
			}
			dst.Children[i] = childClone
		}
	}
	return dst
}

func (n *Node) AddClass(class string) bool {
	class = strings.TrimSpace(class)
	if class == "" {
		return false
	}
	if n.classSet == nil {
		n.classSet = make(map[string]struct{})
	}
	if _, exists := n.classSet[class]; exists {
		return false
	}
	n.classSet[class] = struct{}{}
	return true
}

func (n *Node) RemoveClass(class string) bool {
	class = strings.TrimSpace(class)
	if class == "" || len(n.classSet) == 0 {
		return false
	}
	if _, exists := n.classSet[class]; !exists {
		return false
	}
	delete(n.classSet, class)
	return true
}

func (n *Node) HasClass(class string) bool {
	if len(n.classSet) == 0 {
		return false
	}
	_, exists := n.classSet[strings.TrimSpace(class)]
	return exists
}

func (n *Node) Classes() []string {
	if len(n.classSet) == 0 {
		return nil
	}
	classes := make([]string, 0, len(n.classSet))
	for c := range n.classSet {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	return classes
}
