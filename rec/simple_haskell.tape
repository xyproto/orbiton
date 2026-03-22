# Record a simple Haskell editing workflow with Orbiton
Output simple_haskell.gif

Require o
Require ghc
Require haskell-language-server-wrapper
Require ormolu

Source common/style.tape

# Open a new Haskell file
Type "o tmp/main.hs"
Enter
Sleep 500ms

# Insert Haskell template with ctrl-w
Ctrl+W
Sleep 1s

# Compile and run with double ctrl-space
Source common/run.tape

# Go to the beginning of line 2 ("main = putStrLn ...")
Ctrl+A
Sleep 100ms

# Move right past "main = " (7 chars) to insert "do" before "putStrLn"
Right 7

# Insert "do\n  " to create a do block, pushing putStrLn to next indented line
Type "do"
Enter
Type "  "
Sleep 100ms

# Move to end of the putStrLn line
Ctrl+E
Sleep 100ms

# Add a new line for the second putStrLn
Enter
Sleep 100ms

# Type "putStr" to trigger HLS autocomplete for putStrLn
Type "putStr"
Sleep 5s

# Tab to autocomplete with HLS
Tab
Wait+Screen /Completions/
Sleep 1500ms

# Accept the first completion suggestion (putStrLn)
Enter
Sleep 100ms

# Complete the line: putStrLn "hey"
Type '"hey"'

# Pause to show the completed code
Sleep 1500ms

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
