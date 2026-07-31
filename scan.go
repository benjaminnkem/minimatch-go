package minimatch

import "github.com/benjaminnkem/minimatch-go/internal/scan"

// Re-export scan types for advanced callers.
type (
	Token       = scan.Token
	TokenKind   = scan.TokenKind
	ExtglobType = scan.ExtglobType
)

// Token kind constants.
const (
	TokenText         = scan.TokenText
	TokenExtglobOpen  = scan.TokenExtglobOpen
	TokenPipe         = scan.TokenPipe
	TokenExtglobClose = scan.TokenExtglobClose
)

// Extglob type constants.
const (
	ExtglobNegate   = scan.ExtglobNegate
	ExtglobOptional = scan.ExtglobOptional
	ExtglobPlus     = scan.ExtglobPlus
	ExtglobStar     = scan.ExtglobStar
	ExtglobOne      = scan.ExtglobOne
)

// IsExtglobType reports whether c is an extglob type character.
func IsExtglobType(c byte) bool { return scan.IsExtglobType(c) }

// Scan tokenizes a single path-segment pattern string.
func Scan(segment string, opts Options) []Token {
	return scan.Scan(segment, opts.NoExt, opts.EffectiveMaxExtglobRecursion())
}
