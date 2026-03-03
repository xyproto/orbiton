# Record a simple Bash editing workflow with Orbiton
Output simple_bash.gif

Require o
Require bash
Require bash-language-server
Require shfmt

Source style.tape

# Open a new Bash script
Type "o /tmp/main.sh"
Sleep 1s
Enter
Sleep 1s

# Insert Bash template with ctrl-w
Ctrl+W
Sleep 1s

# Run the program with double ctrl-space
Source run.tape

# Move to the end of the line: print("Hello, World!")
Ctrl+E
Sleep 500ms

# Insert two new lines
Enter
Enter
Sleep 500ms

# Insert more text
Type "for x in 1 2 3; do"
Enter
Type "echo $x"
Enter
Type "do"
Sleep 1s

# Tab to autocomplete "do" to "done"
Tab
Wait+Screen /Completions/
Sleep 500ms

# Select the second option
Down
Sleep 500ms

# Accept the completion suggestion
Enter
Sleep 500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
