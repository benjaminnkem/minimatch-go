package minimatch

import (
	"strings"
)

// bodySeg is one non-** section between globstars, with the last file index
// where it may still start (TypeScript bodySegments entry).
type bodySeg struct {
	parts []PatternPart
	after int
}

// Match reports whether path p matches pattern under opts.
//
// Corresponds to TypeScript minimatch(p, pattern, options).
func Match(p, pattern string, opts Options) (bool, error) {
	if err := ValidatePattern(pattern); err != nil {
		return false, err
	}
	if !opts.NoComment && len(pattern) > 0 && pattern[0] == '#' {
		return false, nil
	}
	m, err := NewMinimatch(pattern, opts)
	if err != nil {
		return false, err
	}
	return m.Match(p), nil
}

// Match reports whether path f matches this compiled pattern.
func (m *Minimatch) Match(f string) bool {
	return m.MatchPartial(f, m.Partial)
}

// MatchPartial is Match with an explicit partial flag.
func (m *Minimatch) MatchPartial(f string, partial bool) bool {
	if m.Comment {
		return false
	}
	if m.Empty {
		return f == ""
	}
	if f == "/" && partial {
		return true
	}

	if m.IsWindows {
		f = strings.ReplaceAll(f, `\`, `/`)
	}

	ff := m.SlashSplit(f)

	filename := ""
	if len(ff) > 0 {
		filename = ff[len(ff)-1]
	}
	if filename == "" {
		for i := len(ff) - 2; i >= 0 && filename == ""; i-- {
			filename = ff[i]
		}
	}

	for _, pattern := range m.Set {
		file := ff
		if m.Options.MatchBase && len(pattern) == 1 {
			file = []string{filename}
		}
		if m.MatchOne(file, pattern, partial) {
			if m.Options.FlipNegate {
				return true
			}
			return !m.Negate
		}
	}
	if m.Options.FlipNegate {
		return false
	}
	return m.Negate
}

// MatchOne matches a split path against one compiled pattern row.
func (m *Minimatch) MatchOne(file []string, pattern []PatternPart, partial bool) bool {
	pattern = append([]PatternPart(nil), pattern...)

	fileStartIndex := 0
	patternStartIndex := 0

	if m.IsWindows {
		fileDrive := len(file) > 0 && driveLetterRE.MatchString(file[0])
		fileUNC := !fileDrive && len(file) >= 4 &&
			file[0] == "" && file[1] == "" && file[2] == "?" &&
			driveLetterRE.MatchString(file[3])

		patternDrive := len(pattern) > 0 && pattern[0].isStringPart() &&
			driveLetterRE.MatchString(pattern[0].Str)
		patternUNC := !patternDrive && len(pattern) >= 4 &&
			pattern[0].isStringPart() && pattern[0].Str == "" &&
			pattern[1].isStringPart() && pattern[1].Str == "" &&
			pattern[2].isStringPart() && pattern[2].Str == "?" &&
			pattern[3].isStringPart() && driveLetterRE.MatchString(pattern[3].Str)

		fdi, pdi := -1, -1
		if fileUNC {
			fdi = 3
		} else if fileDrive {
			fdi = 0
		}
		if patternUNC {
			pdi = 3
		} else if patternDrive {
			pdi = 0
		}
		if fdi >= 0 && pdi >= 0 {
			fd, pd := file[fdi], pattern[pdi].Str
			if strings.EqualFold(fd, pd) {
				pattern[pdi].Str = fd
				patternStartIndex = pdi
				fileStartIndex = fdi
			}
		}
	}

	if m.Options.EffectiveOptimizationLevel() >= 2 {
		file = m.LevelTwoFileOptimize(file)
	}

	if patternHasGlobStar(pattern) {
		return m.matchGlobstar(file, pattern, partial, fileStartIndex, patternStartIndex)
	}
	return m.matchOne(file, pattern, partial, fileStartIndex, patternStartIndex)
}

func patternHasGlobStar(pattern []PatternPart) bool {
	for _, p := range pattern {
		if p.IsGlobStar {
			return true
		}
	}
	return false
}

func indexOfGlobStar(pattern []PatternPart, from int) int {
	for i := from; i < len(pattern); i++ {
		if pattern[i].IsGlobStar {
			return i
		}
	}
	return -1
}

func lastIndexOfGlobStar(pattern []PatternPart) int {
	for i := len(pattern) - 1; i >= 0; i-- {
		if pattern[i].IsGlobStar {
			return i
		}
	}
	return -1
}

func (m *Minimatch) matchOne(
	file []string,
	pattern []PatternPart,
	partial bool,
	fileIndex, patternIndex int,
) bool {
	fi, pi := fileIndex, patternIndex
	fl, pl := len(file), len(pattern)
	for fi < fl && pi < pl {
		p := pattern[pi]
		f := file[fi]
		if p.IsGlobStar {
			return false
		}
		if !p.matchSegment(f) {
			return false
		}
		fi++
		pi++
	}
	if fi == fl && pi == pl {
		return true
	}
	if fi == fl {
		return partial
	}
	if pi == pl {
		return fi == fl-1 && file[fi] == ""
	}
	return false
}

func (m *Minimatch) matchGlobstar(
	file []string,
	pattern []PatternPart,
	partial bool,
	fileIndex, patternIndex int,
) bool {
	firstgs := indexOfGlobStar(pattern, patternIndex)
	lastgs := lastIndexOfGlobStar(pattern)

	var head, body, tail []PatternPart
	if partial {
		head = pattern[patternIndex:firstgs]
		body = pattern[firstgs+1:]
		tail = nil
	} else {
		head = pattern[patternIndex:firstgs]
		if firstgs < lastgs {
			body = pattern[firstgs+1 : lastgs]
		} else {
			body = nil
		}
		tail = pattern[lastgs+1:]
	}

	if len(head) > 0 {
		if fileIndex+len(head) > len(file) {
			return false
		}
		if !m.matchOne(file[fileIndex:fileIndex+len(head)], head, partial, 0, 0) {
			return false
		}
		fileIndex += len(head)
	}

	fileTailMatch := 0
	if len(tail) > 0 {
		if len(tail)+fileIndex > len(file) {
			return false
		}
		tailStart := len(file) - len(tail)
		if m.matchOne(file, tail, partial, tailStart, 0) {
			fileTailMatch = len(tail)
		} else {
			if len(file) == 0 || file[len(file)-1] != "" || fileIndex+len(tail) == len(file) {
				return false
			}
			tailStart--
			if !m.matchOne(file, tail, partial, tailStart, 0) {
				return false
			}
			fileTailMatch = len(tail) + 1
		}
	}

	if len(body) == 0 {
		sawSome := fileTailMatch != 0
		for i := fileIndex; i < len(file)-fileTailMatch; i++ {
			f := file[i]
			sawSome = true
			if f == "." || f == ".." || (!m.Options.Dot && strings.HasPrefix(f, ".")) {
				return false
			}
		}
		return partial || sawSome
	}

	// Build body segments between ** markers.
	var bodySegments []bodySeg
	bodySegments = append(bodySegments, bodySeg{})
	cur := 0
	nonGsParts := 0
	nonGsPartsSums := []int{0}
	for _, b := range body {
		if b.IsGlobStar {
			nonGsPartsSums = append(nonGsPartsSums, nonGsParts)
			bodySegments = append(bodySegments, bodySeg{})
			cur = len(bodySegments) - 1
		} else {
			bodySegments[cur].parts = append(bodySegments[cur].parts, b)
			nonGsParts++
		}
	}

	fileLength := len(file) - fileTailMatch
	// TS assigns after using nonGsPartsSums from the end of the list.
	ii := len(bodySegments) - 1
	for j := 0; j < len(bodySegments); j++ {
		sum := 0
		if ii >= 0 && ii < len(nonGsPartsSums) {
			sum = nonGsPartsSums[ii]
		}
		ii--
		bodySegments[j].after = fileLength - (sum + len(bodySegments[j].parts))
	}

	// TypeScript: return !!result — null and false are both non-matches.
	return m.matchGSBody(file, bodySegments, fileIndex, 0, partial, 0, fileTailMatch != 0) == gsTrue
}

// gsResult mirrors TS boolean | null for globstar body recursion.
type gsResult int

const (
	gsFalse gsResult = iota
	gsTrue
	gsNull // cannot keep trying
)

func (m *Minimatch) matchGSBody(
	file []string,
	bodySegments []bodySeg,
	fileIndex, bodyIndex int,
	partial bool,
	globStarDepth int,
	sawTail bool,
) gsResult {
	if bodyIndex >= len(bodySegments) {
		for i := fileIndex; i < len(file); i++ {
			sawTail = true
			f := file[i]
			if f == "." || f == ".." || (!m.Options.Dot && strings.HasPrefix(f, ".")) {
				return gsFalse
			}
		}
		if sawTail {
			return gsTrue
		}
		return gsFalse
	}

	bs := bodySegments[bodyIndex]
	body := bs.parts
	after := bs.after

	for fileIndex <= after {
		// match body at fileIndex
		end := fileIndex + len(body)
		if end > len(file) {
			break
		}
		// TypeScript passes file.slice(0, fileIndex+body.length) with start fileIndex
		ok := m.matchOne(file[:end], body, partial, fileIndex, 0)
		if ok && globStarDepth < m.MaxGlobstarRecursion {
			sub := m.matchGSBody(file, bodySegments, end, bodyIndex+1, partial, globStarDepth+1, sawTail)
			if sub != gsFalse {
				return sub
			}
		}
		if fileIndex >= len(file) {
			break
		}
		f := file[fileIndex]
		if f == "." || f == ".." || (!m.Options.Dot && strings.HasPrefix(f, ".")) {
			return gsFalse
		}
		fileIndex++
	}
	// TypeScript: return partial || null
	// When partial is true this is boolean true (!!true === true).
	// When partial is false this is null (!!null === false).
	if partial {
		return gsTrue
	}
	return gsNull
}
