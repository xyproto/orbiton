# Terminal Title

Change the title if the currently running terminal emulator supports it.

## Currently supported terminal emulators

* `konsole`
* `alacritty`
* `gnome-terminal`

For unsupported terminal emulators, the `MustSet` function will try the same terminal codes as for `gnome-terminal`.

## Example use

~~~go
package main

import (
    "github.com/xyproto/termtitle"
)

func main() {
    termtitle.Set("TESTING 1 2 3")
}
~~~

## Terminal codes

For `konsole` a working string seems to be:

    \033]0;TITLE\a

While for `gnome-terminal`, this one works:

    \033]30;TITLE\007

For `alacritty`, this seems to work:

    \033]2;TITLE\007

`TITLE` is the title that will be set.

## General info

* Version: 1.5.1
* License: BSD-3
