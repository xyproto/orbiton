# Record a simple Rust editing workflow with Orbiton
Output simple_rust.gif

Require o
Require rustc
Require rust-analyzer
Require rustfmt

Source common/style.tape

# Open a new Rust file
Type "o tmp/main.rs"
Sleep 2s
Enter
Sleep 100ms

# Insert Rust template with ctrl-w
Ctrl+W
Sleep 2s

# Compile and run with double ctrl-space
Source common/run.tape

# Move cursor to the end of the line
Ctrl+E
Sleep 100ms

# Insert a new line
Enter
Sleep 100ms

# Type "pri" to trigger rust-analyzer autocomplete for println!
Type "pri"
Sleep 2s

# Tab to autocomplete with rust-analyzer
Tab
Wait+Screen /Completions/
Sleep 2s

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the line: println!("hey");
Type '"hey");'
Sleep 2s

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
