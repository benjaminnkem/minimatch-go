package minimatch

// FillNegs copies pattern tails into negative extglob alternatives.
//
// Corresponds to TypeScript AST.#fillNegs(). It must run on the root (or any
// node — work is always applied via the root). Typical order matches
// toRegExpSource:
//
//	ast.Flatten()
//	ast.FillNegs()
//
// For each !(...) node registered at parse time, every following sibling in
// each list ancestor is copyIn'd into each alternative of that !. That way
// a pattern like "!(a)b" negates "ab", not just "a".
//
// FillNegs does not match paths or emit regular expressions. It mutates the
// tree in place and returns the receiver for chaining. Calling it twice is
// a no-op (TypeScript #filledNegs).
func (n *AST) FillNegs() *AST {
	if n == nil {
		return nil
	}
	root := n.root
	if root == nil {
		root = n
	}
	if root.filledNegs {
		return n
	}

	// TypeScript calls toString() once before mutating so cached strings
	// reflect the pre-fill tree. We compute String for parity; we do not
	// cache it on the node.
	_ = root.String()

	root.filledNegs = true

	// Process LIFO (TypeScript: while ((n = this.#negs.pop())))
	for len(root.negs) > 0 {
		neg := root.negs[len(root.negs)-1]
		root.negs = root.negs[:len(root.negs)-1]
		if neg.Type != ExtglobNegate {
			continue
		}
		p := neg
		pp := p.parent
		for pp != nil {
			// TypeScript: for (i = parentIndex+1; !pp.type && i < parts.length; i++)
			// Only list parents contribute following siblings.
			if pp.IsList() {
				for i := p.parentIndex + 1; i < len(pp.Parts); i++ {
					for _, alt := range neg.Parts {
						if alt.Node == nil {
							// Extglob alternatives are always list nodes.
							continue
						}
						alt.Node.copyIn(pp.Parts[i])
					}
				}
			}
			p = pp
			pp = p.parent
		}
	}
	return n
}

// copyIn appends a clone of part into n (TypeScript AST.copyIn).
func (n *AST) copyIn(part ASTPart) {
	if n == nil {
		return
	}
	if part.Node == nil {
		n.pushText(part.Text)
		return
	}
	n.pushNode(part.Node.clone(n))
}

// clone deep-copies n under parent (TypeScript AST.clone).
//
// Because FillNegs sets filledNegs before cloning tails, cloned ! nodes are
// not re-registered in root.negs.
func (n *AST) clone(parent *AST) *AST {
	if n == nil {
		return nil
	}
	c := &AST{
		Type:     n.Type,
		EmptyExt: n.EmptyExt,
		Options:  parent.root.Options,
		parent:   parent,
		// parentIndex at construction time (TS constructor), before push.
		parentIndex: len(parent.Parts),
		root:        parent.root,
	}
	// Do not push onto root.negs: filledNegs is already true during FillNegs.
	// Recurse parts via copyIn so nested structure is cloned.
	for _, p := range n.Parts {
		c.copyIn(p)
	}
	return c
}

// FilledNegs reports whether FillNegs has been applied on this tree's root.
func (n *AST) FilledNegs() bool {
	if n == nil || n.root == nil {
		return false
	}
	return n.root.filledNegs
}
