package ast

// Config is the subset of minimatch.Options needed by the AST compiler.
type Config struct {
	Dot                 bool
	NoCase              bool
	NoCaseMagicOnly     bool
	NoExt               bool
	MaxExtglobRecursion int // effective value; default 2 supplied by caller
}
