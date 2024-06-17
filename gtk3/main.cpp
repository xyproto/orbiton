#include "fs.h"
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

static GtkWidget* terminal;
static GtkWidget* window;

const gdouble font_scale_step = 0.05;

void signal_and_quit()
{
    // Start by hiding the window
    gtk_widget_hide(window);

    // Handle subprocesses
    if (child_pid != -1) {
        if (!force_enable) {
            // If force was used at start, don't unlock the file.
            // Only unlock the file if force was not used at start.
            // Unlock the file by sending an unlock signal (USR1)
            kill(child_pid, SIGUSR1);
        }
        // This lets o save the file and then sleep a tiny bit, then quit the parent
        usleep(20000);
        kill(child_pid, SIGTERM);
    }
    gtk_main_quit();
}

void wait_and_quit()
{
    // Start by hiding the window
    gtk_widget_hide(window);
    // Handle subprocesses
    if (child_pid != -1 && !force_enable) {
        // If force was used at start, don't unlock the file.
        // Only unlock the file if force was not used at start.
        // Unlock the file by sending an unlock signal (USR1)
        kill(child_pid, SIGUSR1);
    }
    usleep(20000);
    gtk_main_quit();
}

// trigger_redraw will trigger a terminal resize event when the window is focused (SIGWINCH)
void trigger_redraw()
{
    // vte_terminal_set_size(VTE_TERMINAL(terminal), -1, -1);
    if (child_pid != -1) {
        kill(child_pid, SIGWINCH);
    }
}

void signal_handler(int signal_num)
{
    usleep(20000);
    // If we are here, it means we received a SIGWINCH signal from o.
    // This means that o is ready to resize.
    // Answer by sending SIGWINCH to o (the child_pid), to make it
    // resize itself according to the current VTE terminal size.
    trigger_redraw();
}

gboolean size_changed(GtkWidget* widget, GtkAllocation* allocation, void* data)
{
    // printf("new size %dx%d\n", allocation->width, allocation->height);
    trigger_redraw();
    gtk_widget_show(widget);
    return true; // event was handled
}

// Synthesized keypress events
static GdkEvent* ctrl_v_event = nullptr;
static GdkEvent* ctrl_p_event = nullptr;
static GdkEvent* ctrl_n_event = nullptr;
static GdkEvent* ctrl_l_event = nullptr;
static GdkEvent* return_key_event = nullptr;

// Synthesize an ctrl+v keypress if it has not been created,
// then send the event.
static gboolean send_ctrl_v_keypress(GtkWidget* widget)
{
    if (ctrl_v_event == nullptr) {
        ctrl_v_event = gdk_event_new(GDK_KEY_PRESS);
        if (ctrl_v_event == nullptr) {
            // ERROR: Can not allocate memory
            return false;
        }
        ctrl_v_event->key.keyval = GDK_KEY_v;
        ctrl_v_event->key.window = gtk_widget_get_window(widget);
        ctrl_v_event->key.length = 1;
        ctrl_v_event->key.send_event = true;
        ctrl_v_event->key.time = GDK_CURRENT_TIME;
        ctrl_v_event->key.state = GDK_CONTROL_MASK;
    }
    // Send the event
    gtk_main_do_event(ctrl_v_event);
    return true; // keypress was handled
}

// Synthesize a ctrl+p keypress if it has not been created,
// then send the event.
static gboolean send_scroll_up_keypress(GtkWidget* widget)
{
    if (ctrl_p_event == nullptr) {
        ctrl_p_event = gdk_event_new(GDK_KEY_PRESS);
        if (ctrl_p_event == nullptr) {
            // ERROR: Can not allocate memory
            return false;
        }
        ctrl_p_event->key.keyval = GDK_KEY_p;
        ctrl_p_event->key.window = gtk_widget_get_window(widget);
        ctrl_p_event->key.length = 1;
        ctrl_p_event->key.send_event = true;
        ctrl_p_event->key.time = GDK_CURRENT_TIME;
        ctrl_p_event->key.state = GDK_CONTROL_MASK;
    }
    // Send the event
    gtk_main_do_event(ctrl_p_event);
    return true; // keypress was handled
}

// Synthesize a ctrl+n keypress if it has not been created,
// then send the event.
static gboolean send_scroll_down_keypress(GtkWidget* widget)
{
    if (ctrl_n_event == nullptr) {
        ctrl_n_event = gdk_event_new(GDK_KEY_PRESS);
        if (ctrl_n_event == nullptr) {
            // ERROR: Can not allocate memory
            return false;
        }
        ctrl_n_event->key.keyval = GDK_KEY_n;
        ctrl_n_event->key.window = gtk_widget_get_window(widget);
        ctrl_n_event->key.length = 1;
        ctrl_n_event->key.send_event = true;
        ctrl_n_event->key.time = GDK_CURRENT_TIME;
        ctrl_n_event->key.state = GDK_CONTROL_MASK;
    }
    // Send the event
    gtk_main_do_event(ctrl_n_event);
    return true; // keypress was handled
}

// Synthesize a ctrl+l keypress if it has not been created,
// then send the event.
static gboolean send_ctrl_l(GtkWidget* widget)
{
    if (ctrl_l_event == nullptr) {
        ctrl_l_event = gdk_event_new(GDK_KEY_PRESS);
        if (ctrl_l_event == nullptr) {
            // ERROR: Can not allocate memory
            return false;
        }
        ctrl_l_event->key.keyval = GDK_KEY_l;
        ctrl_l_event->key.window = gtk_widget_get_window(widget);
        ctrl_l_event->key.length = 1;
        ctrl_l_event->key.send_event = true;
        ctrl_l_event->key.time = GDK_CURRENT_TIME;
        ctrl_l_event->key.state = GDK_CONTROL_MASK;
    }
    // Send the event
    gtk_main_do_event(ctrl_l_event);
    return true; // keypress was handled
}

// Synthesize a return keypress if it has not been created,
// then send the event.
static gboolean send_return(GtkWidget* widget)
{
    if (return_key_event == nullptr) {
        return_key_event = gdk_event_new(GDK_KEY_PRESS);
        if (return_key_event == nullptr) {
            // ERROR: Can not allocate memory
            return false;
        }
        return_key_event->key.keyval = GDK_KEY_Return;
        return_key_event->key.window = gtk_widget_get_window(widget);
        return_key_event->key.length = 1;
        return_key_event->key.send_event = true;
        return_key_event->key.time = GDK_CURRENT_TIME;
        return_key_event->key.state = 0; // GDK_CONTROL_MASK;
    }
    // Send the event
    gtk_main_do_event(return_key_event);
    return true; // keypress was handled
}

// mouse_clicked only receives and handles some mouse button events.
// Right mouse button and some special mouse buttons are received and handled.
gboolean mouse_clicked(GtkWidget* widget, GdkEventButton* event, gpointer data)
{
    if (event->button == 3) {
        // right mouse button, pastes with ctrl+v
        return send_ctrl_v_keypress(widget);
    }
    if (event->button == 9) {
        // special button forward, scrolls up (or to the previous search result) with ctrl+p
        return send_scroll_up_keypress(widget);
    }
    if (event->button == 8) {
        // special button back, scrolls down (or to the next search result) with ctrl+n
        return send_scroll_down_keypress(widget);
    }
    return false;
}

gboolean mouse_scrolled(GtkWidget* widget, GdkEventScroll* event)
{
    if (event->direction == GDK_SCROLL_UP) {
        return send_scroll_up_keypress(widget);
    }
    if (event->direction == GDK_SCROLL_DOWN) {
        return send_scroll_down_keypress(widget);
    }
    return false; // button is not handled
}

gboolean key_pressed(GtkWidget* widget, GdkEventKey* event, gpointer user_data)
{
    // for macOS and the cmd button
    if (event->state == 268435472) {
        event->state = GDK_CONTROL_MASK;
    }
    // for keys that are not handled by the o editor
    switch (event->keyval) {
    case GDK_KEY_Page_Up:
        // Send ctrl+p instead
        event->keyval = GDK_KEY_P;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_Page_Down:
        // Send ctrl+n instead
        event->keyval = GDK_KEY_N;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_Home:
        if (event->state == GDK_CONTROL_MASK) { // ctrl + home
            // Send ctrl+l,return, toggle between top and end
            // TODO: talk directly with o instead
            send_ctrl_l(widget);
            sleep(1.0);
            send_return(widget);
            return true;
        } else { // was home pressed, but without ctrl?
            // Send ctrl+a instead
            event->keyval = GDK_KEY_A;
            event->state = GDK_CONTROL_MASK;
        }
        break;
    case GDK_KEY_End:
        if (event->state == GDK_CONTROL_MASK) { // ctrl + end
            // Send ctrl+l,return, toggle between top and end
            // TODO: talk directly with o instead
            send_ctrl_l(widget);
            sleep(1.0);
            send_return(widget);
            return true;
        } else { // was end pressed, but without ctrl?
            // Send ctrl+e instead
            event->keyval = GDK_KEY_E;
            event->state = GDK_CONTROL_MASK;
        }
        break;
    case GDK_KEY_Delete:
        if (event->state == GDK_SHIFT_MASK) { // shift + delete, cut
            // Send ctrl+x instead
            event->keyval = GDK_KEY_X;
            event->state = GDK_CONTROL_MASK;
        } else {
            // Send ctrl+d instead
            event->keyval = GDK_KEY_D;
            event->state = GDK_CONTROL_MASK;
        }
        break;
    case GDK_KEY_Insert:
        if (event->state == GDK_SHIFT_MASK) { // shift + insert, paste
            // Send ctrl+v instead
            event->keyval = GDK_KEY_V;
            event->state = GDK_CONTROL_MASK;
        } else if (event->state == GDK_CONTROL_MASK) { // ctrl + insert, copy
            // Send ctrl+c instead
            event->keyval = GDK_KEY_C;
            event->state = GDK_CONTROL_MASK;
        } else {
            // Send return instead
            event->keyval = GDK_KEY_Return;
        }
        break;
    case GDK_KEY_F1:
        // Send ctrl+o instead, to show the menu
        event->keyval = GDK_KEY_O;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_F2:
        // Send ctrl+s instead, to save
        event->keyval = GDK_KEY_S;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_F3:
        // Send ctrl+n instead, to find next, if a search is ongoing
        event->keyval = GDK_KEY_N;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_F4:
        // Send ctrl+t instead, to toggle between C or C++ header/source files
        event->keyval = GDK_KEY_T;
        event->state = GDK_CONTROL_MASK;
        break;
    // F5 to F8 could be used for debugging!
    case GDK_KEY_F10:
        // Send ctrl+o, to give users the option of saving and quitting
        // F10 is too easy to press by accident to be bound to ctrl+q directly
        event->keyval = GDK_KEY_O;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_F12:
        // Send ctrl+r instead, to open/close a portal
        event->keyval = GDK_KEY_R;
        event->state = GDK_CONTROL_MASK;
        break;
    case GDK_KEY_plus:
    case GDK_KEY_KP_Add:
        // Increase the font scale, if ctrl was held
        if (event->state == GDK_CONTROL_MASK) {
            gdouble scale = vte_terminal_get_font_scale(VTE_TERMINAL(widget));
            vte_terminal_set_font_scale(VTE_TERMINAL(widget), scale + font_scale_step);
            return true; // keypress is handled to completion
        }
        break;
    case GDK_KEY_minus:
    case GDK_KEY_KP_Subtract:
        // Decrease the font scale, if ctrl was held
        if (event->state == GDK_CONTROL_MASK) {
            gdouble scale = vte_terminal_get_font_scale(VTE_TERMINAL(widget));
            vte_terminal_set_font_scale(VTE_TERMINAL(widget), scale - font_scale_step);
            return true; // keypress is handled to completion
        }
        break;
    }

    return false; // keypress is not handled to completion here
}

// file_contains checks if the given filename contains the given string x
bool file_contains(const std::string filename, const std::string x)
{
    std::ifstream t(filename);
    std::string contents;
    t.seekg(0, std::ios::end);
    contents.reserve(t.tellg());
    t.seekg(0, std::ios::beg);
    contents.assign((std::istreambuf_iterator<char>(t)), std::istreambuf_iterator<char>());
    return contents.find(x) != std::string::npos;
}

// env_str return the contents of an environment variable,
// but if the contents are empty, the default value is returned.
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
    using fs::exists;
    using fs::path;
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

// has_font_family checks if a font family for the given
// Pango font description string exists on the system.
bool has_font_family(const char* font_desc_str)
{
    auto chosen_font_description = pango_font_description_from_string(font_desc_str);
    const char* chosen_font_family = pango_font_description_get_family(chosen_font_description);
    std::string chosen_font_family_str = std::string(chosen_font_family);
    // List font families, thanks
    // https://gist.github.com/raimue/634213828f7ff86b9a6f4698ed488d85
    PangoFontFamily** families;
    int n_families;
    auto fontmap = pango_cairo_font_map_get_default();
    pango_font_map_list_families(fontmap, &families, &n_families);

    for (int n = 0; n < n_families; n++) {
        // Convert to a description and back, then to a std::string
        const char* x_family_name = pango_font_family_get_name(families[n]);
        const char* x_font_family
            = pango_font_description_get_family(pango_font_description_from_string(x_family_name));
        std::string x_font_family_str = std::string(x_font_family);

        // Compare the two strings, but skip spaces and compare letters case-insensitively
        bool equal = true;
        size_t i2 = 0;
        for (size_t i = 0; i < chosen_font_family_str.length(); i++) {
            if (i2 >= x_font_family_str.length()) {
                equal = false;
                break;
            }
            if (chosen_font_family_str.at(i) == ' ') {
                continue;
            }
            if (x_font_family_str.at(i2) == ' ') {
                i2++;
                i--;
                continue;
            }
            if (tolower(chosen_font_family_str.at(i)) != tolower(x_font_family_str.at(i2))) {
                equal = false;
                break;
            }
            i2++;
        }

        if (equal) {
            if (families != nullptr) {
                g_free(families);
            }
            return true;
        }
    }
    if (families != nullptr) {
        g_free(families);
    }
    return false;
}

// free_key_events will call gdk_event_free on the allocated static events
void free_key_events()
{
    if (ctrl_v_event != nullptr) {
        gdk_event_free(ctrl_v_event);
    }
    if (ctrl_p_event != nullptr) {
        gdk_event_free(ctrl_p_event);
    }
    if (ctrl_n_event != nullptr) {
        gdk_event_free(ctrl_n_event);
    }
    if (ctrl_l_event != nullptr) {
        gdk_event_free(ctrl_l_event);
    }
    if (return_key_event != nullptr) {
        gdk_event_free(return_key_event);
    }
}

int main(int argc, char* argv[])
{
    // Initialize Gtk, the window and the terminal
    gtk_init(&argc, &argv);

    // Create a new window and terminal
    window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    terminal = vte_terminal_new();

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

    std::filesystem::path full_path = (argc > 0) && argv[0] ? argv[0] : "";
    std::string exe_filename = full_path.filename().string();

    // Check if the executable starts with "l", "r", "s", "v" or "b"
    bool lightDefaultTheme = !exe_filename.empty() && exe_filename[0] == 'l';
    bool litmusTheme = !exe_filename.empty() && exe_filename[0] == 't';
    bool redBlackTheme = !exe_filename.empty() && exe_filename[0] == 'r';
    bool synthWaveMode = !exe_filename.empty() && exe_filename[0] == 's';
    bool tealTheme = !exe_filename.empty() && exe_filename[0] == 't';
    bool vsTheme = !exe_filename.empty() && exe_filename[0] == 'v';
    bool blueEditTheme = !exe_filename.empty() && (exe_filename[0] == 'b' || exe_filename[0] == 'e');

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

    using fs::exists;
    using fs::path;
    using fs::perms;
    using fs::status;

    // Search for the o executable in $PATH
    // Thanks https://stackoverflow.com/a/14571264
    // TODO: Extract to a "which" function
    char* dup = strdup(getenv("PATH"));
    char* s = dup;
    char* p = nullptr;
    path found { "o"s }; // name of executable to search for, can be symlink
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

    // Set the OG environment variable, which affects the behavior of "o"
    setenv("OG", "1", true);

    bool lightMode = false;
    bool lightPalette = false;

    // Setting SHELL to /bin/csh uses the red/black theme...
    if (redBlackTheme) {
        setenv("O_THEME", "redblack", true);
        lightPalette = false;
        lightMode = false;
    } else if (tealTheme) {
        setenv("O_THEME", "teal", true);
        lightPalette = false;
        lightMode = false;
    } else if (synthWaveMode) {
        setenv("O_THEME", "synthwave", true);
        lightPalette = false;
        lightMode = false; // makes no difference
    } else if (vsTheme) {
        setenv("O_THEME", "vs", true);
        lightPalette = true;
        lightMode = true;
    } else if (blueEditTheme) {
        setenv("O_THEME", "blueedit", true);
        lightPalette = false;
        lightMode = true;
    } else if (lightDefaultTheme) {
        setenv("O_THEME", "default", true);
        lightPalette = true;
        lightMode = true;
    } else if (litmusTheme) {
        setenv("O_THEME", "litmus", true);
        lightPalette = false;
        lightMode = false;
    }

    // Inform `o` that the version of the theme suitable for a palette with
    // a light background color should be used.
    if (lightMode) {
        setenv("O_LIGHT", "1", true);
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

    const auto pal_size = 16;
    const auto pal = (GdkRGBA*)malloc(sizeof(GdkRGBA) * pal_size);

    auto fg = GdkRGBA { 0.9, 0.9, 0.9, 1.0 };
    auto bg = GdkRGBA { 0.0, 0.0, 0.0, 1.0 };
    auto cb = GdkRGBA { 0.3, 0.7, 0.6, 0.9 }; // cursor block color

    if (!lightPalette) {

        // Inspired by the mterm color scheme
        pal[0] = { 0.23, 0.25, 0.32, 1.0 }; // black
        pal[1] = { 0.79, 0.34, 0.36, 1.0 }; // red, used for the "private" keyword
        pal[2] = { 0.68, 0.79, 0.59, 1.0 }; // green
        pal[3] = { 0.87, 0.74, 0.49, 1.0 }; // yellow
        pal[4] = { 0.55, 0.68, 0.80, 1.0 }; // blue
        pal[5] = { 0.70, 0.55, 0.67, 1.0 }; // magenta
        pal[6] = { 0.58, 0.80, 0.86, 1.0 }; // cyan
        pal[7] = { 0.94, 0.96, 0.99, 1.0 }; // light gray
        pal[8] = { 0.34, 0.38, 0.46, 1.0 }; // dark gray
        pal[9] = { 0.92, 0.30, 0.30, 1.0 }; // light red, used for keywords
        pal[10] = { 0.68, 0.80, 0.59, 1.0 }; // light green
        pal[11] = { 1.00, 0.90, 0.65, 1.0 }; // light yellow
        pal[12] = { 0.55, 0.68, 0.90, 1.0 }; // light blue
        pal[13] = { 0.75, 0.60, 0.72, 1.0 }; // light magenta
        pal[14] = { 0.61, 0.78, 0.78, 1.0 }; // light cyan
        pal[15] = { 0.90, 0.91, 0.93, 1.0 }; // white

    } else {
        fg = GdkRGBA { 0.01, 0.01, 0.01, 1.0 }; // dark foreground
        bg = GdkRGBA { 1.0, 1.0, 1.0, 1.0 }; // light background
        cb = GdkRGBA { 0.2, 0.2, 0.2, 0.9 }; // cursor block color

        pal[0] = { 0.0, 0.0, 0.0, 1.0 }; // black
        pal[1] = { 0.65, 0.14, 0.16, 1.0 }; // red, used for the "private" keyword (OK)
        pal[2] = { 0.0, 0.2, 0.05, 1.0 }; // green
        pal[3] = { 0.7, 0.44, 0.1, 1.0 }; // yellow
        pal[4] = { 0.04, 0.04, 0.25, 1.0 }; // blue
        pal[5] = { 0.5, 0.1, 0.1, 1.0 }; // magenta (now red)
        pal[6] = { 0.05, 0.2, 0.2, 1.0 }; // cyan
        pal[7] = { 0.0, 0.45, 0.45, 1.0 }; // light gray (now cyan) (used for numbers)
        pal[8] = { 0.02, 0.4, 0.02, 1.0 }; // dark gray (now green, used for comments)
        pal[9] = { 0.6, 0.05, 0.05, 1.0 }; // light red (used for keywords)
        pal[10] = { 0.0, 0.0, 0.5, 1.0 }; // light green (now blue)
        pal[11]
            = { 0.4, 0.02, 0.3, 1.0 }; // light yellow (now magenta, used for strings and stars)
        pal[12] = { 0.05, 0.05, 0.4, 1.0 }; // light blue
        pal[13] = { 0.02, 0.45, 0.45, 1.0 }; // light cyan (not used much)
        pal[14] = { 0.7, 0.1, 0.6, 1.0 }; // light magenta (not used much)
        pal[15] = { 0.8, 0.8, 0.8, 1.0 }; // white
    }

    // Adjustments

    if (blueEditTheme) { // && !lightPalette) {
        bg = GdkRGBA { 0.25, 0.32, 0.40, 1.0 }; // light background
        pal[4] = { 0.25, 0.32, 0.40, 1.0 }; // proper dark blue
    }

    vte_terminal_set_colors(VTE_TERMINAL(terminal), &fg, &bg, pal, 16);

    vte_terminal_set_color_cursor(VTE_TERMINAL(terminal), &cb);

    // Set cursor block text color
    const auto ct = GdkRGBA { 0.0, 0.0, 0.0, 0.9 };
    vte_terminal_set_color_cursor_foreground(VTE_TERMINAL(terminal), &ct);

    // Get the O_FONT environment variable, or nullptr
    const char* font_desc_str = std::getenv("O_FONT");

    // Also check if OG_FONT is set, if O_FONT was not set
    if (font_desc_str == nullptr) {
        font_desc_str = std::getenv("OG_FONT");
    }

    // Also check if KO_FONT is set, if no environment variable was specified
    if (font_desc_str == nullptr) {
        font_desc_str = std::getenv("KO_FONT");
    }

    // Set a default font if no environment variable was specified
    if (font_desc_str == nullptr) {
#ifdef __MACH__
        font_desc_str = "Menlo 16";
#else
        font_desc_str = "JetBrainsMonoNL 12";
#endif
    }

    // Check if the currently configured font can be found
    // TODO: Iterate over a strings to check for instead of this pyramid
    if (!has_font_family(font_desc_str)) {
        // If not, try to find a usable font
        font_desc_str = "Iosevka 12";
        if (!has_font_family(font_desc_str)) {
            font_desc_str = "Terminus 10";
            if (!has_font_family(font_desc_str)) {
                font_desc_str = "Monospace 10";
                if (!has_font_family(font_desc_str)) {
                    // Final font family fallback
                    font_desc_str = "Courier 10";
                }
            }
        }
    }

    auto chosen_font_description = pango_font_description_from_string(font_desc_str);
    vte_terminal_set_font(VTE_TERMINAL(terminal), chosen_font_description);

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
    g_signal_connect(terminal, "key-press-event", G_CALLBACK(key_pressed), nullptr);
    g_signal_connect(window, "scroll-event", G_CALLBACK(mouse_scrolled), nullptr);
    g_signal_connect(window, "button-press-event", G_CALLBACK(mouse_clicked), nullptr);
    g_signal_connect(window, "size-allocate", G_CALLBACK(size_changed), nullptr);

    // Add the terminal to the window
    gtk_container_add(GTK_CONTAINER(window), terminal);

    // Silence command line errors and warnings from Gtk.
    // This g_warning call in gdk_event_get_device spams stderr if not:
    // https://code.woboq.org/gtk/gtk/gdk/gdkevents.c.html#1666
    fclose(stderr);

    // Show the window
    gtk_widget_show_all(window);

    // Fullscreen
    if (!std::getenv("O_WINDOWED")) {
        gtk_window_fullscreen(GTK_WINDOW(window));
    }

    // Bring to front
    gtk_window_present(GTK_WINDOW(window));

    signal(SIGWINCH, signal_handler); // o sends SIGWINCH when it is ready to resize

    // Run the main loop
    gtk_main();

    free_key_events();

    return EXIT_SUCCESS;
}
