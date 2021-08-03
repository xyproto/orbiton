# The o GUI

This is a GUI for o, written in C++, using the GTK toolkit.

Tested with GTK+4 on Arch Linux.

Compile with [cxx](https://github.com/xyproto/cxx):

    cxx

Install with, for instance:

    sudo install -Dm755 og /usr/bin/gui

The font can be set with the `GUI_FONT` environment variable, like this:

    export GUI_FONT="iosevka 16"
