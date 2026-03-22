# Record a Go debug workflow with Orbiton (using the -d flag)
Output debug_go.gif

Require o
Require go
Require dlv

Source common/style.tape

# Open a new Go file and insert the template
Type "o tmp/main.go"
Sleep 2s
Enter
Sleep 200ms

# Insert Go template with ctrl-w
Ctrl+W
Sleep 2s

# Move up to the opening brace and add the declaration before fmt.Println
Up
Ctrl+E
Sleep 100ms
Enter
Sleep 100ms
Type "x := 42"
Down
Ctrl+E
Enter
Type "x = 256"
Enter
Type "fmt.Println(x)"
Sleep 1s

# Save the file
Ctrl+S
Sleep 200ms

# Open the ctrl-o menu
Ctrl+O
Sleep 100ms

# Select debug mode
Type e
Enter
Sleep 100ms

# Disable keybinding overview
Ctrl+K
Sleep 500ms

# Start the debug session: compile with debug info and launch Delve
Ctrl+Space
Wait+Screen /Running/
Sleep 500ms

# Step over the first line (x := 42 is now executed and in scope)
Ctrl+O
Sleep 2s

# Add a watch for x with ctrl-w
Ctrl+W
Sleep 100ms
Type "x"
Enter
Wait+Screen /x:/
Sleep 500ms

# Step over the fmt.Println line
Ctrl+O
Sleep 2s

# Step
Ctrl+O
Sleep 1s

# Step
Ctrl+O
Sleep 1s

# Step
Ctrl+O
Sleep 2s

# Exit debug mode with ctrl-q, then quit the editor
Ctrl+Q
Ctrl+Q
