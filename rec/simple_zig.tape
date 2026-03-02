# Record a simple Zig editing workflow with Orbiton
Output simple_zig.gif

Require o
Require zig
Require zls

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set TypingSpeed 30ms

# Open a new Zig file
Type "o /tmp/main.zig"
Enter
Sleep 1s

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
Sleep 3s

# Move cursor down
Down
Sleep 500ms

# Insert a new line
Enter
Sleep 500ms

# Move cursor up
Up
Sleep 500ms

# Insert indentation
Tab
Sleep 500ms

# Type "std.debug.p" to trigger zls autocomplete for std.debug.print
Type "std.debug.p"
Sleep 3s

# Tab to autocomplete with zls
Tab
Sleep 2s

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

# Open file browser on /tmp
Type "o /tmp"
Enter
Sleep 1s

# Filter to show only main.zig
Type "main.zig"
Sleep 1s

# Select the file
Down
Sleep 500ms

# Delete the file
Delete
Sleep 1s

# Quit the file browser
Ctrl+Q
Sleep 1s
