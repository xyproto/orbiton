# Record a simple Gleam editing workflow with Orbiton
Output simple_gleam.gif

Require o
Require gleam

Source common/style.tape

# Open a new Gleam file
Type "o tmp/main.gleam"
Enter
Sleep 500ms

# Insert Gleam template with ctrl-w
Ctrl+W
Sleep 1s

# Compile and run with double ctrl-space
Source common/run.tape

# Move cursor to the end of the line
Ctrl+E
Sleep 100ms

# Insert a new line
Enter
Sleep 100ms

# Type "io.p" to trigger gleam-lsp autocomplete for io.println
Type "io.p"
Sleep 3s

# Tab to autocomplete with gleam-lsp
Tab
Wait+Screen /Completions/
Sleep 1500ms

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the line: io.println("hey")
Type '"hey")'

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
