// -*- go -*-
%{
    package gdb

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
			typeKey: typeString,
			classKey: class,
		}
		if payload != nil {
			out[payloadKey] = payload
		}
		return out
	}
%}

%union{
	text         string
	record       map[string]interface{}
	class_result struct{class string; payload map[string]interface{}}
	result_pair  struct{variable string; value interface{}}
	value        interface{}
	list         []interface{}
}

%token text

%type <text>         text token
%type <record>       result_list record async_record stream_record result_record tuple
%type <class_result> class_result
%type <result_pair>  result
%type <value>        value
%type <list>         list value_list tuple_list

%%

all:
	record { yylex.(*parser).output = $1 };

record:
	async_record |
	stream_record |
	result_record;

async_record:
	token '*' class_result { $$ = newClassResult("exec",   $3.class, $3.payload) } |
	token '+' class_result { $$ = newClassResult("status", $3.class, $3.payload) } |
	token '+' text ',' tuple { $$ = newClassResult("status", $3, $5) } |
	token '=' class_result { $$ = newClassResult("notify", $3.class, $3.payload) };

class_result:
	text ',' result_list { $$.class, $$.payload = $1, $3 } |
	text                 { $$.class, $$.payload = $1, nil };

stream_record:
	'~' text { $$ = map[string]interface{}{typeKey: "console", payloadKey: $2} } |
	'@' text { $$ = map[string]interface{}{typeKey: "target",  payloadKey: $2} } |
	'&' text { $$ = map[string]interface{}{typeKey: "log",     payloadKey: $2} };

result_record:
	token '^' class_result
	{
		$$ = map[string]interface{}{sequenceKey: $1, classKey: $3.class}
		if $3.payload != nil { $$[payloadKey] = $3.payload }
	};

result_list:
	result_list ',' result { $$[$3.variable] = $3.value } |
	result                 { $$ = map[string]interface{}{$1.variable: $1.value} };

token:
	     { $$ = "" } |
	text { $$ = $1 }

result:
	text '=' value { $$.variable, $$.value = $1, $3 };

value:
	text  { $$ = $1 } |
	tuple { $$ = $1 } |
	list  { $$ = $1 };

value_list:
	value_list ',' value { $$ = append($$, $3) } |
	value                { $$ = []interface{}{$1} };

tuple:
	'{' result_list '}' { $$ = $2 } |
	'{' '}'             { $$ = map[string]interface{}{} };

tuple_list:
	tuple_list ',' result { $$ = append($$, map[string]interface{}{$3.variable: $3.value}) } |
	result                { $$ = []interface{}{map[string]interface{}{$1.variable: $1.value} } };

list:
	'[' value_list ']' { $$ = $2 } |
	'[' tuple_list ']' { $$ = $2 } |
	'[' ']'            { $$ = []interface{}{} };
