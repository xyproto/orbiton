# Record a simple Haskell editing workflow with Orbiton
Output simple_haskell.gif

Require o
Require ghc
Require haskell-language-server-wrapper
Require ormolu

Source common/style.tape

# Open a new Haskell file
Type "o tmp/main.hs"
Sleep 2s
Enter
Sleep 100ms

# Insert Haskell template with ctrl-w
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

# Type "putS" to trigger HLS autocomplete for putStrLn
Type "putS"
Sleep 5s

# Tab to autocomplete with HLS
Tab
Wait+Screen /Completions/
Sleep 2s

# Select the second option (putStrLn)
Down

# Accept the completion suggestion
Enter
Sleep 100ms

# Complete the line: putStrLn("hey")
Type '"hey")'
Sleep 2s

# Compile and run again to see new output
Source common/run.tape

# Format, save and quit
Source common/fsq.tape
