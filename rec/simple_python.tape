# Record a simple Python editing workflow with Orbiton
Output simple_python.gif

Require o
Require python3
Require pylsp

Source common/style.tape

# Open a new Python file
Type "o tmp/main.py"
Enter
Sleep 500ms

# Insert Python template with ctrl-w
Ctrl+W
Sleep 1s

# Compile and run with double ctrl-space
Source common/run.tape

# Move to the end of the line: print("Hello, World!")
Ctrl+E
Sleep 100ms

# Insert a new line
Enter
Sleep 100ms

# Type "pri" to trigger pylsp autocomplete for print
Type "pri"
Sleep 3s

# Tab to autocomplete with pylsp
Tab
Wait+Screen /Completions/
Sleep 1500ms

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the line: print("hey")
Type '"hey")'

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
