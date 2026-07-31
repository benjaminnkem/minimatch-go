package minimatch

import "github.com/benjaminnkem/minimatch-go/internal/ast"

// AST is the extglob syntax tree for a path segment.
type AST = ast.AST

// ASTPart is one entry in an AST node.
type ASTPart = ast.ASTPart

// RegExpSource is the result of AST.ToRegExpSource.
type RegExpSource = ast.RegExpSource

// MMPattern is a compiled path-segment pattern (literal or regexp2).
type MMPattern = ast.MMPattern

// ParseGlob parses a path-segment pattern into an AST.
func ParseGlob(pattern string, opts Options) *AST {
	return ast.ParseGlob(pattern, toASTConfig(opts))
}

// ParseTokens builds an AST from tokens previously produced by Scan.
func ParseTokens(src string, tokens []Token, opts Options) *AST {
	return ast.ParseTokens(src, tokens, toASTConfig(opts))
}

func toASTConfig(o Options) ast.Config {
	return ast.Config{
		Dot:                 o.Dot,
		NoCase:              o.NoCase,
		NoCaseMagicOnly:     o.NoCaseMagicOnly,
		NoExt:               o.NoExt,
		MaxExtglobRecursion: o.EffectiveMaxExtglobRecursion(),
	}
}
