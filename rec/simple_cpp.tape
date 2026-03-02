# Record a simple C++ editing workflow with Orbiton
Output simple_cpp.gif

Require o
Require clang++
Require clangd

Set FontSize 14
Set Width 1200
Set Height 600
Set Shell "bash"
Set TypingSpeed 30ms

# Open a new C++ file
Type "o /tmp/main.cpp"
Enter
Sleep 1s

# Insert C++ template with ctrl-w
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

# Type "std::co" to trigger clangd autocomplete for std::cout
Type "std::co"
Sleep 3s

# Tab to autocomplete with clangd
Tab
Sleep 2s

# Accept the completion suggestion
Enter
Sleep 500ms

# Complete the line: std::cout << "hey"s << std::endl;
Type ' << "hey"s << std::endl;'
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

# Filter to show only main.cpp
Type "main.cpp"
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
