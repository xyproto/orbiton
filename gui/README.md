# The o GUI

This is a GUI for o, written in C++, using the VTE library.

Tested with VTE 2.91 on Arch Linux.

## Building

If `o` and [`cxx`](https://github.com/xyproto/cxx) are installed, running `o main.cpp` and then pressing `ctrl+space` is enough to build the GUI application.

Alternatively, use the `Makefile` in the parent directory:

    make gui

## Installation

### Linux

In the parent directory, run:

    make gui-install

## Font configuration

The font can be set via the `GUI_FONT` environment variable, like this:

    export GUI_FONT="iosevka 16"
