// Package minimatch is an idiomatic Go port of the JavaScript/TypeScript
// minimatch library: a bash-style glob matcher used widely in the npm
// ecosystem.
//
// This package preserves the observable behaviour of the reference
// implementation at minimatch (TypeScript), including options semantics,
// edge cases, and ordering. It is not a mechanical file-by-file translation.
//
// # Package layout
//
//	minimatch-go/                      module github.com/benjaminnkem/minimatch-go
//	  *.go                             public API (package minimatch)
//	  internal/
//	    brace/                         bash brace expansion
//	    scan/                          path-segment lexer
//	    class/                         [character classes] + POSIX
//	    ast/                           extglob AST + segment regexp compile
//	  testdata/                        Node oracles and fixtures
//	  .github/workflows/               CI
//
// Callers import only github.com/benjaminnkem/minimatch-go. Implementation packages
// under internal/ are not part of the compatibility surface.
//
// The TypeScript tree under ../minimatch is the behavioural specification
// and is read-only for this port.
//
// # Reference
//
//	https://github.com/isaacs/minimatch
package minimatch
