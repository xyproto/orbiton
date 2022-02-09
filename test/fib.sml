(* single line comment *)

fun fib (n: C_Int.int) : C_Int.int = (* single line comment *)
  if n <= 1
     then 1
     else fib (n - 1) + fib (n - 2)

(*
multi
line
comment
*)

val () = (_export "fib" public: (C_Int.int -> C_Int.int) -> unit;) fib (* multi
line
comment *)
