# Record a simple C++ editing workflow with Orbiton
Output simple_cpp.gif

Require o
Require clangd
Require clang-format

Source common/style.tape

# Open a new C++ file
Type "o tmp/main.cpp"
Enter
Sleep 500ms

# Insert C++ template with ctrl-w
Ctrl+W
Sleep 1s

# Compile and run with double ctrl-space
Source common/run.tape

# Move cursor to the end of the line
Ctrl+E
Sleep 100ms

# Insert a new line
Enter
Sleep 100ms

# Type "std::co" to trigger clangd autocomplete for std::cout
Type "std::co"
Sleep 3s

# Tab to autocomplete with clangd
Tab
Wait+Screen /Completions/
Sleep 1500ms

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the line: std::cout << "hey"s << std::endl;
Type ' << "hey"s << std::endl;'

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
