// Package gdb provides a convenient way to interact with the GDB/MI
// interface. The methods offered by this module are very low level, the main
// goals are:
//
// - avoid the tedious parsing of the MI2 line-based text interface;
//
// - bypass a known bug(https://sourceware.org/bugzilla/show_bug.cgi?id=8759)
//   which prevents to distinguish the target program's output from MI2 records.
//
// The objects returned as a result of the commands or as asynchronous
// notifications are generic Go maps suitable to be converted to JSON format
// with json.Unmarshal(). The fields present in such objects are blindly added
// according to the records returned from GDB (see
// https://sourceware.org/gdb/onlinedocs/gdb/GDB_002fMI-Output-Syntax.html):
// tuples are map[string]interface{} and lists are []interface{}. There are a
// couple of exceptions to this:
//
// - the record class, where present, is represented by the "class" field;
//
// - the record type is represented using the "type" field as follows:
//     "+": "status"
//     "=": "notify"
//     "~": "console"
//     "@": "target"
//     "&": "log"
//
// - the optional result list is stored into a tuple under the "payload" field.
package gdb

//go:generate goyacc -o grammar.go grammar.y
