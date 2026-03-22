# Record a C debug workflow with Orbiton (using the -d flag)
Output debug_c.gif

Require o
Require gcc
Require gdb

Source common/style.tape

# Open a new C file and insert the template
Type "o tmp/main.c"
Enter
Sleep 500ms

# Insert C template with ctrl-w
Ctrl+W
Sleep 1s

# Move up to the opening brace and add a local variable before printf
Up
Ctrl+E
Enter
Type "int x = 256;"
Down
Ctrl+E
Enter
Type "x = 42;"
Enter
Type 'printf("%d\n", x);'

# Pause to let the viewer read the code
Sleep 1500ms

# Save the file
Ctrl+S
Sleep 300ms

# Open the Ctrl+O menu
Ctrl+O
Sleep 100ms

# Select "Debug mode"
Type e
Enter
Sleep 100ms

# Disable keybinding overview
Ctrl+K
Sleep 1s

# Start the debug session: compile with debug info and launch GDB
Ctrl+Space
Wait+Screen /Running/
Sleep 1500ms

# Add a watch for x with ctrl-w
Ctrl+W
Sleep 200ms
Type "x"
Enter
Wait+Screen /x:/
Sleep 1500ms

# Step over the first line (x = 42 is now executed and in scope)
Ctrl+O
Sleep 2s

# Step over the printf line
Ctrl+O
Sleep 1s

# Step over the x = 256 line
Ctrl+O
Sleep 1s

# Step over the printf
Ctrl+O
Sleep 2s

# Step over the return line
Ctrl+O
Sleep 1s

# Step one final time, to end the debugging
Ctrl+O
Sleep 2s

# Exit debug mode with ctrl-q, then quit the editor
Ctrl+Q
Ctrl+Q
