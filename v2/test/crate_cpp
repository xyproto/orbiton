// This program draws and manipulates a 3D crate using basic object operations such as rotation, translation, and scaling.
// It handles basic graphics rendering, loading a BMP image, and window management in a Windows environment.

#include <cstdio>
#include <cstdlib>
#include <sys/stat.h>
#include <windows.h>

#define _USE_MATH_DEFINES
#include <cmath>

// Define window and image buffer dimensions
#define WIN_WIDTH 640 // window width
#define WIN_HEIGHT 400 // window height
#define IMGBUFF_WIDTH 256 // maximum BMP image buffer width
#define IMGBUFF_HEIGHT 256 // maximum BMP image buffer height

// Define constants for image and object counts
#define IMGOBJCNT 1 // number of image objects
#define OBJCNT 1 // number of objects
#define OPCNT 128 // max number of operations on an object

// Define constants for identifying image and object
#define CRATEIMG 0 // image ID for the crate
#define CRATE 0 // object ID for the crate

// Define operation codes for object transformations
#define IMGOBJ -1 // indicate image operation
#define HIDE 0 // hide the object
#define XROT 1 // rotate around X axis
#define YROT 2 // rotate around Y axis
#define ZROT 3 // rotate around Z axis
#define XLOC 4 // translate along X axis
#define YLOC 5 // translate along Y axis
#define ZLOC 6 // translate along Z axis
#define XSZ 7 // scale along X axis
#define YSZ 8 // scale along Y axis
#define ZSZ 9 // scale along Z axis
#define END 0 // end of operations

// Constants for timers and states
#define TRUE 1
#define FALSE 0
#define ID_TIMER 1 // timer identifier for screen refresh

// Global variables for canvas and image data
int canvasWidth; // current canvas width
int canvasHeight; // current canvas height
int xCenter, yCenter; // center coordinates of the canvas
BITMAPINFO pbmi[40]; // bitmap information
BYTE canvas[(WIN_WIDTH * 3 + WIN_WIDTH % 4) * WIN_HEIGHT]; // pixel data for canvas

int imgObj[IMGOBJCNT][IMGBUFF_WIDTH * IMGBUFF_HEIGHT]; // array for storing image pixel data
int objOps[OBJCNT][OPCNT]; // operations to be performed on each object
float objOpValues[OBJCNT][OPCNT]; // values for each object operation

// Buffers for storing precomputed sine and cosine values
float hUcRotValues[OPCNT]; // horizontal unit circle rotation values
float vUcRotValues[OPCNT]; // vertical unit circle rotation values

// Helper function to round a floating-point number
inline double round(double val) { return floor(val + 0.5); }

// Loads a BMP image file into an image object buffer
void loadImg(int imgObjNum, char* imgFile)
{
    int x, y;
    struct stat stat_p;
    FILE* bmpFile;
    int imgWidth;
    int headerSize;

    // Check if the file exists
    if (-1 == stat(imgFile, &stat_p))
        return;

    // Open the BMP file in binary mode
    bmpFile = fopen(imgFile, "rb");
    if (!bmpFile)
        return;

    // Calculate image width (must be square and power of 2)
    imgWidth = pow(2, (int)(log(sqrt(stat_p.st_size / 3)) / log(2)));

    // Compute the header size and skip the header in the BMP file
    headerSize = stat_p.st_size - imgWidth * imgWidth * 3;
    fseek(bmpFile, headerSize + 1, SEEK_CUR);

    // Read the pixel data from the BMP file and store it in imgObj buffer
    for (y = IMGBUFF_HEIGHT / 2 - imgWidth / 2; y < imgWidth + IMGBUFF_HEIGHT / 2 - imgWidth / 2; y++) {
        for (x = IMGBUFF_WIDTH / 2 - imgWidth / 2; x < imgWidth + IMGBUFF_WIDTH / 2 - imgWidth / 2; x++) {
            if ((x >= 0) && (x < IMGBUFF_WIDTH) && (y >= 0) && (y < IMGBUFF_HEIGHT))
                imgObj[imgObjNum][IMGBUFF_WIDTH * y + x] = (int)fgetc(bmpFile) / 128 * 255;

            // Skip the padding in BMP files (since BMP has 24 bits per pixel, 3 bytes)
            fseek(bmpFile, 2, SEEK_CUR);
        }
    }

    // Close the BMP file after loading
    fclose(bmpFile);
}

// Rotates a point around the origin by a specified angle (in degrees)
void rot(float* horiP, float* vertP, float degrees)
{
    if (degrees != degrees) // check if NaN
        return;

    // Calculate unit circle coordinates for the given degrees
    float hUc = cos(degrees * (M_PI * 2.0 / 360.0));
    float vUc = sin(degrees * (M_PI * 2.0 / 360.0));

    // Perform the rotation on the given point
    float h = *vertP * (-vUc) + *horiP * hUc;
    float v = *horiP * vUc + *vertP * hUc;

    // Store the rotated coordinates
    *horiP = h;
    *vertP = v;
}

// Rotate a point using precomputed unit circle values
void ucRot(float hUc, float vUc, float* hP, float* vP)
{
    if (hUc != hUc || vUc != vUc) // check if NaN
        return;

    // Rotate the point using the given unit circle coordinates
    float h = *vP * (-vUc) + *hP * hUc;
    float v = *hP * vUc + *vP * hUc;

    // Store the rotated coordinates
    *hP = h;
    *vP = v;
}

// Apply a series of transformations (operations) to an object and compute new positions and brightness
void applyObjOps(int objNum, int opNum, int x, int y, int* xDelta, int* yDelta, int* brightness)
{
    int i;
    float xPt = x - IMGBUFF_WIDTH / 2;
    float yPt = y - IMGBUFF_HEIGHT / 2;
    float zPt = 0;
    float perspctv = 350; // perspective distance
    float cameraLens = 200; // camera lens focal length
    float cameraDistance = -200; // distance from camera

    // Apply each operation to the object
    for (i = opNum; i < OPCNT; i++) {
        if (objOps[objNum][i] == END || objOps[objNum][i] == IMGOBJ)
            break;

        // Apply rotations
        if (objOps[objNum][i] == XROT)
            ucRot(hUcRotValues[i], vUcRotValues[i], &yPt, &zPt);

        if (objOps[objNum][i] == YROT)
            ucRot(hUcRotValues[i], vUcRotValues[i], &xPt, &zPt);

        if (objOps[objNum][i] == ZROT)
            ucRot(hUcRotValues[i], vUcRotValues[i], &xPt, &yPt);

        // Apply translations
        if (objOps[objNum][i] == XLOC)
            xPt += objOpValues[objNum][i];

        if (objOps[objNum][i] == YLOC)
            yPt += objOpValues[objNum][i];

        if (objOps[objNum][i] == ZLOC)
            zPt += objOpValues[objNum][i];

        // Apply scaling
        if (objOps[objNum][i] == XSZ)
            xPt *= objOpValues[objNum][i];

        if (objOps[objNum][i] == YSZ)
            yPt *= objOpValues[objNum][i];

        if (objOps[objNum][i] == ZSZ)
            yPt *= objOpValues[objNum][i];
    }

    // Calculate final screen coordinates with perspective projection
    *xDelta = round(xPt * perspctv / (perspctv - zPt) + xCenter);
    *yDelta = round(yPt * perspctv / (perspctv - zPt) + yCenter);

    // Calculate brightness based on object's distance from the camera
    if (zPt >= 0)
        *brightness = 128 + zPt / cameraLens * 127;
    else
        *brightness = 128 - zPt / cameraDistance * 127;

    if (zPt > cameraLens || zPt < cameraDistance)
        *brightness = 0;
}

// Clear the canvas by resetting all pixels to black
void clearCanvas()
{
    int i;
    int bytesWidth = canvasWidth * 3 + canvasWidth % 4; // width in bytes (RGB + padding)

    // Set all pixels to 0 (black)
    for (i = 0; i < bytesWidth * canvasHeight; i++) {
        canvas[i] = 0x0;
    }
}

// Draw objects on the canvas by applying transformations and brightness calculations
void objsToCanvas()
{
    int i, j, x, y;
    int xDelta, yDelta, canvasDataLoc;
    int bytesWidth = canvasWidth * 3 + canvasWidth % 4; // byte width of canvas
    int brightness;
    int imgObjNum;

    for (i = 0; i < OBJCNT; i++) {
        // Precompute rotation values for efficiency
        for (j = 0; j < OPCNT; j++) {
            if (objOps[i][j] == END)
                break;

            if ((objOps[i][j] == XROT) || (objOps[i][j] == YROT) || (objOps[i][j] == ZROT)) {
                hUcRotValues[j] = cos(objOpValues[i][j] * (M_PI * 2.0 / 360.0));
                vUcRotValues[j] = sin(objOpValues[i][j] * (M_PI * 2.0 / 360.0));
            }
        }

        // Apply operations to objects and render them
        for (j = 0; j < OPCNT; j++) {
            if (objOps[i][j] == END)
                break;

            if (objOps[i][j] == IMGOBJ) {
                imgObjNum = objOpValues[i][j];

                // Iterate through each pixel of the image and apply object transformations
                for (y = 0; y < IMGBUFF_HEIGHT; y++) {
                    for (x = 0; x < IMGBUFF_WIDTH; x++) {
                        if (imgObjNum < IMGOBJCNT)
                            brightness = imgObj[imgObjNum][IMGBUFF_WIDTH * y + x];

                        if (brightness) {
                            applyObjOps(i, j + 1, x, y, &xDelta, &yDelta, &brightness);

                            canvasDataLoc = xDelta * 3 + bytesWidth * yDelta;

                            // Draw only if the current pixel is brighter
                            if ((xDelta >= 0) && (xDelta < canvasWidth) && (yDelta >= 0)
                                && (yDelta < canvasHeight) && (brightness > canvas[0 + canvasDataLoc])) {
                                canvas[0 + canvasDataLoc] = brightness;
                                canvas[1 + canvasDataLoc] = brightness;
                                canvas[2 + canvasDataLoc] = brightness;
                            }
                        }
                    }
                }
            }
        }
    }
}

// Update object operations and their values
void chgObj(int obj, int opNum, int op, float opValue)
{
    if (obj >= OBJCNT || opNum >= OPCNT)
        return;

    objOps[obj][opNum] = op;
    objOpValues[obj][opNum] = opValue;
}

// Window procedure to handle messages and events
LRESULT CALLBACK WndProc(HWND hwnd, UINT message, WPARAM wParam, LPARAM lParam)
{
    static HDC hdc;
    static PAINTSTRUCT ps;
    static int opInc;
    static int dragLMouse = FALSE;
    static int xMouseLoc, yMouseLoc;
    static int xMouseLocSave, yMouseLocSave;
    static float xCrateRot = 0;
    static float yCrateRot = 0;
    static float zCrateLoc = 0;

    switch (message) {
        case WM_CREATE:
            // Initialize the bitmap header
            pbmi->bmiHeader.biSize = 40;
            pbmi->bmiHeader.biWidth = WIN_WIDTH;
            pbmi->bmiHeader.biHeight = WIN_HEIGHT;
            pbmi->bmiHeader.biPlanes = 1;
            pbmi->bmiHeader.biBitCount = 24;
            pbmi->bmiHeader.biCompression = BI_RGB;
            pbmi->bmiHeader.biSizeImage = WIN_WIDTH * WIN_HEIGHT;
            pbmi->bmiHeader.biXPelsPerMeter = 0;
            pbmi->bmiHeader.biYPelsPerMeter = 0;
            pbmi->bmiHeader.biClrUsed = 0;
            pbmi->bmiHeader.biClrImportant = 0;

            // Set a timer for screen refresh
            SetTimer(hwnd, ID_TIMER, 40, nullptr);

            // Load the crate image
            loadImg(CRATEIMG, (char*)IMGDIR "crate.bmp");

            return 0;

        case WM_SIZE:
            // Update canvas size and center coordinates
            canvasWidth = LOWORD(lParam);
            canvasHeight = HIWORD(lParam);
            xCenter = canvasWidth / 2;
            yCenter = canvasHeight / 2;

            // Update bitmap header for the new size
            pbmi->bmiHeader.biWidth = canvasWidth;
            pbmi->bmiHeader.biHeight = canvasHeight;
            pbmi->bmiHeader.biSizeImage = canvasWidth * canvasHeight;

            return 0;

        case WM_TIMER:
            // Update crate rotations
            xCrateRot += 1;
            yCrateRot += 1;

            // Update rotations based on mouse dragging
            if (dragLMouse) {
                xCrateRot += (yMouseLoc - yMouseLocSave) * -3.0;
                yCrateRot += (xMouseLoc - xMouseLocSave) * -3.0;
            }
            xMouseLocSave = xMouseLoc;
            yMouseLocSave = yMouseLoc;

            // Apply transformations and redraw the crate
            opInc = 0;
            chgObj(CRATE, opInc++, IMGOBJ, CRATEIMG); // draw crate
            chgObj(CRATE, opInc++, ZLOC, 64);
            chgObj(CRATE, opInc++, XROT, xCrateRot);
            chgObj(CRATE, opInc++, YROT, yCrateRot);
            chgObj(CRATE, opInc++, ZLOC, zCrateLoc);

            // Draw crate on the canvas
            clearCanvas();
            objsToCanvas();

            InvalidateRect(hwnd, nullptr, TRUE);
            UpdateWindow(hwnd);

            // Draw the updated canvas
            hdc = GetDC(hwnd);
            SetDIBitsToDevice(hdc, 0, 0, canvasWidth, canvasHeight, 0, 0, 0, canvasHeight, canvas, pbmi, DIB_RGB_COLORS);
            ReleaseDC(hwnd, hdc);

            return 0;

        case WM_LBUTTONDOWN:
            dragLMouse = TRUE;
            return 0;

        case WM_LBUTTONUP:
            dragLMouse = FALSE;
            return 0;

        case WM_MOUSEMOVE:
            xMouseLocSave = xMouseLoc;
            yMouseLocSave = yMouseLoc;
            xMouseLoc = LOWORD(lParam);
            yMouseLoc = canvasHeight - HIWORD(lParam);
            return 0;

        case WM_MOUSEWHEEL:
            if (HIWORD(wParam) == 120)
                zCrateLoc += 10;

            if (HIWORD(wParam) == 65416)
                zCrateLoc -= 10;

            return 0;

        case WM_KEYDOWN:
            // Handle exit on Esc or 'q'
            if (LOWORD(wParam) == 27 || LOWORD(wParam) == 81) {
                KillTimer(hwnd, ID_TIMER);
                PostQuitMessage(0);
            }
            return 0;

        case WM_DESTROY:
            KillTimer(hwnd, ID_TIMER);
            PostQuitMessage(0);
            return 0;

        default:
            return DefWindowProc(hwnd, message, wParam, lParam);
    }
}

// Application entry point
int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, PSTR szCmdLine, int iCmdShow)
{
    HWND hwnd;
    MSG msg;
    WNDCLASS wndclass;

    // Define window class attributes
    wndclass.style = CS_HREDRAW | CS_VREDRAW;
    wndclass.lpfnWndProc = WndProc;
    wndclass.cbClsExtra = 0;
    wndclass.cbWndExtra = 0;
    wndclass.hInstance = hInstance;
    wndclass.hIcon = LoadIcon(nullptr, IDI_APPLICATION);
    wndclass.hCursor = LoadCursor(nullptr, IDC_ARROW);
    wndclass.hbrBackground = 0;
    wndclass.lpszMenuName = nullptr;
    wndclass.lpszClassName = szAppName;

    // Register window class and create the window
    RegisterClass(&wndclass);
    hwnd = CreateWindow(szAppName, szAppName, WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU, 0, 0, WIN_WIDTH, WIN_HEIGHT, nullptr, nullptr, hInstance, nullptr);
    ShowWindow(hwnd, iCmdShow);
    UpdateWindow(hwnd);

    // Main message loop
    while (GetMessage(&msg, nullptr, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
    }

    return msg.wParam;
}
