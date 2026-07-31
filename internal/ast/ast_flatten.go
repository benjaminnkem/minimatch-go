package ast

import "github.com/tochison/minimatch/internal/scan"

// Flatten rewrites nested extglobs into equivalent shallower forms.
//
// Corresponds to TypeScript AST.#flatten(), which runs before fillNegs and
// toRegExpSource. It does not match paths or emit regular expressions.
//
// Three rewrites (checked in this order per child, up to 10 passes):
//
//  1. Adopt — pull a nested extglob’s alternatives into the parent when
//     quantifier semantics allow (adoptionMap).
//  2. Adopt with space — same, but add an empty alternative so optional /
//     star-zero semantics stay correct (adoptionWithSpaceMap).
//  3. Usurp — when the parent has a single nested extglob child, replace
//     the parent’s type with a mapped type (usurpMap).
//
// List nodes only recurse into children. Flatten mutates the tree in place
// and returns the receiver for chaining.
func (n *AST) Flatten() *AST {
	if n == nil {
		return nil
	}
	n.flatten()
	return n
}

func (n *AST) flatten() {
	if n.IsList() {
		for _, p := range n.Parts {
			if p.Node != nil {
				p.Node.flatten()
			}
		}
		return
	}

	// Extglob node: iterative adopt / usurp (TypeScript up to 10 passes).
	for iterations := 0; iterations < 10; iterations++ {
		done := true
		for i := 0; i < len(n.Parts); i++ {
			c := n.Parts[i].Node
			if c == nil {
				continue
			}
			c.flatten()
			if n.canAdopt(c, adoptionMap) {
				done = false
				n.adopt(c, i)
			} else if n.canAdopt(c, adoptionWithSpaceMap) {
				done = false
				n.adoptWithSpace(c, i)
			} else if n.canUsurp(c) {
				done = false
				n.usurp(c)
			}
		}
		if done {
			break
		}
	}
}

// adoptionMap: parent → child types that may be adopted without an empty alt.
// TypeScript adoptionMap.
var adoptionMap = map[scan.ExtglobType][]scan.ExtglobType{
	scan.ExtglobNegate:   {scan.ExtglobOne},
	scan.ExtglobOptional: {scan.ExtglobOptional, scan.ExtglobOne},
	scan.ExtglobOne:      {scan.ExtglobOne},
	scan.ExtglobStar:     {scan.ExtglobStar, scan.ExtglobPlus, scan.ExtglobOptional, scan.ExtglobOne},
	scan.ExtglobPlus:     {scan.ExtglobPlus, scan.ExtglobOne},
}

// adoptionWithSpaceMap: adopt but insert an empty alternative.
// TypeScript adoptionWithSpaceMap.
var adoptionWithSpaceMap = map[scan.ExtglobType][]scan.ExtglobType{
	scan.ExtglobNegate: {scan.ExtglobOptional},
	scan.ExtglobOne:    {scan.ExtglobOptional},
	scan.ExtglobPlus:   {scan.ExtglobOptional, scan.ExtglobStar},
}

// usurpMap: parent type → (child type → resulting parent type).
// TypeScript usurpMap. Missing parent types cannot usurp.
var usurpMap = map[scan.ExtglobType]map[scan.ExtglobType]scan.ExtglobType{
	scan.ExtglobNegate: {
		scan.ExtglobNegate: scan.ExtglobOne, // !(!(...)) → @(...)
	},
	scan.ExtglobOptional: {
		scan.ExtglobStar: scan.ExtglobStar, // ?(*(...)) → *(...)
		scan.ExtglobPlus: scan.ExtglobStar, // ?(+ (...)) → *(...)
	},
	scan.ExtglobOne: {
		scan.ExtglobNegate:   scan.ExtglobNegate,
		scan.ExtglobOptional: scan.ExtglobOptional,
		scan.ExtglobOne:      scan.ExtglobOne,
		scan.ExtglobStar:     scan.ExtglobStar,
		scan.ExtglobPlus:     scan.ExtglobPlus,
	},
	scan.ExtglobPlus: {
		scan.ExtglobOptional: scan.ExtglobStar, // +(? (...)) mapped when not adopt-with-space
		scan.ExtglobStar:     scan.ExtglobStar,
	},
}

func typeIn(list []scan.ExtglobType, t scan.ExtglobType) bool {
	for _, x := range list {
		if x == t {
			return true
		}
	}
	return false
}

// canAdopt reports whether child is a single-extglob list alternative that
// this extglob may adopt under map.
//
// TypeScript #canAdopt: child.type === null, child.parts.length === 1,
// grandchild is extglob, and map allows parent←grandchild type.
func (n *AST) canAdopt(child *AST, m map[scan.ExtglobType][]scan.ExtglobType) bool {
	if n == nil || !n.IsExtglob() || child == nil || !child.IsList() {
		return false
	}
	if len(child.Parts) != 1 || child.Parts[0].Node == nil {
		return false
	}
	gc := child.Parts[0].Node
	if !gc.IsExtglob() {
		return false
	}
	allowed, ok := m[n.Type]
	if !ok {
		return false
	}
	return typeIn(allowed, gc.Type)
}

// adopt replaces child (a list wrapping one nested extglob) with that
// nested extglob’s alternatives.
//
// TypeScript #adopt: splice(index, 1, ...gc.#parts) and reparent.
func (n *AST) adopt(child *AST, index int) {
	if index < 0 || index >= len(n.Parts) || n.Parts[index].Node != child {
		return
	}
	if len(child.Parts) != 1 || child.Parts[0].Node == nil {
		return
	}
	gc := child.Parts[0].Node
	// Build replacement parts from grandchild alternatives.
	repl := make([]ASTPart, len(gc.Parts))
	for i, p := range gc.Parts {
		repl[i] = p
		if p.Node != nil {
			p.Node.parent = n
		}
	}
	// splice: n.Parts[index] = repl...
	newParts := make([]ASTPart, 0, len(n.Parts)-1+len(repl))
	newParts = append(newParts, n.Parts[:index]...)
	newParts = append(newParts, repl...)
	newParts = append(newParts, n.Parts[index+1:]...)
	n.Parts = newParts
}

// adoptWithSpace is adopt after adding an empty alternative to the nested
// extglob so zero-match semantics remain valid.
//
// TypeScript #adoptWithSpace: blank list with ” pushed onto gc, then #adopt.
func (n *AST) adoptWithSpace(child *AST, index int) {
	if len(child.Parts) != 1 || child.Parts[0].Node == nil {
		return
	}
	gc := child.Parts[0].Node
	// blank = new AST(null, gc); blank.#parts.push('') — empty string is kept.
	blank := &AST{
		Type:    0,
		root:    n.root,
		parent:  gc,
		Options: n.root.Options,
	}
	blank.Parts = []ASTPart{{Text: ""}}
	// push blank onto gc (TypeScript gc.push(blank))
	blank.parent = gc
	blank.parentIndex = len(gc.Parts)
	blank.root = gc.root
	gc.Parts = append(gc.Parts, ASTPart{Node: blank})
	n.adopt(child, index)
}

// canUsurp reports whether this extglob has exactly one list child that
// wraps a single nested extglob, and usurpMap allows the type change.
//
// TypeScript #canUsurp: this.parts.length === 1, child list of one extglob.
func (n *AST) canUsurp(child *AST) bool {
	if n == nil || !n.IsExtglob() || child == nil || !child.IsList() {
		return false
	}
	if len(n.Parts) != 1 || n.Parts[0].Node != child {
		return false
	}
	if len(child.Parts) != 1 || child.Parts[0].Node == nil {
		return false
	}
	gc := child.Parts[0].Node
	if !gc.IsExtglob() {
		return false
	}
	m, ok := usurpMap[n.Type]
	if !ok {
		return false
	}
	_, ok = m[gc.Type]
	return ok
}

// usurp replaces this extglob’s type and parts with the nested child’s.
//
// TypeScript #usurp.
func (n *AST) usurp(child *AST) {
	if len(child.Parts) != 1 || child.Parts[0].Node == nil {
		return
	}
	gc := child.Parts[0].Node
	m := usurpMap[n.Type]
	nt, ok := m[gc.Type]
	if !ok {
		return
	}
	n.Parts = gc.Parts
	for _, p := range n.Parts {
		if p.Node != nil {
			p.Node.parent = n
		}
	}
	n.Type = nt
	n.EmptyExt = false
}
