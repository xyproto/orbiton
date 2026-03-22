# Record a simple Bash editing workflow with Orbiton
Output simple_bash.gif

Require o
Require bash
Require bash-language-server
Require shfmt

Source common/style.tape

# Open a new Bash script
Type "o tmp/main.sh"
Enter
Sleep 500ms

# Insert Bash template with ctrl-w
Ctrl+W
Sleep 1s

# Run the program with double ctrl-space
Source common/run.tape

# Move to the end of the line
Ctrl+E
Sleep 100ms

# Insert two new lines
Enter
Enter
Sleep 100ms

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
Sleep 1500ms

# Select the second option
Down
Sleep 100ms

# Accept the completion suggestion
Enter
Sleep 100ms

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
