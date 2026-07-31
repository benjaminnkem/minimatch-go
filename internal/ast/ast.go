package ast

import (
	"github.com/tochison/minimatch/internal/scan"
)

// AST is the extglob syntax tree for a single path-segment pattern.
//
// It corresponds to the TypeScript minimatch AST class after #parseAST /
// fromGlob. Call Flatten to apply adopt/usurp rewrites (#flatten) before
// fillNegs and toRegExpSource. Construction and Flatten do not match paths
// or compile regular expressions.
//
// # Node kinds
//
// There is one Go type for all nodes. Kind is determined by Type:
//
//   - List node (Type == 0): an ordered sequence of text runs and/or child
//     AST nodes. The root is always a list node. Each extglob alternative
//     body is also a list node. TypeScript: type === null.
//
//   - Extglob node (Type is !, ?, +, *, or @): a choice among alternatives.
//     Parts are only list-node children (one per alternative), never raw
//     text. TypeScript: type === '!' | '?' | '+' | '*' | '@'.
//
// # Part entries
//
// List-node Parts alternate freely:
//
//   - Text parts: raw pattern text (may contain "*", "?", character classes,
//     escapes). Empty strings are never stored (TypeScript push skips ”).
//
//   - Child AST parts: nested extglob nodes or (after unfinished demotion)
//     list nodes holding demoted literal text.
//
// Extglob-node Parts are always child list nodes (alternatives), including
// empty alternatives.
//
// # EmptyExt
//
// EmptyExt is true when the extglob closed with an empty first body and no
// prior alternatives were flushed yet (TypeScript #emptyExt for patterns
// like "*()"). Used later for regexp emission of empty !() / *().
//
// Call Flatten then FillNegs on the root before compiling to a regular
// expression (same order as TypeScript toRegExpSource).
type AST struct {
	// Type is the extglob operator, or 0 for a list node (TS type === null).
	Type scan.ExtglobType

	// Parts is the ordered content of this node (see package comment).
	Parts []ASTPart

	// EmptyExt marks an extglob that closed with an empty body (e.g. "*()").
	EmptyExt bool

	// Options are the parse options from the root (shared by the tree).
	Options Config

	parent      *AST
	parentIndex int
	root        *AST
	id          int

	// Root-only bookkeeping for FillNegs (TypeScript #negs / #filledNegs).
	negs       []*AST
	filledNegs bool

	// Compilation state (TypeScript #hasMagic / #uflag), updated by ToRegExpSource.
	hasMagic bool
	uFlag    bool
}

// ASTPart is one entry in an AST node’s Parts slice.
//
// Exactly one of Text or Node is used:
//   - Node == nil: Text is a non-empty raw string (except demotion may use any string)
//   - Node != nil: child AST; Text is ignored
type ASTPart struct {
	Text string
	Node *AST
}

// IsText reports whether p is a raw text part.
func (p ASTPart) IsText() bool { return p.Node == nil }

// IsList reports whether n is a list node (TypeScript type === null).
func (n *AST) IsList() bool { return n != nil && n.Type == 0 }

// IsExtglob reports whether n is an extglob node.
func (n *AST) IsExtglob() bool { return n != nil && n.Type != 0 }

// Parent returns the parent node, if any.
func (n *AST) Parent() *AST {
	if n == nil {
		return nil
	}
	return n.parent
}

// Root returns the root of the tree.
func (n *AST) Root() *AST {
	if n == nil {
		return nil
	}
	return n.root
}

// Depth returns the depth of this node (root is 0), matching TS AST.depth.
func (n *AST) Depth() int {
	if n == nil || n.parent == nil {
		return 0
	}
	return n.parent.Depth() + 1
}

// ID returns a stable per-tree identity for debugging (like TS AST.id).
func (n *AST) ID() int {
	if n == nil {
		return 0
	}
	return n.id
}

// ParseGlob parses a path-segment pattern into an AST.
//
// Corresponds to TypeScript AST.fromGlob(pattern, options):
// scan structure with Scan, then build the tree. Does not flatten, fill
// negative tails, or compile to a regular expression.
func ParseGlob(pattern string, opts Config) *AST {
	max := opts.MaxExtglobRecursion
	if max == 0 {
		max = 2 // TypeScript default maxExtglobRecursion
	}
	return ParseTokens(pattern, scan.Scan(pattern, opts.NoExt, max), opts)
}

// ParseTokens builds an AST from tokens previously produced by Scan on src.
//
// src must be the same string that was scanned (used for unfinished-extglob
// demotion spans). tokens should be Scan(src, opts).
func ParseTokens(src string, tokens []scan.Token, opts Config) *AST {
	p := &astParser{
		src:    src,
		tokens: tokens,
		opts:   opts,
		nextID: 1,
	}
	root := p.newNode(0, nil)
	p.parseSequence(root)
	return root
}

type astParser struct {
	src    string
	tokens []scan.Token
	i      int
	opts   Config
	nextID int
}

func (p *astParser) newNode(typ scan.ExtglobType, parent *AST) *AST {
	n := &AST{
		Type:    typ,
		Options: p.opts,
		id:      p.nextID,
	}
	p.nextID++
	if parent == nil {
		n.root = n
		n.Options = p.opts
	} else {
		n.parent = parent
		n.parentIndex = len(parent.Parts)
		n.root = parent.root
		n.Options = parent.root.Options
	}
	// TypeScript: if (type === '!' && !this.#root.#filledNegs) this.#negs.push(this)
	if typ == scan.ExtglobNegate && n.root != nil && !n.root.filledNegs {
		n.root.negs = append(n.root.negs, n)
	}
	// Extglobs are inherently magical (TypeScript constructor).
	if typ != 0 {
		n.hasMagic = true
	}
	return n
}

// parseSequence fills a list node from the current token position until
// EOF or a Pipe / ExtglobClose that belongs to an enclosing extglob.
func (p *astParser) parseSequence(into *AST) {
	for p.i < len(p.tokens) {
		tok := p.tokens[p.i]
		switch tok.Kind {
		case scan.TokenText:
			into.pushText(tok.Text)
			p.i++
		case scan.TokenExtglobOpen:
			open := tok
			p.i++
			child := p.parseExtglob(scan.ExtglobType(open.Text[0]), into, open.Start)
			into.pushNode(child)
		case scan.TokenPipe, scan.TokenExtglobClose:
			// Leave for the enclosing parseExtglob.
			return
		default:
			p.i++
		}
	}
}

// parseExtglob builds an extglob node. The ExtglobOpen token is already consumed.
// openStart is the byte offset of the type character in src (for demotion).
func (p *astParser) parseExtglob(typ scan.ExtglobType, parent *AST, openStart int) *AST {
	ext := p.newNode(typ, parent)
	var alts []*AST

	for {
		alt := p.newNode(0, ext)
		p.parseSequence(alt)

		if p.i >= len(p.tokens) {
			// Unfinished extglob — TypeScript demotes the node to a list
			// holding the raw text from the type character through EOF.
			p.demote(ext, openStart)
			return ext
		}

		tok := p.tokens[p.i]
		switch tok.Kind {
		case scan.TokenPipe:
			p.i++
			alts = append(alts, alt)
			continue
		case scan.TokenExtglobClose:
			p.i++
			// emptyExt: first body empty and no alternatives completed yet
			// (TypeScript: acc === '' && ast.#parts.length === 0).
			if len(alts) == 0 && len(alt.Parts) == 0 {
				ext.EmptyExt = true
			}
			alts = append(alts, alt)
			for _, a := range alts {
				ext.pushNode(a)
			}
			return ext
		default:
			// Should not happen if tokens come from Scan; demote for safety.
			p.demote(ext, openStart)
			return ext
		}
	}
}

// demote converts an unfinished/malformed extglob into a list node with a
// single text part, matching TypeScript:
//
//	ast.type = null
//	ast.#hasMagic = undefined
//	ast.#parts = [str.substring(pos - 1)]
func (p *astParser) demote(ext *AST, openStart int) {
	raw := p.src[openStart:]
	ext.Type = 0
	ext.EmptyExt = false
	ext.Parts = nil
	if raw != "" {
		ext.Parts = []ASTPart{{Text: raw}}
	}
}

func (n *AST) pushText(s string) {
	if s == "" || n == nil {
		return
	}
	n.Parts = append(n.Parts, ASTPart{Text: s})
}

func (n *AST) pushNode(child *AST) {
	if n == nil || child == nil {
		return
	}
	// parentIndex is fixed at construction time (TypeScript constructor),
	// including the case where several extglob alternatives are created
	// before any are pushed — all keep parentIndex 0 until push.
	child.parent = n
	child.root = n.root
	n.Parts = append(n.Parts, ASTPart{Node: child})
}

// String reconstructs a pattern string from the tree (TypeScript toString
// before flatten). List nodes concatenate parts; extglob nodes emit
// type(alt|alt|...).
func (n *AST) String() string {
	if n == nil {
		return ""
	}
	if n.IsList() {
		var b []byte
		for _, p := range n.Parts {
			if p.Node != nil {
				b = append(b, p.Node.String()...)
			} else {
				b = append(b, p.Text...)
			}
		}
		return string(b)
	}
	// Extglob: type(alt|alt|...)
	var b []byte
	b = append(b, byte(n.Type), '(')
	for i, p := range n.Parts {
		if i > 0 {
			b = append(b, '|')
		}
		if p.Node != nil {
			b = append(b, p.Node.String()...)
		} else {
			b = append(b, p.Text...)
		}
	}
	b = append(b, ')')
	return string(b)
}

// ToJSON returns a JSON-like structure comparable to TypeScript AST.toJSON()
// before fillNegs (no start/end markers beyond root list markers when
// isStart/isEnd apply without filled negs).
//
// List: [ [], ...parts..., {}? ] when isStart / isEnd on root-like nodes.
// Extglob: [ type, ...alternative JSONs ]
//
// This is for inspection and tests only; it is not a stable wire format.
func (n *AST) ToJSON() any {
	if n == nil {
		return nil
	}
	if n.IsExtglob() {
		out := make([]any, 0, 1+len(n.Parts))
		out = append(out, string(n.Type))
		for _, p := range n.Parts {
			if p.Node != nil {
				out = append(out, p.Node.ToJSON())
			} else {
				out = append(out, p.Text)
			}
		}
		return out
	}
	// List node
	out := make([]any, 0, len(n.Parts)+2)
	if n.isStart() {
		out = append(out, []any{})
	}
	for _, p := range n.Parts {
		if p.Node != nil {
			out = append(out, p.Node.ToJSON())
		} else {
			out = append(out, p.Text)
		}
	}
	// TypeScript: if (isEnd && (root || (filledNegs && parent?.type === '!'))) push {}
	if n.isEnd() {
		if n == n.root || (n.root != nil && n.root.filledNegs && n.parent != nil && n.parent.Type == scan.ExtglobNegate) {
			out = append(out, map[string]any{})
		}
	}
	return out
}

// isStart mirrors TypeScript AST.isStart for structural markers in ToJSON.
func (n *AST) isStart() bool {
	if n.root == n {
		return true
	}
	if n.parent == nil || !n.parent.isStart() {
		return false
	}
	if n.parentIndex == 0 {
		return true
	}
	for i := 0; i < n.parentIndex; i++ {
		pp := n.parent.Parts[i]
		if pp.Node == nil || pp.Node.Type != scan.ExtglobNegate {
			return false
		}
	}
	return true
}

// isEnd mirrors TypeScript AST.isEnd (without relying on filledNegs).
func (n *AST) isEnd() bool {
	if n.root == n {
		return true
	}
	if n.parent != nil && n.parent.Type == scan.ExtglobNegate {
		return true
	}
	if n.parent == nil || !n.parent.isEnd() {
		return false
	}
	if n.IsList() {
		return n.parent.isEnd()
	}
	pl := 0
	if n.parent != nil {
		pl = len(n.parent.Parts)
	}
	return n.parentIndex == pl-1
}
