//line grammar.y:4
package gdb

import __yyfmt__ "fmt"

//line grammar.y:4
const (
	terminator  = "(gdb) " // yes there's the trailing space
	typeKey     = "type"
	classKey    = "class"
	payloadKey  = "payload"
	sequenceKey = "sequence"
)

// avoid DRY due to a poor lexer
func newClassResult(typeString, class string, payload map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{
		typeKey:  typeString,
		classKey: class,
	}
	if payload != nil {
		out[payloadKey] = payload
	}
	return out
}

//line grammar.y:27
type yySymType struct {
	yys          int
	text         string
	record       map[string]interface{}
	class_result struct {
		class   string
		payload map[string]interface{}
	}
	result_pair struct {
		variable string
		value    interface{}
	}
	value interface{}
	list  []interface{}
}

const text = 57346

var yyToknames = []string{
	"text",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 32
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 59

var yyAct = []int{

	27, 36, 26, 38, 47, 50, 31, 18, 37, 48,
	28, 32, 25, 30, 51, 40, 44, 30, 49, 40,
	20, 22, 23, 31, 28, 24, 28, 19, 21, 29,
	41, 17, 35, 33, 34, 30, 11, 12, 16, 13,
	10, 46, 45, 14, 15, 7, 8, 9, 1, 43,
	52, 53, 42, 39, 5, 4, 3, 2, 6,
}
var yyPact = []int{

	36, -1000, -1000, -1000, -1000, -1000, 31, 40, 34, 27,
	-1000, 23, 24, 23, 23, -1000, -1000, -1000, -1000, 18,
	-1000, 5, -1000, -1000, 6, 22, -1, -1000, 3, -1000,
	20, 6, 4, 16, -1000, -1000, -1000, -1000, -1000, -1000,
	0, -1000, 2, -2, -1000, -1000, -1000, 3, 4, -1000,
	6, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 58, 2, 57, 56, 55, 54, 3, 7, 0,
	1, 53, 52, 49, 48,
}
var yyR1 = []int{

	0, 14, 3, 3, 3, 4, 4, 4, 4, 8,
	8, 5, 5, 5, 6, 2, 2, 1, 1, 9,
	10, 10, 10, 12, 12, 7, 7, 13, 13, 11,
	11, 11,
}
var yyR2 = []int{

	0, 1, 1, 1, 1, 3, 3, 5, 3, 3,
	1, 2, 2, 2, 3, 3, 1, 0, 1, 3,
	1, 1, 1, 3, 1, 3, 2, 3, 1, 3,
	3, 2,
}
var yyChk = []int{

	-1000, -14, -3, -4, -5, -6, -1, 9, 10, 11,
	4, 5, 6, 8, 12, 4, 4, 4, -8, 4,
	-8, 4, -8, -8, 7, 7, -2, -9, 4, -7,
	13, 7, 8, -2, 14, -9, -10, 4, -7, -11,
	15, 14, -12, -13, 16, -10, -9, 4, 7, 16,
	7, 16, -10, -9,
}
var yyDef = []int{

	17, -2, 1, 2, 3, 4, 0, 0, 0, 0,
	18, 0, 0, 0, 0, 11, 12, 13, 5, 10,
	6, 10, 8, 14, 0, 0, 9, 16, 0, 7,
	0, 0, 0, 0, 26, 15, 19, 20, 21, 22,
	0, 25, 0, 0, 31, 24, 28, 20, 0, 29,
	0, 30, 23, 27,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 11, 3,
	3, 3, 5, 6, 7, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 8, 3, 3, 10, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 15, 3, 16, 12, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 13, 3, 14, 9,
}
var yyTok2 = []int{

	2, 3, 4,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(yyToknames) {
		if yyToknames[c-4] != "" {
			return yyToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(c), uint(char))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		//line grammar.y:48
		{
			yylex.(*parser).output = yyS[yypt-0].record
		}
	case 2:
		yyVAL.record = yyS[yypt-0].record
	case 3:
		yyVAL.record = yyS[yypt-0].record
	case 4:
		yyVAL.record = yyS[yypt-0].record
	case 5:
		//line grammar.y:56
		{
			yyVAL.record = newClassResult("exec", yyS[yypt-0].class_result.class, yyS[yypt-0].class_result.payload)
		}
	case 6:
		//line grammar.y:57
		{
			yyVAL.record = newClassResult("status", yyS[yypt-0].class_result.class, yyS[yypt-0].class_result.payload)
		}
	case 7:
		//line grammar.y:58
		{
			yyVAL.record = newClassResult("status", yyS[yypt-2].text, yyS[yypt-0].record)
		}
	case 8:
		//line grammar.y:59
		{
			yyVAL.record = newClassResult("notify", yyS[yypt-0].class_result.class, yyS[yypt-0].class_result.payload)
		}
	case 9:
		//line grammar.y:62
		{
			yyVAL.class_result.class, yyVAL.class_result.payload = yyS[yypt-2].text, yyS[yypt-0].record
		}
	case 10:
		//line grammar.y:63
		{
			yyVAL.class_result.class, yyVAL.class_result.payload = yyS[yypt-0].text, nil
		}
	case 11:
		//line grammar.y:66
		{
			yyVAL.record = map[string]interface{}{typeKey: "console", payloadKey: yyS[yypt-0].text}
		}
	case 12:
		//line grammar.y:67
		{
			yyVAL.record = map[string]interface{}{typeKey: "target", payloadKey: yyS[yypt-0].text}
		}
	case 13:
		//line grammar.y:68
		{
			yyVAL.record = map[string]interface{}{typeKey: "log", payloadKey: yyS[yypt-0].text}
		}
	case 14:
		//line grammar.y:72
		{
			yyVAL.record = map[string]interface{}{sequenceKey: yyS[yypt-2].text, classKey: yyS[yypt-0].class_result.class}
			if yyS[yypt-0].class_result.payload != nil {
				yyVAL.record[payloadKey] = yyS[yypt-0].class_result.payload
			}
		}
	case 15:
		//line grammar.y:78
		{
			yyVAL.record[yyS[yypt-0].result_pair.variable] = yyS[yypt-0].result_pair.value
		}
	case 16:
		//line grammar.y:79
		{
			yyVAL.record = map[string]interface{}{yyS[yypt-0].result_pair.variable: yyS[yypt-0].result_pair.value}
		}
	case 17:
		//line grammar.y:82
		{
			yyVAL.text = ""
		}
	case 18:
		//line grammar.y:83
		{
			yyVAL.text = yyS[yypt-0].text
		}
	case 19:
		//line grammar.y:86
		{
			yyVAL.result_pair.variable, yyVAL.result_pair.value = yyS[yypt-2].text, yyS[yypt-0].value
		}
	case 20:
		//line grammar.y:89
		{
			yyVAL.value = yyS[yypt-0].text
		}
	case 21:
		//line grammar.y:90
		{
			yyVAL.value = yyS[yypt-0].record
		}
	case 22:
		//line grammar.y:91
		{
			yyVAL.value = yyS[yypt-0].list
		}
	case 23:
		//line grammar.y:94
		{
			yyVAL.list = append(yyVAL.list, yyS[yypt-0].value)
		}
	case 24:
		//line grammar.y:95
		{
			yyVAL.list = []interface{}{yyS[yypt-0].value}
		}
	case 25:
		//line grammar.y:98
		{
			yyVAL.record = yyS[yypt-1].record
		}
	case 26:
		//line grammar.y:99
		{
			yyVAL.record = map[string]interface{}{}
		}
	case 27:
		//line grammar.y:102
		{
			yyVAL.list = append(yyVAL.list, map[string]interface{}{yyS[yypt-0].result_pair.variable: yyS[yypt-0].result_pair.value})
		}
	case 28:
		//line grammar.y:103
		{
			yyVAL.list = []interface{}{map[string]interface{}{yyS[yypt-0].result_pair.variable: yyS[yypt-0].result_pair.value}}
		}
	case 29:
		//line grammar.y:106
		{
			yyVAL.list = yyS[yypt-1].list
		}
	case 30:
		//line grammar.y:107
		{
			yyVAL.list = yyS[yypt-1].list
		}
	case 31:
		//line grammar.y:108
		{
			yyVAL.list = []interface{}{}
		}
	}
	goto yystack /* stack new state and value */
}
