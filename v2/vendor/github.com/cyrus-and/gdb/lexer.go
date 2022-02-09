package gdb

type tokenType int
type lexerState int

const (
	normal lexerState = iota
	inQuotation
	inEscape
)

type token struct {
	tokenType tokenType
	value     string
}

type parser struct {
	tokens <-chan token
	output map[string]interface{}
}

func lexer(input string) <-chan token { // no checks here...
	position := 0
	state := normal
	tokens := make(chan token)
	var value []byte
	go func() {
		for position < len(input) {
			next := input[position]
			switch state {
			case normal:
				switch next {
				case '^', '*', '+', '=', '~', '@', '&', ',', '{', '}', '[', ']':
					if value != nil {
						tokens <- token{tokenType(text), string(value)}
						value = nil
					}
					tokens <- token{tokenType(next), string(next)}
				case '"':
					state = inQuotation
					if value != nil {
						tokens <- token{tokenType(text), string(value)}
						value = nil
					}
				default:
					value = append(value, next)
				}
			case inQuotation:
				switch next {
				case '"':
					state = normal
					if value != nil {
						tokens <- token{tokenType(text), string(value)}
					} else {
						tokens <- token{tokenType(text), ""}
					}
					value = nil
				case '\\':
					state = inEscape
				default:
					value = append(value, next)
				}
			case inEscape:
				switch next {
				case 'a':
					next = '\a'
				case 'b':
					next = '\b'
				case 'f':
					next = '\f'
				case 'n':
					next = '\n'
				case 'r':
					next = '\r'
				case 't':
					next = '\t'
				case 'v':
					next = '\v'
				case '\\':
					next = '\\'
				case '\'':
					next = '\''
				case '"':
					next = '"'
				}
				value = append(value, next)
				state = inQuotation
			}
			position++
		}
		if value != nil {
			tokens <- token{tokenType(text), string(value)}
			value = nil
		}
		close(tokens)
	}()
	return tokens
}

func (p *parser) Lex(lval *yySymType) int {
	// fetch the next token
	token, ok := <-p.tokens
	if ok {
		// save the value and return the token type
		lval.text = token.value
		return int(token.tokenType)
	} else {
		return 0 // no more tokens
	}
}

func (p *parser) Error(err string) {
	// errors are GDB bugs if the grammar is correct
	panic(err)
}

func parseRecord(data string) map[string]interface{} {
	parser := parser{lexer(data), map[string]interface{}{}}
	yyParse(&parser)
	return parser.output
}
