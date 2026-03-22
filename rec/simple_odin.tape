# Record a simple Odin editing workflow with Orbiton
Output simple_odin.gif

Require o
Require odin
Require odinfmt

Source common/style.tape

# Open a new Odin file
Type "o tmp/main.odin"
Enter
Sleep 500ms

# Insert Odin template with ctrl-w
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

# Type a second fmt.println line
Type 'fmt.println("hey");'

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
