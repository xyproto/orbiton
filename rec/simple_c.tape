# Record a simple C editing workflow with Orbiton
Output simple_c.gif

Require o
Require gcc
Require clangd
Require clang-format

Source common/style.tape

# Open a new C file
Type "o /tmp/main.c"
Sleep 1s
Enter
Sleep 1s

# Insert C template with ctrl-w
Ctrl+W
Sleep 1s

# Compile and run with double ctrl-space
Source common/run.tape

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
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
