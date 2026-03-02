# Record a simple Python editing workflow with Orbiton
Output simple_python.gif

Require o
Require python3
Require pylsp

Source style.tape

# Open a new Python file
Type "o /tmp/main.py"
Enter
Sleep 2s

# Insert Python template with ctrl-w
Ctrl+W
Sleep 1s

# Save with ctrl-s
Ctrl+S
Sleep 500ms

# Run with double ctrl-space
Ctrl+Space
Sleep 200ms
Ctrl+Space
Wait+Screen /Program output/
Sleep 2s

# Move to the end of the line: print("Hello, World!")
Ctrl+E
Sleep 500ms

# Insert a new line
Enter
Sleep 500ms

# Type "pri" to trigger pylsp autocomplete for print
Type "pri"
Sleep 3s

# Tab to autocomplete with pylsp
Tab
Wait+Screen /Completions/
Sleep 1s

# Accept the completion suggestion
Enter
Sleep 500ms

# Complete the line: print("hey")
Type '"hey")'
Sleep 1s

# Compile and run again to see new output
Ctrl+Space
Sleep 200ms
Ctrl+Space
Wait+Screen /Program output/
Sleep 4s

# Save
Ctrl+S
Sleep 500ms

# Quit the editor
Ctrl+Q
Sleep 1s
