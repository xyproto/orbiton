#include <cstdio>
#include <filesystem>
#include <iostream>
#include <signal.h>
#include <string>
#include <unistd.h>
#include <vte/vte.h>

/*
 * A terminal emulator only for running o.
 * Inspired by: https://vincent.bernat.ch/en/blog/2017-write-own-terminal
 */

using namespace std::string_literals;

static GPid child_pid = -1;

// new_window creates and returns a Gtk window.
// The given title is used as the window title.
auto new_window(std::string const& title)
{
    auto window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(window), title.c_str());
    return window;
}

void signal_and_quit()
{
    if (child_pid != -1) {
        // This lets o save the file and then quit
        kill(child_pid, SIGTERM);
    }
    gtk_main_quit();
}

void wait_and_quit()
{
    sleep(1);
    gtk_main_quit();
}

int main(int argc, char* argv[])
{
    // Initialize Gtk, the window and the terminal
    gtk_init(&argc, &argv);

    // Open README.md by default, if no filename is given
    auto filename = "README.md"s;
    auto flag = ""s;
    if (argc > 2) {
        flag = argv[1];
        filename = argv[2];
    } else if (argc > 1) {
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
    const char* command[4];
    command[0] = found.c_str();
    if (flag == "") {
        command[1] = filename.c_str();
        command[2] = nullptr;
    } else {
        command[1] = flag.c_str();
        command[2] = filename.c_str();
        command[3] = nullptr;
    }

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
        &child_pid, // child PID
        nullptr, nullptr);
#pragma GCC diagnostic pop

    // std::cout << "PID " << child_pid << std::endl;

    const auto fg = GdkRGBA { 0.9, 0.9, 0.9, 1.0 };
    const auto bg = GdkRGBA { 0.1, 0.1, 0.1, 1.0 };

    const auto pal_size = 16;
    const auto pal = (GdkRGBA*)malloc(sizeof(GdkRGBA) * pal_size);

    // Inspired by the mterm color scheme
    pal[0] = { 0.28, 0.30, 0.37, 1.0 };
    pal[1] = { 0.79, 0.43, 0.46, 1.0 };
    pal[2] = { 0.68, 0.79, 0.59, 1.0 };
    pal[3] = { 0.97, 0.84, 0.59, 1.0 };
    pal[4] = { 0.55, 0.68, 0.80, 1.0 };
    pal[5] = { 0.70, 0.55, 0.67, 1.0 };
    pal[6] = { 0.58, 0.80, 0.86, 1.0 };
    pal[7] = { 0.94, 0.96, 0.99, 1.0 };
    pal[8] = { 0.34, 0.38, 0.46, 1.0 };
    pal[9] = { 0.79, 0.43, 0.46, 1.0 };
    pal[10] = { 0.68, 0.80, 0.59, 1.0 };
    pal[11] = { 0.97, 0.84, 0.59, 1.0 };
    pal[12] = { 0.55, 0.68, 0.90, 1.0 };
    pal[13] = { 0.75, 0.60, 0.72, 1.0 };
    pal[14] = { 0.61, 0.78, 0.78, 1.0 };
    pal[15] = { 0.90, 0.91, 0.93, 1.0 };

    vte_terminal_set_colors(VTE_TERMINAL(terminal), &fg, &bg, pal, 16);

    // Set cursor block color
    const auto cb = GdkRGBA { 0.3, 0.7, 0.6, 0.9 };
    vte_terminal_set_color_cursor(VTE_TERMINAL(terminal), &cb);

    // Set cursor block text color
    const auto ct = GdkRGBA { 0.0, 0.0, 0.0, 0.9 };
    vte_terminal_set_color_cursor_foreground(VTE_TERMINAL(terminal), &ct);

    // Set font
    const char* font_desc_str = std::getenv("GUI_FONT");
    if (font_desc_str == nullptr) {
        font_desc_str = "terminus 10"s.c_str(); // the default font
    }
    vte_terminal_set_font(
        VTE_TERMINAL(terminal), pango_font_description_from_string(font_desc_str));

    // Config
    vte_terminal_set_scrollback_lines(VTE_TERMINAL(terminal), 0);
    vte_terminal_set_scroll_on_output(VTE_TERMINAL(terminal), FALSE);
    vte_terminal_set_scroll_on_keystroke(VTE_TERMINAL(terminal), FALSE);
    vte_terminal_set_mouse_autohide(VTE_TERMINAL(terminal), TRUE);
    vte_terminal_set_allow_hyperlink(VTE_TERMINAL(terminal), TRUE);

    // Set cursor shape to BLOCK
    vte_terminal_set_cursor_shape(VTE_TERMINAL(terminal), VTE_CURSOR_SHAPE_BLOCK);

    // Set cursor blink to OFF
    vte_terminal_set_cursor_blink_mode(VTE_TERMINAL(terminal), VTE_CURSOR_BLINK_OFF);

    // Connect some signals
    g_signal_connect(window, "delete-event", wait_and_quit, nullptr);
    g_signal_connect(terminal, "child-exited", signal_and_quit, nullptr);

    // Add the terminal to the window
    gtk_container_add(GTK_CONTAINER(window), terminal);

    // Show the window and run the Gtk event loop
    gtk_widget_show_all(window);
    gtk_main();

    return EXIT_SUCCESS;
}
