# Record a simple C editing workflow with Orbiton
Output simple_c.gif

Require o
Require gcc
Require clangd

Source style.tape

# Open a new C file
Type "o /tmp/main.c"
Enter
Sleep 2s

# Insert C template with ctrl-w
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

# Type "pr" to trigger clangd autocomplete
Type "pr"
Sleep 3s

# Tab to autocomplete with clangd
Tab
Wait+Screen /Completions/
Sleep 1s

# Accept the completion suggestion
Enter
Sleep 500ms

# Complete the printf line: printf("hey\n");
Type '"hey'
Type '\n");'
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
