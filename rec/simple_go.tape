# Record a simple Go editing workflow with Orbiton
Output simple_go.gif

Require o
Require go
Require gopls
Require goimports

Source common/style.tape

# Open a new Go file
Type "o tmp/main.go"
Enter
Sleep 500ms

# Insert Go template with ctrl-w
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

# Type "fmt.P" to trigger gopls autocomplete for fmt.Println
Type "fmt.P"
Sleep 3s

# Tab to autocomplete with gopls
Tab
Wait+Screen /Completions/
Sleep 1500ms

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the line: fmt.Println("hey")
Type '"hey")'

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
