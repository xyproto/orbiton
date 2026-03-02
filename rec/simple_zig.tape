# Record a simple Zig editing workflow with Orbiton
Output simple_zig.gif

Require o
Require zig
Require zls

Source style.tape

# Open a new Zig file
Type "o /tmp/main.zig"
Enter
Sleep 2s

# Insert Zig template with ctrl-w
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

# Type "std.debug.p" to trigger zls autocomplete for std.debug.print
Type "std.debug.p"
Sleep 3s

# Tab to autocomplete with zls
Tab
Wait+Screen /Completions/
Sleep 1s

# Select the second option
Down

# Accept the completion suggestion
Enter
Sleep 500ms

# Complete the line: std.debug.print("hey\n", .{});
Type '"hey'
Type '\n", .{});'
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
