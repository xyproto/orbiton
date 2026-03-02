# Record a simple Rust editing workflow with Orbiton
Output simple_rust.gif

Require o
Require rustc
Require rust-analyzer

Source style.tape

# Open a new Rust file
Type "o /tmp/main.rs"
Enter
Sleep 2s

# Insert Rust template with ctrl-w
Ctrl+W
Sleep 1s

# Save with ctrl-s
Ctrl+S
Sleep 500ms

# Compile and run with double ctrl-space
Ctrl+Space
Sleep 200ms
Ctrl+Space
Wait+Screen /Program output/
Sleep 1s

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
Ctrl+Space
Sleep 200ms
Ctrl+Space
Sleep 4s

# Save
Ctrl+S
Sleep 500ms

# Quit the editor
Ctrl+Q
Sleep 1s
