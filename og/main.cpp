#include <cstdio>
#include <filesystem>
#include <iostream>
#include <string>
#include <vte/vte.h>

/*
 * A terminal emulator only for runnin o.
 * Inspired by: https://vincent.bernat.ch/en/blog/2017-write-own-terminal
 */

using namespace std::string_literals;

// new_window creates and returns a Gtk window.
// The given title is used as the window title.
auto new_window(std::string const& title)
{
    auto window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(window), title.c_str());
    return window;
}

// new_color parses the given string and returns a GdkRGBA struct
// which must be freed after use.
auto new_color(std::string const& c)
{
    auto colorstruct = (GdkRGBA*)malloc(sizeof(GdkRGBA));
    gdk_rgba_parse(colorstruct, c.c_str());
    return colorstruct;
}

void eof() { std::cout << "bye" << std::endl; }

int main(int argc, char* argv[])
{
    // Initialize Gtk, the window and the terminal
    gtk_init(&argc, &argv);

    // Open README.md by default, if no filename is given
    auto filename = "README.md"s;
    if (argc > 1) {
        filename = argv[1];
    }

    auto window = new_window(filename);
    auto terminal = vte_terminal_new();

    using std::filesystem::exists;
    using std::filesystem::path;
    using std::filesystem::perms;
    using std::filesystem::status;

    // Search for the o executable in $PATH
    // Thanks https://stackoverflow.com/a/14571264
    // TODO: Extract to a "which" function
    char* dup = strdup(getenv("PATH"));
    char* s = dup;
    char* p = nullptr;
    path found { "o"s }; // name of executable to search for, may be mutated
    do {
        p = strchr(s, ':');
        if (p != nullptr) {
            p[0] = 0;
        }
        if (exists(path(s) / found)) {
            found = path(s) / found;
            break;
        }
        s = p + 1;
    } while (p != nullptr);
    free(dup);

    // Check again if the executable exists
    if (found == "o"s) {
        std::cerr << found << " does not exist in PATH" << std::endl;
        return EXIT_FAILURE;
    }

    // Build an array of strings, which is the command to be run
    const char* command[3];
    command[0] = found.c_str();
    command[1] = filename.c_str();
    command[2] = nullptr;

    // Check if the executable is executable
    const auto perm = status(command[0]).permissions();
    if ((perm & perms::owner_exec) == perms::none) {
        std::cerr << command[0] << " is not executable for this user" << std::endl;
        return EXIT_FAILURE;
    }

    // Spawn a terminal
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
    vte_terminal_spawn_sync(VTE_TERMINAL(terminal), VTE_PTY_DEFAULT,
        nullptr, // working directory
        (char**)command, // command
        nullptr, // environment
        (GSpawnFlags)0, // spawn flags
        nullptr, nullptr, // child setup
        nullptr, // child PID
        nullptr, nullptr);
#pragma GCC diagnostic pop

    // Set background color to 95% opaque black
    auto black = new_color("rgba(0, 0, 0, 0.95)"s);
    vte_terminal_set_color_background(VTE_TERMINAL(terminal), black);
    free(black);

    // Set foreground color
    //auto green = new_color("chartreuse"s);
    //vte_terminal_set_color_foreground(VTE_TERMINAL(terminal), green);
    //free(green);

    // Set font
    auto font_desc = pango_font_description_from_string("terminus 14");
    vte_terminal_set_font(VTE_TERMINAL(terminal), font_desc);

    // Set cursor shape to BLOCK
    vte_terminal_set_cursor_shape(VTE_TERMINAL(terminal), VTE_CURSOR_SHAPE_BLOCK);

    // Set cursor blink to OFF
    vte_terminal_set_cursor_blink_mode(VTE_TERMINAL(terminal), VTE_CURSOR_BLINK_OFF);

    // Connect some signals
    g_signal_connect(window, "delete-event", gtk_main_quit, nullptr);
    g_signal_connect(terminal, "child-exited", gtk_main_quit, nullptr);
    g_signal_connect(terminal, "eof", eof, nullptr);

    // Add the terminal to the window
    gtk_container_add(GTK_CONTAINER(window), terminal);

    // Show the window and run the Gtk event loop
    gtk_widget_show_all(window);
    gtk_main();

    return EXIT_SUCCESS;
}
