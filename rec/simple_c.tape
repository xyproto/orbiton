# Record a simple C editing workflow with Orbiton
Output simple_c.gif

Require o
Require gcc
Require clangd
Require clang-format

Source common/style.tape

# Open a new C file
Type "o tmp/main.c"
Sleep 2s
Enter
Sleep 100ms

# Insert C template with ctrl-w
Ctrl+W
Sleep 2s

# Compile and run with double ctrl-space
Source common/run.tape

# Move cursor to the end of the line
Ctrl+E
Sleep 100ms

# Insert a new line
Enter
Sleep 100ms

# Type "pr" to trigger clangd autocomplete
Type "pr"
Sleep 3s

# Tab to autocomplete with clangd
Tab
Wait+Screen /Completions/
Sleep 2s

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the printf line: printf("hey\n");
Type '"hey'
Type '\n");'
Sleep 2s

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
