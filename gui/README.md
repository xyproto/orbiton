# The o GUI

This is a GUI for o, written in C++, using the VTE library.

Tested with VTE 2.91 on Arch Linux.

## Build

Compile with [cxx](https://github.com/xyproto/cxx):

    cxx

Or use the `Makefile` in the parent directory:

    make gui

## Install

Install with, for instance:

    sudo install -Dm755 og /usr/bin/gui

Or, for Linux, use the `Makefile` in the parent directory:

    make install-gui

## Font configuration

The font can be set via the `GUI_FONT` environment variable, like this:

    export GUI_FONT="iosevka 16"
