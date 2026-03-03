# Record a simple Rust editing workflow with Orbiton
Output simple_rust.gif

Require o
Require rustc
Require rust-analyzer
Require rustfmt

Source style.tape

# Open a new Rust file
Type "o /tmp/main.rs"
Sleep 1s
Enter
Sleep 1s

# Insert Rust template with ctrl-w
Ctrl+W
Sleep 1s

# Compile and run with double ctrl-space
Source common/run.tape

# Move cursor to the end of the line
Ctrl+E
Sleep 500ms

# Insert a new line
Enter
Sleep 500ms

# Type "pri" to trigger rust-analyzer autocomplete for println!
Type "pri"
Sleep 8s

# Tab to autocomplete with rust-analyzer
Tab
Wait+Screen /Completions/
Sleep 1s

# Accept the completion suggestion
Enter
Sleep 500ms

# Complete the line: println!("hey");
Type '"hey");'
Sleep 1s

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
