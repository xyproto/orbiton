#include <cstdio>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <signal.h>
#include <stdlib.h>
#include <streambuf>
#include <string>
#include <unistd.h>
#include <vte/vte.h>

/*
 * A terminal emulator only for running o.
 * Inspired by: https://vincent.bernat.ch/en/blog/2017-write-own-terminal
 */

using namespace std::string_literals;

static GPid child_pid = -1; // PID of the child process, the o editor, or -1

static bool force_enable = false; // was the file locked, so that the -f flag was used?

void signal_and_quit()
{
    if (child_pid != -1) {
        if (!force_enable) {
            // If force was used at start, don't unlock the file.
            // Only unlock the file if force was not used at start.
            // Unlock the file by sending an unlock signal (USR1)
            kill(child_pid, SIGUSR1);
            sleep(0.5);
        }
        // This lets o save the file and then sleep a tiny bit, then quit the parent
        kill(child_pid, SIGTERM);
        sleep(0.5);
    }
    gtk_main_quit();
}

void wait_and_quit()
{
    if (child_pid != -1 && !force_enable) {
        // If force was used at start, don't unlock the file.
        // Only unlock the file if force was not used at start.
        // Unlock the file by sending an unlock signal (USR1)
        kill(child_pid, SIGUSR1);
        sleep(0.5);
    }
    sleep(0.5);
    gtk_main_quit();
}

bool file_contains(const std::string lock_filename, const std::string x)
{
    // Open the lock file, using the given lock_filename
    std::ifstream t(lock_filename);
    std::string contents;
    t.seekg(0, std::ios::end);
    contents.reserve(t.tellg());
    t.seekg(0, std::ios::beg);
    contents.assign((std::istreambuf_iterator<char>(t)), std::istreambuf_iterator<char>());
    // Check if the file contents contains the given string x
    return contents.find(x) != std::string::npos;
}

// Return the contents of an environment variable, but
// if the contents are empty, return the given default value.
std::string env_str(std::string env_name, std::string default_value)
{
    char* e = std::getenv(env_name.c_str());
    if (e == nullptr) {
        return default_value;
    }
    return std::string(e);
}

// is_locked checks if the given filename is found in either
// ~/.cache/o/lockfile.txt or $XDG_CACHE_DIR/o/lockefile.txt.
bool is_locked(std::string filename)
{
    using std::filesystem::exists;
    using std::filesystem::path;
    path xdg_cache_dir(env_str("XDG_CACHE_DIR"s, "."s));
    path home_dir(env_str("HOME"s, "."s));
    path xdg_cache_lockfile = xdg_cache_dir / path("o/lockfile.txt"s);
    path home_lockfile = home_dir / path(".cache/o/lockfile.txt"s);
    if (exists(xdg_cache_lockfile)) {
        return file_contains(xdg_cache_lockfile, filename);
    }
    if (exists(home_lockfile)) {
        return file_contains(home_lockfile, filename);
    }
    return false;
}

int main(int argc, char* argv[])
{
    // Initialize Gtk, the window and the terminal
    gtk_init(&argc, &argv);

    // Create a new window and terminal
    auto window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    auto terminal = vte_terminal_new();

    // The file to edit
    std::string filename;

    // Gather flags and filename arguments
    bool givenFilename = false;
    auto flag = ""s;
    if (argc > 2) {
        flag = argv[1];
        filename = argv[2];
        givenFilename = true;
    } else if (argc > 1) {
        filename = argv[1];
        givenFilename = true;
    }

    // Show the file chooser dialog, if no filename was given
    if (!givenFilename) {
        auto dialog = gtk_file_chooser_dialog_new("Open File", GTK_WINDOW(window),
            GTK_FILE_CHOOSER_ACTION_OPEN, "_Cancel", GTK_RESPONSE_CANCEL, "_Open",
            GTK_RESPONSE_ACCEPT, nullptr);
        if (gtk_dialog_run(GTK_DIALOG(dialog)) == GTK_RESPONSE_ACCEPT) {
            char* selectedFilename = gtk_file_chooser_get_filename(GTK_FILE_CHOOSER(dialog));
            filename = std::string(selectedFilename);
            g_free(selectedFilename);
        } else {
            // Did not get GTK_RESPONSE_ACCEPT, just end the program here
            // gtk_widget_destroy(dialog);
            // gtk_main_quit();
            return EXIT_FAILURE;
        }
        gtk_widget_destroy(dialog);
    }

    // Set the Window title
    gtk_window_set_title(GTK_WINDOW(window), filename.c_str());

    // Set the default Window size
    // gtk_window_set_default_size(GTK_WINDOW(window), 800, 600);

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
        if (is_locked(filename)) {
            force_enable = true;
            command[1] = "-f";
            command[2] = filename.c_str();
            command[3] = nullptr;
        } else {
            command[1] = filename.c_str();
            command[2] = nullptr;
        }
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
    pal[0] = { 0.23, 0.25, 0.32, 1.0 }; // black
    pal[1] = { 0.79, 0.34, 0.36, 1.0 }; // red, used for the "private" keyword
    pal[2] = { 0.68, 0.79, 0.59, 1.0 }; // green
    pal[3] = { 0.97, 0.84, 0.59, 1.0 }; // yellow
    pal[4] = { 0.55, 0.68, 0.80, 1.0 }; // blue
    pal[5] = { 0.70, 0.55, 0.67, 1.0 }; // magenta
    pal[6] = { 0.58, 0.80, 0.86, 1.0 }; // cyan
    pal[7] = { 0.94, 0.96, 0.99, 1.0 }; // light gray
    pal[8] = { 0.34, 0.38, 0.46, 1.0 }; // dark gray
    pal[9] = { 0.92, 0.30, 0.30, 1.0 }; // light red, used for keywords
    pal[10] = { 0.68, 0.80, 0.59, 1.0 }; // light green
    pal[11] = { 0.97, 0.84, 0.59, 1.0 }; // light yellow
    pal[12] = { 0.55, 0.68, 0.90, 1.0 }; // light blue
    pal[13] = { 0.75, 0.60, 0.72, 1.0 }; // light magenta
    pal[14] = { 0.61, 0.78, 0.78, 1.0 }; // light cyan
    pal[15] = { 0.90, 0.91, 0.93, 1.0 }; // white

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
    g_signal_connect(window, "destroy", wait_and_quit, nullptr);
    g_signal_connect(window, "delete-event", wait_and_quit, nullptr);
    g_signal_connect(terminal, "child-exited", signal_and_quit, nullptr);

    // Add the terminal to the window
    gtk_container_add(GTK_CONTAINER(window), terminal);

    // Show the window and run the Gtk event loop
    gtk_widget_show_all(window);
    gtk_main();

    return EXIT_SUCCESS;
}
