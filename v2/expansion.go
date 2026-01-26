package main

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/xyproto/iferr"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

func (e *Editor) handleReturnAutocomplete(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string, indent *bool, leadingWhitespace *string) bool {
	if e.autocompleteCLikeParens(trimmedLine, currentLeadingWhitespace, indent, leadingWhitespace) {
		return true
	}
	if e.autocompleteGoForRange(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoForNumericRange(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoIfInRange(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoListComprehension(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoTernary(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoArrowLambda(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoInAssignment(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteGoIfErrQuestion(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteIfErr(c, trimmedLine, currentLeadingWhitespace) {
		return true
	}
	if e.autocompleteTagExpansion(c, trimmedLine, currentLeadingWhitespace, indent, leadingWhitespace) {
		return true
	}
	return false
}

func (e *Editor) autocompleteCLikeParens(trimmedLine, currentLeadingWhitespace string, indent *bool, leadingWhitespace *string) bool {
	if !cLikeFor(e.mode) {
		return false
	}

	// Add missing parenthesis for "if ... {", "} else if", "} elif", "for", "while" and "when" for C-like languages
	for _, kw := range []string{"for", "foreach", "foreach_reverse", "if", "switch", "when", "while", "while let", "} else if", "} elif"} {
		if strings.HasPrefix(trimmedLine, kw+" ") && !strings.HasPrefix(trimmedLine, kw+" (") {
			kwLenPlus1 := len(kw) + 1
			if kwLenPlus1 < len(trimmedLine) {
				if strings.HasSuffix(trimmedLine, " {") && kwLenPlus1 < len(trimmedLine) && len(trimmedLine) > 3 {
					// Add ( and ), keep the final "{"
					e.SetCurrentLine(currentLeadingWhitespace + kw + " (" + trimmedLine[kwLenPlus1:len(trimmedLine)-2] + ") {")
					e.pos.mut.Lock()
					e.pos.sx += 2
					e.pos.mut.Unlock()
					return true
				} else if !strings.HasSuffix(trimmedLine, ")") {
					// Add ( and ), there is no final "{"
					e.SetCurrentLine(currentLeadingWhitespace + kw + " (" + trimmedLine[kwLenPlus1:] + ")")
					e.pos.mut.Lock()
					e.pos.sx += 2
					e.pos.mut.Unlock()
					*indent = true
					*leadingWhitespace = e.indentation.String() + currentLeadingWhitespace
					return true
				}
			}
		}
	}
	return false
}

func (e *Editor) autocompleteGoForRange(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	replacement, ok := goForRangeAutocomplete(trimmedLine)
	if !ok {
		return false
	}

	e.SetCurrentLine(currentLeadingWhitespace + replacement)
	// Keep the cursor aligned with the end of the line as if nothing changed.
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoForNumericRange(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	replacement, ok := goForNumericRangeAutocomplete(trimmedLine)
	if !ok {
		return false
	}

	e.SetCurrentLine(currentLeadingWhitespace + replacement)
	e.End(c)
	return true
}

func goForRangeAutocomplete(trimmedLine string) (string, bool) {
	if !strings.HasPrefix(trimmedLine, "for ") {
		return "", false
	}

	hasBrace := strings.HasSuffix(trimmedLine, "{")
	base := strings.TrimSpace(strings.TrimSuffix(trimmedLine, "{"))
	if !strings.HasPrefix(base, "for ") {
		return "", false
	}

	rest := strings.TrimSpace(strings.TrimPrefix(base, "for "))
	inIndex := strings.Index(rest, " in ")
	var lhs, rhs string
	if inIndex != -1 {
		lhs = strings.TrimSpace(rest[:inIndex])
		rhs = strings.TrimSpace(rest[inIndex+len(" in "):])
	} else {
		tokens := strings.Fields(rest)
		if len(tokens) < 3 {
			return "", false
		}
		inToken := -1
		for i, tok := range tokens {
			if tok == "in" {
				inToken = i
				break
			}
		}
		if inToken <= 0 || inToken == len(tokens)-1 {
			return "", false
		}
		lhs = strings.Join(tokens[:inToken], " ")
		rhs = strings.Join(tokens[inToken+1:], " ")
	}
	if lhs == "" || rhs == "" {
		return "", false
	}

	replacement := "for "
	if strings.Contains(lhs, ",") {
		replacement += lhs
	} else {
		replacement += "_, " + lhs
	}
	replacement += " := range " + rhs
	if hasBrace {
		replacement += " {"
	}
	return replacement, true
}

func goForNumericRangeAutocomplete(trimmedLine string) (string, bool) {
	if !strings.HasPrefix(trimmedLine, "for ") {
		return "", false
	}

	hasBrace := strings.HasSuffix(trimmedLine, "{")
	base := strings.TrimSpace(strings.TrimSuffix(trimmedLine, "{"))
	if !strings.HasPrefix(base, "for ") {
		return "", false
	}

	rest := strings.TrimSpace(strings.TrimPrefix(base, "for "))
	inIndex := strings.Index(rest, " in ")
	if inIndex == -1 {
		return "", false
	}
	lhs := strings.TrimSpace(rest[:inIndex])
	rangeExpr := strings.TrimSpace(rest[inIndex+len(" in "):])
	if lhs == "" || rangeExpr == "" {
		return "", false
	}
	if strings.Contains(lhs, ",") {
		return "", false
	}

	start, end, inclusive, ok := parseGoNumericRange(rangeExpr)
	if !ok {
		return "", false
	}

	op := "<"
	if inclusive {
		op = "<="
	}
	replacement := "for " + lhs + " := " + start + "; " + lhs + " " + op + " " + end + "; " + lhs + "++"
	if hasBrace {
		replacement += " {"
	}
	return replacement, true
}

func parseGoNumericRange(expr string) (string, string, bool, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", "", false, false
	}
	dots := strings.Index(expr, "..")
	if dots == -1 {
		return "", "", false, false
	}
	start := strings.TrimSpace(expr[:dots])
	rest := strings.TrimSpace(expr[dots+2:])
	inclusive := true
	if strings.HasPrefix(rest, "<") {
		inclusive = false
		rest = strings.TrimSpace(rest[1:])
	}
	end := strings.TrimSpace(rest)
	if start == "" || end == "" {
		return "", "", false, false
	}
	return start, end, inclusive, true
}

func (e *Editor) autocompleteGoIfInRange(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	lhs, rhs, hasBrace, negated, ok := goIfInAutocomplete(trimmedLine)
	if !ok || !hasBrace {
		return false
	}

	var replacement string
	switch e.goInContainerKind(rhs, "if true {") {
	case goInContainerMap:
		if negated {
			replacement = "if _, ok := " + rhs + "[" + lhs + "]; !ok {"
		} else {
			replacement = "if _, ok := " + rhs + "[" + lhs + "]; ok {"
		}
	case goInContainerArray:
		if negated {
			replacement = "if !slices.Contains(" + rhs + "[:], " + lhs + ") {"
		} else {
			replacement = "if slices.Contains(" + rhs + "[:], " + lhs + ") {"
		}
		e.ensureGoImport("slices")
	case goInContainerSlice:
		if negated {
			replacement = "if !slices.Contains(" + rhs + ", " + lhs + ") {"
		} else {
			replacement = "if slices.Contains(" + rhs + ", " + lhs + ") {"
		}
		e.ensureGoImport("slices")
	default:
		return e.autocompleteGoIfInFallback(c, currentLeadingWhitespace, lhs, rhs, negated)
	}
	e.SetCurrentLine(currentLeadingWhitespace + replacement)
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoListComprehension(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	assignKind, target, exprPart, iterVar, iterable, cond, ok := goListComprehensionAutocomplete(trimmedLine)
	if !ok {
		return false
	}

	targetName := target
	targetDecl := target
	hasVarType := false
	if assignKind == "var" {
		name, decl, hasType, ok := goParseVarTarget(target)
		if !ok {
			return false
		}
		targetName = name
		targetDecl = decl
		hasVarType = hasType
	}

	elemType, hasElemType := e.goListComprehensionElemType(exprPart, iterable)
	var initLine string
	switch assignKind {
	case "var":
		if hasVarType {
			initLine = "var " + targetDecl
		} else if hasElemType {
			initLine = "var " + targetName + " []" + elemType
		} else {
			initLine = "var " + targetName + " []any"
		}
	case "=":
		if hasElemType {
			initLine = target + " = []" + elemType + "{}"
		} else {
			initLine = target + " = nil"
		}
	default:
		if hasElemType {
			initLine = target + " := []" + elemType + "{}"
		} else {
			initLine = target + " := []any{}"
		}
	}

	rangeLHS := "_, " + iterVar
	if strings.Contains(iterVar, ",") {
		rangeLHS = iterVar
	}

	oneIndentation := e.indentation.String()
	lines := []string{
		initLine,
		"for " + rangeLHS + " := range " + iterable + " {",
	}
	if cond != "" {
		lines = append(lines, oneIndentation+"if "+cond+" {")
		lines = append(lines, oneIndentation+oneIndentation+targetName+" = append("+targetName+", "+exprPart+")")
		lines = append(lines, oneIndentation+"}")
	} else {
		lines = append(lines, oneIndentation+targetName+" = append("+targetName+", "+exprPart+")")
	}
	lines = append(lines, "}")

	for i, line := range lines {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoTernary(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	assignKind, target, expr, ok := goSplitAssignment(trimmedLine)
	if !ok || target == "" || expr == "" {
		return false
	}
	if strings.Contains(target, ",") {
		return false
	}

	cond, trueExpr, falseExpr, ok := goParseTernary(expr)
	if !ok {
		return false
	}

	oneIndentation := e.indentation.String()
	var lines []string

	if assignKind == "=" {
		lines = []string{
			"if " + cond + " {",
			oneIndentation + target + " = " + trueExpr,
			"} else {",
			oneIndentation + target + " = " + falseExpr,
			"}",
		}
	} else {
		targetName := target
		targetDecl := target
		hasVarType := false
		if assignKind == "var" {
			name, decl, hasType, ok := goParseVarTarget(target)
			if !ok {
				return false
			}
			targetName = name
			targetDecl = decl
			hasVarType = hasType
		}
		declLine := ""
		if assignKind == "var" && hasVarType {
			declLine = "var " + targetDecl
		} else {
			typ, okType := e.goTernaryType(trueExpr, falseExpr)
			if !okType {
				typ = "any"
			}
			declLine = "var " + targetName + " " + typ
		}
		lines = []string{
			declLine,
			"if " + cond + " {",
			oneIndentation + targetName + " = " + trueExpr,
			"} else {",
			oneIndentation + targetName + " = " + falseExpr,
			"}",
		}
	}

	for i, line := range lines {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoArrowLambda(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	line := trimmedLine
	changed := false
	for {
		replaced, ok := goArrowLambdaReplacement(line)
		if !ok || replaced == line {
			break
		}
		line = replaced
		changed = true
	}
	if !changed {
		return false
	}
	e.SetCurrentLine(currentLeadingWhitespace + line)
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoInAssignment(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go {
		return false
	}

	assignKind, target, lhs, rhs, negated, ok := goInAssignmentAutocomplete(trimmedLine)
	if !ok {
		return false
	}

	var lines []string
	switch e.goInContainerKind(rhs, "var _ = 0") {
	case goInContainerMap:
		lines = goInAssignmentMapLines(assignKind, target, lhs, rhs, negated)
	case goInContainerArray:
		if negated {
			lines = []string{goInAssignmentLine(assignKind, target, "!slices.Contains("+rhs+"[:], "+lhs+")")}
		} else {
			lines = []string{goInAssignmentLine(assignKind, target, "slices.Contains("+rhs+"[:], "+lhs+")")}
		}
		e.ensureGoImport("slices")
	case goInContainerSlice:
		if negated {
			lines = []string{goInAssignmentLine(assignKind, target, "!slices.Contains("+rhs+", "+lhs+")")}
		} else {
			lines = []string{goInAssignmentLine(assignKind, target, "slices.Contains("+rhs+", "+lhs+")")}
		}
		e.ensureGoImport("slices")
	default:
		return e.autocompleteGoInAssignmentFallback(c, currentLeadingWhitespace, assignKind, target, lhs, rhs, negated)
	}

	for i, line := range lines {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoIfInFallback(c *vt.Canvas, currentLeadingWhitespace, lhs, rhs string, negated bool) bool {
	oneIndentation := e.indentation.String()
	foundVar := "found"
	elemVar := "e"
	loopHeader := "for _, " + elemVar + " := range " + rhs + " {"
	compare := elemVar + " == " + lhs

	lines := []string{
		foundVar + " := false",
		loopHeader,
		oneIndentation + "if " + compare + " {",
		oneIndentation + oneIndentation + foundVar + " = true",
		oneIndentation + oneIndentation + "break",
		oneIndentation + "}",
		"}",
	}
	if negated {
		lines = append(lines, "if !"+foundVar+" {")
	} else {
		lines = append(lines, "if "+foundVar+" {")
	}

	for i, line := range lines {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func (e *Editor) autocompleteGoInAssignmentFallback(c *vt.Canvas, currentLeadingWhitespace, assignKind, target, lhs, rhs string, negated bool) bool {
	oneIndentation := e.indentation.String()
	foundVar := "found"
	elemVar := "e"
	lines := []string{
		foundVar + " := false",
		"for _, " + elemVar + " := range " + rhs + " {",
		oneIndentation + "if " + elemVar + " == " + lhs + " {",
		oneIndentation + oneIndentation + foundVar + " = true",
		oneIndentation + oneIndentation + "break",
		oneIndentation + "}",
		"}",
	}
	if negated {
		lines = append(lines, goInAssignmentLine(assignKind, target, "!"+foundVar))
	} else {
		lines = append(lines, goInAssignmentLine(assignKind, target, foundVar))
	}

	for i, line := range lines {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func goIfInAutocomplete(trimmedLine string) (string, string, bool, bool, bool) {
	if !strings.HasPrefix(trimmedLine, "if ") {
		return "", "", false, false, false
	}

	hasBrace := strings.HasSuffix(trimmedLine, "{")
	base := strings.TrimSpace(strings.TrimSuffix(trimmedLine, "{"))
	if !strings.HasPrefix(base, "if ") {
		return "", "", hasBrace, false, false
	}

	rest := strings.TrimSpace(strings.TrimPrefix(base, "if "))
	lhs, rhs, negated, ok := goParseInExpression(rest)
	if !ok {
		return "", "", hasBrace, false, false
	}
	return lhs, rhs, hasBrace, negated, true
}

func goInAssignmentAutocomplete(trimmedLine string) (string, string, string, string, bool, bool) {
	assignKind, target, expr, ok := goSplitAssignment(trimmedLine)
	if !ok || target == "" || expr == "" {
		return "", "", "", "", false, false
	}
	lhs, rhs, negated, ok := goParseInExpression(expr)
	if !ok {
		return "", "", "", "", false, false
	}
	return assignKind, target, lhs, rhs, negated, true
}

func goSplitAssignment(line string) (string, string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", "", "", false
	}
	if strings.HasPrefix(trimmed, "var ") {
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "var "))
		idx := goFindAssignmentOperator(rest)
		if idx == -1 {
			return "", "", "", false
		}
		if rest[idx:idx+1] != "=" {
			return "", "", "", false
		}
		target := strings.TrimSpace(rest[:idx])
		expr := strings.TrimSpace(rest[idx+1:])
		return "var", target, expr, true
	}
	idx := goFindAssignmentOperator(trimmed)
	if idx == -1 {
		return "", "", "", false
	}
	if idx+1 < len(trimmed) && trimmed[idx:idx+2] == ":=" {
		target := strings.TrimSpace(trimmed[:idx])
		expr := strings.TrimSpace(trimmed[idx+2:])
		return ":=", target, expr, true
	}
	target := strings.TrimSpace(trimmed[:idx])
	expr := strings.TrimSpace(trimmed[idx+1:])
	return "=", target, expr, true
}

func goParseVarTarget(target string) (string, string, bool, bool) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", "", false, false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", "", false, false
	}
	name := fields[0]
	if strings.Contains(name, ",") || !isSimpleGoIdentifier(name) {
		return "", "", false, false
	}
	hasType := len(fields) > 1
	return name, trimmed, hasType, true
}

func goFindAssignmentOperator(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] != '=' {
			continue
		}
		if i > 0 {
			switch line[i-1] {
			case ':':
				return i - 1
			case '!', '<', '>', '=':
				continue
			}
		}
		if i+1 < len(line) && line[i+1] == '=' {
			continue
		}
		return i
	}
	return -1
}

func goInAssignmentLine(kind, target, expr string) string {
	switch kind {
	case "var":
		return "var " + target + " = " + expr
	case "=":
		return target + " = " + expr
	default:
		return target + " := " + expr
	}
}

func goInAssignmentMapLines(kind, target, lhs, rhs string, negated bool) []string {
	if kind == "var" {
		if negated {
			return []string{
				"_, ok := " + rhs + "[" + lhs + "]",
				"var " + target + " = !ok",
			}
		}
		return []string{
			"_, ok := " + rhs + "[" + lhs + "]",
			"var " + target + " = ok",
		}
	}
	if kind == ":=" && !negated {
		return []string{"_, " + target + " := " + rhs + "[" + lhs + "]"}
	}
	if negated {
		return []string{
			"_, ok := " + rhs + "[" + lhs + "]",
			goInAssignmentLine(kind, target, "!ok"),
		}
	}
	return []string{"_, " + target + " = " + rhs + "[" + lhs + "]"}
}

func goParseInExpression(expr string) (string, string, bool, bool) {
	tokens := strings.Fields(expr)
	if len(tokens) < 3 {
		return "", "", false, false
	}
	inToken := -1
	negated := false
	for i, tok := range tokens {
		if tok == "!in" {
			negated = true
			inToken = i
			break
		}
		if tok == "not" && i+1 < len(tokens) && tokens[i+1] == "in" {
			negated = true
			inToken = i + 1
			break
		}
		if tok == "in" {
			inToken = i
			break
		}
	}
	if inToken == -1 || inToken == len(tokens)-1 {
		return "", "", false, false
	}
	lhsEnd := inToken
	if negated && tokens[inToken] == "in" {
		lhsEnd = inToken - 1
	}
	if lhsEnd <= 0 {
		return "", "", false, false
	}
	lhs := strings.Join(tokens[:lhsEnd], " ")
	rhs := strings.Join(tokens[inToken+1:], " ")
	if lhs == "" || rhs == "" {
		return "", "", false, false
	}
	return lhs, rhs, negated, true
}

func goListComprehensionAutocomplete(trimmedLine string) (string, string, string, string, string, string, bool) {
	assignKind, target, expr, ok := goSplitAssignment(trimmedLine)
	if !ok {
		return "", "", "", "", "", "", false
	}
	expr = strings.TrimSpace(expr)
	if len(expr) < 2 || expr[0] != '[' || expr[len(expr)-1] != ']' {
		return "", "", "", "", "", "", false
	}
	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" {
		return "", "", "", "", "", "", false
	}

	tokens := strings.Fields(inner)
	forIndex := -1
	for i, tok := range tokens {
		if tok == "for" {
			forIndex = i
			break
		}
	}
	if forIndex <= 0 {
		return "", "", "", "", "", "", false
	}
	exprPart := strings.Join(tokens[:forIndex], " ")

	inIndex := -1
	for i := forIndex + 1; i < len(tokens); i++ {
		if tokens[i] == "in" {
			inIndex = i
			break
		}
	}
	if inIndex == -1 || inIndex == len(tokens)-1 {
		return "", "", "", "", "", "", false
	}
	iterVar := strings.Join(tokens[forIndex+1:inIndex], " ")
	if iterVar == "" {
		return "", "", "", "", "", "", false
	}

	ifIndex := -1
	for i := inIndex + 1; i < len(tokens); i++ {
		if tokens[i] == "if" {
			ifIndex = i
			break
		}
	}

	iterable := ""
	cond := ""
	if ifIndex == -1 {
		iterable = strings.Join(tokens[inIndex+1:], " ")
	} else {
		iterable = strings.Join(tokens[inIndex+1:ifIndex], " ")
		cond = strings.Join(tokens[ifIndex+1:], " ")
		if cond == "" {
			return "", "", "", "", "", "", false
		}
	}
	if iterable == "" || exprPart == "" {
		return "", "", "", "", "", "", false
	}
	return assignKind, target, exprPart, iterVar, iterable, cond, true
}

func goParseTernary(expr string) (string, string, string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", "", "", false
	}
	qIndex := -1
	colonIndex := -1
	depth := 0
	var quote rune
	escaped := false
	for i, r := range expr {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"', '`':
			quote = r
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '?':
			if depth == 0 && qIndex == -1 {
				qIndex = i
			}
		case ':':
			if depth == 0 && qIndex != -1 {
				colonIndex = i
			}
		}
		if colonIndex != -1 {
			break
		}
	}
	if qIndex == -1 || colonIndex == -1 || colonIndex <= qIndex+1 {
		return "", "", "", false
	}
	cond := strings.TrimSpace(expr[:qIndex])
	trueExpr := strings.TrimSpace(expr[qIndex+1 : colonIndex])
	falseExpr := strings.TrimSpace(expr[colonIndex+1:])
	if cond == "" || trueExpr == "" || falseExpr == "" {
		return "", "", "", false
	}
	return cond, trueExpr, falseExpr, true
}

func goArrowLambdaReplacement(line string) (string, bool) {
	runes := []rune(line)
	if len(runes) < 3 {
		return "", false
	}

	depthAt := make([]int, len(runes))
	inQuoteAt := make([]bool, len(runes))
	depth := 0
	var quote rune
	escaped := false

	for i, r := range runes {
		depthAt[i] = depth
		inQuoteAt[i] = quote != 0

		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}

		switch r {
		case '\'', '"', '`':
			quote = r
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		}
	}

	arrow := -1
	arrowDepth := 0
	for i := 0; i < len(runes)-1; i++ {
		if inQuoteAt[i] {
			continue
		}
		if runes[i] == '=' && runes[i+1] == '>' {
			arrow = i
			arrowDepth = depthAt[i]
			break
		}
	}
	if arrow == -1 {
		return "", false
	}

	paramsStart := 0
	for i := arrow - 1; i >= 0; i-- {
		if inQuoteAt[i] {
			continue
		}
		if depthAt[i] < arrowDepth {
			paramsStart = i + 1
			break
		}
		if i == 0 {
			paramsStart = 0
		}
	}

	bodyEnd := len(runes)
	for i := arrow + 2; i < len(runes); i++ {
		if inQuoteAt[i] {
			continue
		}
		if depthAt[i] < arrowDepth {
			bodyEnd = i
			break
		}
		if depthAt[i] == arrowDepth && runes[i] == ',' {
			bodyEnd = i
			break
		}
	}

	params := strings.TrimSpace(string(runes[paramsStart:arrow]))
	body := strings.TrimSpace(string(runes[arrow+2 : bodyEnd]))
	if params == "" || body == "" {
		return "", false
	}

	hadParens := strings.HasPrefix(params, "(") && strings.HasSuffix(params, ")")
	if hadParens {
		params = strings.TrimSpace(params[1 : len(params)-1])
	}
	if params == "" {
		return "", false
	}

	params = goArrowLambdaParamsSegment(params, hadParens)
	if params == "" {
		return "", false
	}

	paramParts := strings.Split(params, ",")
	decls := make([]string, 0, len(paramParts))
	for _, part := range paramParts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", false
		}
		if strings.ContainsAny(part, " \t") {
			decls = append(decls, part)
		} else {
			decls = append(decls, part+" any")
		}
	}
	paramsDecl := strings.Join(decls, ", ")

	var replacement string
	if strings.HasPrefix(body, "{") && strings.HasSuffix(body, "}") {
		replacement = "func(" + paramsDecl + ") " + body
	} else {
		replacement = "func(" + paramsDecl + ") any { return " + body + " }"
	}

	newLine := string(runes[:paramsStart]) + replacement + string(runes[bodyEnd:])
	return newLine, true
}

func goArrowLambdaParamsSegment(params string, allowMultiple bool) string {
	parts := strings.Split(params, ",")
	if len(parts) <= 1 {
		trimmed := strings.TrimSpace(params)
		if trimmed == "" {
			return ""
		}
		if strings.ContainsAny(trimmed, " \t") {
			fields := strings.Fields(trimmed)
			if len(fields) == 0 || !isSimpleGoIdentifier(fields[0]) {
				return ""
			}
			return trimmed
		}
		if !isSimpleGoIdentifier(trimmed) {
			return ""
		}
		return trimmed
	}
	if !allowMultiple {
		last := strings.TrimSpace(parts[len(parts)-1])
		if last == "" || !isSimpleGoIdentifier(last) {
			return ""
		}
		return last
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return ""
		}
	}
	return strings.TrimSpace(params)
}

func isSimpleGoIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
		if i == 0 && unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

type goInContainerKind int

const (
	goInContainerUnknown goInContainerKind = iota
	goInContainerSlice
	goInContainerArray
	goInContainerMap
)

func (e *Editor) goInContainerKind(rhs, replacement string) goInContainerKind {
	filename := e.filename
	if filename == "" {
		filename = "input.go"
	}
	contents := e.contentsWithLineReplacement(e.LineIndex(), replacement)
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, filename, contents, parser.AllErrors)
	if file != nil {
		info := &types.Info{
			Types:  make(map[ast.Expr]types.TypeAndValue),
			Uses:   make(map[*ast.Ident]types.Object),
			Defs:   make(map[*ast.Ident]types.Object),
			Scopes: make(map[ast.Node]*types.Scope),
		}
		config := &types.Config{
			Importer: importer.Default(),
			Error:    func(error) {},
		}
		pkg, _ := config.Check(file.Name.Name, fset, []*ast.File{file}, info)
		if pkg != nil {
			pos := token.NoPos
			if tokFile := fset.File(file.Pos()); tokFile != nil {
				line := int(e.LineIndex()) + 1
				if line >= 1 && line <= tokFile.LineCount() {
					pos = tokFile.LineStart(line)
				}
			}
			if pos == token.NoPos {
				pos = file.Pos()
			}
			if tv, evalErr := types.Eval(fset, pkg, pos, rhs); evalErr == nil && tv.Type != nil {
				if kind := goInContainerKindFromType(tv.Type); kind != goInContainerUnknown {
					return kind
				}
			}
		}
	}
	return goInContainerKindFromExprString(rhs)
}

func (e *Editor) goEvalExprType(expr, replacement string) (types.Type, *types.Package, bool) {
	filename := e.filename
	if filename == "" {
		filename = "input.go"
	}
	contents := e.contentsWithLineReplacement(e.LineIndex(), replacement)
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, filename, contents, parser.AllErrors)
	if file == nil {
		return nil, nil, false
	}
	info := &types.Info{
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Uses:   make(map[*ast.Ident]types.Object),
		Defs:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	config := &types.Config{
		Importer: importer.Default(),
		Error:    func(error) {},
	}
	pkg, _ := config.Check(file.Name.Name, fset, []*ast.File{file}, info)
	if pkg == nil {
		return nil, nil, false
	}
	pos := token.NoPos
	if tokFile := fset.File(file.Pos()); tokFile != nil {
		line := int(e.LineIndex()) + 1
		if line >= 1 && line <= tokFile.LineCount() {
			pos = tokFile.LineStart(line)
		}
	}
	if pos == token.NoPos {
		pos = file.Pos()
	}
	tv, err := types.Eval(fset, pkg, pos, expr)
	if err != nil || tv.Type == nil {
		return nil, nil, false
	}
	return tv.Type, pkg, true
}

func (e *Editor) goListComprehensionElemType(exprPart, iterable string) (string, bool) {
	const replacement = "var _ = 0"
	if t, pkg, ok := e.goEvalExprType(exprPart, replacement); ok {
		return goTypeString(t, pkg), true
	}
	if t, pkg, ok := e.goEvalExprType(iterable, replacement); ok {
		if elem := goRangeValueType(t); elem != nil {
			return goTypeString(elem, pkg), true
		}
	}
	return "", false
}

func (e *Editor) goTernaryType(trueExpr, falseExpr string) (string, bool) {
	const replacement = "var _ = 0"
	if t, pkg, ok := e.goEvalExprType(trueExpr, replacement); ok {
		return goTypeString(t, pkg), true
	}
	if t, pkg, ok := e.goEvalExprType(falseExpr, replacement); ok {
		return goTypeString(t, pkg), true
	}
	return "", false
}

func goTypeString(t types.Type, currentPkg *types.Package) string {
	qualifier := func(p *types.Package) string {
		if currentPkg != nil && p.Path() == currentPkg.Path() {
			return ""
		}
		return p.Name()
	}
	return types.TypeString(t, qualifier)
}

func goRangeValueType(t types.Type) types.Type {
	if t == nil {
		return nil
	}
	switch tt := t.(type) {
	case *types.Named:
		return goRangeValueType(tt.Underlying())
	case *types.Pointer:
		return goRangeValueType(tt.Elem())
	case *types.Slice:
		return tt.Elem()
	case *types.Array:
		return tt.Elem()
	case *types.Map:
		return tt.Elem()
	case *types.Chan:
		return tt.Elem()
	case *types.Basic:
		if tt.Kind() == types.String {
			return types.Typ[types.Rune]
		}
	}
	if u := t.Underlying(); u != nil && u != t {
		return goRangeValueType(u)
	}
	return nil
}

func goInContainerKindFromType(t types.Type) goInContainerKind {
	switch tt := t.(type) {
	case *types.Named:
		return goInContainerKindFromType(tt.Underlying())
	case *types.Pointer:
		return goInContainerKindFromType(tt.Elem())
	case *types.Slice:
		return goInContainerSlice
	case *types.Array:
		return goInContainerArray
	case *types.Map:
		return goInContainerMap
	}
	if t != nil {
		if underlying := t.Underlying(); underlying != nil && underlying != t {
			return goInContainerKindFromType(underlying)
		}
	}
	return goInContainerUnknown
}

func goInContainerKindFromExprString(rhs string) goInContainerKind {
	expr, err := parser.ParseExpr(rhs)
	if err != nil {
		return goInContainerUnknown
	}
	return goInContainerKindFromExpr(expr)
}

func goInContainerKindFromExpr(expr ast.Expr) goInContainerKind {
	switch t := expr.(type) {
	case *ast.CompositeLit:
		return goInContainerKindFromTypeExpr(t.Type)
	case *ast.CallExpr:
		if ident, ok := t.Fun.(*ast.Ident); ok && len(t.Args) > 0 {
			if ident.Name == "make" || ident.Name == "new" {
				return goInContainerKindFromTypeExpr(t.Args[0])
			}
		}
	}
	return goInContainerUnknown
}

func goInContainerKindFromTypeExpr(expr ast.Expr) goInContainerKind {
	switch t := expr.(type) {
	case *ast.ArrayType:
		if t.Len == nil {
			return goInContainerSlice
		}
		return goInContainerArray
	case *ast.MapType:
		return goInContainerMap
	case *ast.StarExpr:
		return goInContainerKindFromTypeExpr(t.X)
	}
	return goInContainerUnknown
}

func (e *Editor) contentsWithLineReplacement(lineIndex LineIndex, replacement string) string {
	var sb strings.Builder
	l := e.Len()
	for i := 0; i < l; i++ {
		line := e.Line(LineIndex(i))
		if LineIndex(i) == lineIndex {
			line = replacement
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (e *Editor) ensureGoImport(pkg string) bool {
	if e.mode != mode.Go {
		return false
	}
	filename := e.filename
	if filename == "" {
		filename = "input.go"
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, e.String(), parser.ImportsOnly)
	if file == nil || err != nil {
		return false
	}
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, "\"") == pkg {
			return false
		}
	}
	currentLine := int(e.LineIndex()) + 1
	insertLine := 0
	lineText := ""
	var importDecl *ast.GenDecl
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			importDecl = gen
			break
		}
	}
	if importDecl == nil {
		insertLine = fset.Position(file.Package).Line
		lineText = "import \"" + pkg + "\""
	} else {
		lastSpec, ok := importDecl.Specs[len(importDecl.Specs)-1].(*ast.ImportSpec)
		if !ok {
			return false
		}
		insertLine = fset.Position(lastSpec.End()).Line
		if importDecl.Lparen.IsValid() {
			lineText = "\t\"" + pkg + "\""
		} else {
			lineText = "import \"" + pkg + "\""
		}
	}
	if insertLine <= 0 {
		return false
	}
	e.InsertLineBelowAt(LineIndex(insertLine - 1))
	e.SetLine(LineIndex(insertLine), lineText)
	if insertLine <= currentLine {
		e.pos.mut.Lock()
		e.pos.sy++
		e.pos.mut.Unlock()
	}
	return true
}

func (e *Editor) autocompleteGoIfErrQuestion(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if e.mode != mode.Go || !strings.HasSuffix(trimmedLine, "?") {
		return false
	}

	trimmedLine = strings.TrimSuffix(trimmedLine, "?")
	e.SetCurrentLine(currentLeadingWhitespace + trimmedLine)
	e.InsertLineBelow()
	e.pos.sy++

	ifErrBlock := e.ifErrBlockForCurrentFunction()
	for i, line := range strings.Split(strings.TrimSpace(ifErrBlock), "\n") {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func (e *Editor) autocompleteIfErr(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string) bool {
	if (e.mode != mode.Go && e.mode != mode.Odin) || trimmedLine != "iferr" {
		return false
	}

	ifErrBlock := e.ifErrBlockForCurrentFunction()
	// insert the block of text
	for i, line := range strings.Split(strings.TrimSpace(ifErrBlock), "\n") {
		if i != 0 {
			e.InsertLineBelow()
			e.pos.sy++
		}
		e.SetCurrentLine(currentLeadingWhitespace + line)
	}
	e.End(c)
	return true
}

func (e *Editor) ifErrBlockForCurrentFunction() string {
	oneIndentation := e.indentation.String()
	// default "if err != nil" block if iferr.IfErr can not find a more suitable one
	ifErrBlock := "if err != nil {\n" + oneIndentation + "return nil, err\n" + "}\n"
	// search backwards for "func ", return the full contents, the resulting line index and if it was found
	contents, functionLineIndex, found := e.ContentsAndReverseSearchPrefix("func ")
	if found {
		// count the bytes from the start to the end of the "func " line, since this is what iferr.IfErr uses
		byteCount := 0
		for i := LineIndex(0); i <= functionLineIndex; i++ {
			byteCount += len(e.Line(i))
		}
		// fetch a suitable "if err != nil" block for the current function signature
		if generatedIfErrBlock, err := iferr.IfErr([]byte(contents), byteCount); err == nil { // success
			ifErrBlock = generatedIfErrBlock
		}
	}
	return ifErrBlock
}

func (e *Editor) autocompleteTagExpansion(c *vt.Canvas, trimmedLine, currentLeadingWhitespace string, indent *bool, leadingWhitespace *string) bool {
	if (e.mode != mode.XML && e.mode != mode.HTML) || !e.expandTags || trimmedLine == "" {
		return false
	}
	if strings.Contains(trimmedLine, "<") || strings.Contains(trimmedLine, ">") {
		return false
	}
	if strings.ToLower(string(trimmedLine[0])) != string(trimmedLine[0]) {
		return false
	}

	// Words on a line without < or >? Expand into <tag asdf> above and </tag> below.
	words := strings.Fields(trimmedLine)
	tagName := words[0] // must be at least one word
	// the second word after the tag name needs to be ie. x=42 or href=...,
	// and the tag name must only contain letters a-z A-Z
	if (len(words) == 1 || strings.Contains(words[1], "=")) && onlyAZaz(tagName) {
		above := "<" + trimmedLine + ">"
		if tagName == "img" && !strings.Contains(trimmedLine, "alt=") && strings.Contains(trimmedLine, "src=") {
			// Pick out the image URI from the "src=" declaration
			imageURI := ""
			for _, word := range strings.Fields(trimmedLine) {
				if strings.HasPrefix(word, "src=") {
					imageURI = strings.SplitN(word, "=", 2)[1]
					imageURI = strings.TrimPrefix(imageURI, "\"")
					imageURI = strings.TrimSuffix(imageURI, "\"")
					imageURI = strings.TrimPrefix(imageURI, "'")
					imageURI = strings.TrimSuffix(imageURI, "'")
					break
				}
			}
			// If we got something that looks like an image URI, use the description before "." and capitalize it,
			// then use that as the default "alt=" declaration.
			if strings.Contains(imageURI, ".") {
				imageName := capitalizeWords(strings.TrimSuffix(imageURI, filepath.Ext(imageURI)))
				above = "<" + trimmedLine + " alt=\"" + imageName + "\">"
			}
		}
		// Now replace the current line
		e.SetCurrentLine(currentLeadingWhitespace + above)
		e.End(c)
		// And insert a line below
		e.InsertLineBelow()
		// Then if it's not an img tag, insert the closing tag below the current line
		if tagName != "img" {
			e.pos.mut.Lock()
			e.pos.sy++
			e.pos.mut.Unlock()
			below := "</" + tagName + ">"
			e.SetCurrentLine(currentLeadingWhitespace + below)
			e.pos.mut.Lock()
			e.pos.sy--
			e.pos.sx += 2
			e.pos.mut.Unlock()
			*indent = true
			*leadingWhitespace = e.indentation.String() + currentLeadingWhitespace
		}
		return true
	}

	return false
}

// LettersBeforeCursor returns the current word up until the cursor (for autocompletion)
func (e *Editor) LettersBeforeCursor() string {
	y := int(e.DataY())
	runes, ok := e.lines[y]

	if !ok {
		// This should never happen
		return ""
	}
	// Either find x or use the last index of the line
	x, err := e.DataX()
	if err != nil {
		x = len(runes)
	}

	qualifies := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
	}

	// Loop from the position before the current one and then leftwards on the current line.
	// Gather the letters.
	var word []rune
	for i := x - 1; i >= 0; i-- {
		r := runes[i]
		if !qualifies(r) {
			break
		}
		// Gather the letters in reverse
		word = append([]rune{r}, word...)
	}

	// Return the letters as a string
	return string(word)
}

// LettersOrDotBeforeCursor returns the current word up until the cursor (for autocompletion).
// Will also include ".".
func (e *Editor) LettersOrDotBeforeCursor() string {
	y := int(e.DataY())
	runes, ok := e.lines[y]
	if !ok {
		// This should never happen
		return ""
	}
	// Either find x or use the last index of the line
	x, err := e.DataX()
	if err != nil {
		x = len(runes)
	}

	qualifies := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.'
	}

	// Loop from the position before the current one and then leftwards on the current line.
	// Gather the letters.
	var word []rune
	for i := x - 1; i >= 0; i-- {
		r := runes[i]
		if !qualifies(r) {
			break
		}
		// Gather the letters in reverse
		word = append([]rune{r}, word...)
	}
	return string(word)
}
