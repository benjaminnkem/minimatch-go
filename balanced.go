package minimatch

import "strings"

// balanced finds the range of the first balanced pair of open/close strings
// in s, matching the balanced-match package used by brace-expansion.
//
// On success, pre is s before the opening, body is between open and close
// (excluding delimiters), and post is after the closing.
// ok is false when no balanced pair exists.
//
// Corresponds to: import { balanced } from 'balanced-match'
func balanced(open, close, s string) (pre, body, post string, ok bool) {
	r := balancedRange(open, close, s)
	if r == nil {
		return "", "", "", false
	}
	start, end := r[0], r[1]
	return s[:start], s[start+len(open) : end], s[end+len(close):], true
}

// balancedRange returns [start, end] indices of the open and close delimiters,
// or nil. Mirrors balanced-match range().
func balancedRange(a, b, str string) []int {
	ai := indexFrom(str, a, 0)
	bi := indexFrom(str, b, ai+1)
	i := ai
	if ai < 0 || bi <= 0 {
		return nil
	}
	if a == b {
		return []int{ai, bi}
	}
	begs := []int{}
	left := len(str)
	var right int
	rightSet := false
	var result []int
	for i >= 0 && result == nil {
		if i == ai {
			begs = append(begs, i)
			ai = indexFrom(str, a, i+1)
		} else if len(begs) == 1 {
			r := begs[len(begs)-1]
			begs = begs[:len(begs)-1]
			result = []int{r, bi}
		} else {
			beg := begs[len(begs)-1]
			begs = begs[:len(begs)-1]
			if beg < left {
				left = beg
				right = bi
				rightSet = true
			}
			bi = indexFrom(str, b, i+1)
		}
		if ai < bi && ai >= 0 {
			i = ai
		} else {
			i = bi
		}
	}
	if len(begs) > 0 && rightSet {
		result = []int{left, right}
	}
	return result
}

func indexFrom(s, sub string, from int) int {
	if from < 0 {
		from = 0
	}
	if from > len(s) {
		return -1
	}
	j := strings.Index(s[from:], sub)
	if j < 0 {
		return -1
	}
	return from + j
}
