// Package minimatch is an idiomatic Go port of the JavaScript/TypeScript
// minimatch library: a bash-style glob matcher used widely in the npm
// ecosystem.
//
// This package preserves the observable behaviour of the reference
// implementation at minimatch (TypeScript), including options semantics,
// edge cases, and ordering. It is not a mechanical file-by-file translation.
//
// # Status
//
// The port is built incrementally. The foundation layer provides shared
// types (including the full Options / MinimatchOptions model), sentinel
// errors, pattern validation, and Escape/Unescape. Matching, brace
// expansion, character-class parsing, the extglob AST, and the public match
// APIs are added in later steps.
//
// # Package layout
//
// The module root is a single public package, minimatch. That matches
// common Go library style for a focused API (compare path/filepath) and
// keeps exported names stable as subsystems land.
//
//	minimatch-go/                 module github.com/tochison/minimatch
//	  doc.go                      this file — package overview
//	  platform.go                 Platform and host detection
//	  options.go                  Options and documented defaults
//	  errors.go                   sentinel errors
//	  validate.go                 pattern length / validity checks
//	  escape.go / unescape.go     literal escape helpers
//	  token.go / scanner.go       path-segment lexical scan (#parseAST)
//	  brace_expand.go / balanced.go  bash brace expansion
//	  ast.go / ast_flatten.go / ast_fill_negs.go
//	                              extglob AST, flatten, negative tails
//	  parse_class.go              [character classes] + POSIX
//	  ast_regexp.go               toRegExpSource / toMMPattern
//	  minimatch.go / match.go / pattern_part.go
//	                              compile + path matching
//	  …                           future subsystems (makeRe, public API, …)
//
// Future code stays in package minimatch unless a hard boundary appears
// (for example a large brace-expansion implementation that benefits from
// internal/ isolation). Subpackages are not introduced pre-emptively.
//
// The TypeScript tree under ../minimatch is the behavioural specification
// and is read-only for this port.
//
// # Reference
//
//	https://github.com/isaacs/minimatch
package minimatch
