# Record a simple Rust editing workflow with Orbiton
Output simple_rust.gif

Require o
Require rustc
Require rust-analyzer

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set TypingSpeed 30ms

# Open a new Rust file
Type "o /tmp/main.rs"
Enter
Sleep 1s

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
Sleep 3s

# Move cursor down
Down
Sleep 500ms

# Insert a new line
Enter
Sleep 500ms

# Move cursor up
Up
Sleep 500ms

# Insert indentation
Tab
Sleep 500ms

# Type "pri" to trigger rust-analyzer autocomplete for println!
Type "pri"
Sleep 8s

# Tab to autocomplete with rust-analyzer
Tab
Sleep 3s

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

# Open file browser on /tmp
Type "o /tmp"
Enter
Sleep 1s

# Filter to show only main.rs
Type "main.rs"
Sleep 1s

# Select the file
Down
Sleep 500ms

# Delete the file
Delete
Sleep 1s

# Quit the file browser
Ctrl+Q
Sleep 1s
