package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/xyproto/vt100"
	lua "github.com/yuin/gopher-lua"
)

var (
	highScoreFile       = filepath.Join(userCacheDir, "o", "highscore.txt")
	gameBackgroundColor = vt100.DefaultBackground
)

// LuaGame struct to hold the Lua state and relevant functions
type LuaGame struct {
	L          *lua.LState
	UpdateFunc lua.LValue
	DrawFunc   lua.LValue
	InitFunc   lua.LValue
	ResizeFunc lua.LValue
}

// NewLuaGame creates a new LuaGame instance
func NewLuaGame(script string) (*LuaGame, error) {
	L := lua.NewState()
	if err := L.DoFile(script); err != nil {
		return nil, err
	}

	game := &LuaGame{
		L:          L,
		UpdateFunc: L.GetGlobal("updateGame"),
		DrawFunc:   L.GetGlobal("drawGame"),
		InitFunc:   L.GetGlobal("initGame"),
		ResizeFunc: L.GetGlobal("resizeGame"),
	}

	// Initialize the game
	width := int(L.GetGlobal("canvas_width").(lua.LNumber))
	height := int(L.GetGlobal("canvas_height").(lua.LNumber))
	if err := L.CallByParam(lua.P{
		Fn:      game.InitFunc,
		NRet:    0,
		Protect: true,
	}, lua.LNumber(width), lua.LNumber(height)); err != nil {
		return nil, err
	}

	return game, nil
}

// Update calls the update function in the Lua script
func (game *LuaGame) Update(key int) error {
	return game.L.CallByParam(lua.P{
		Fn:      game.UpdateFunc,
		NRet:    0,
		Protect: true,
	}, lua.LNumber(key))
}

// Draw calls the draw function in the Lua script
func (game *LuaGame) Draw() error {
	return game.L.CallByParam(lua.P{
		Fn:      game.DrawFunc,
		NRet:    0,
		Protect: true,
	})
}

// Resize calls the resize function in the Lua script
func (game *LuaGame) Resize(width, height int) error {
	return game.L.CallByParam(lua.P{
		Fn:      game.ResizeFunc,
		NRet:    0,
		Protect: true,
	}, lua.LNumber(width), lua.LNumber(height))
}

// RunGame runs the game loop, returns true if ctrl-q was pressed
func RunGame(script string) (bool, error) {
	game, err := NewLuaGame(script)
	if err != nil {
		return false, err
	}

	// Try loading the highscore from the file, but ignore any errors
	_, _ = loadHighScore()

	c := vt100.NewCanvas()
	c.FillBackground(gameBackgroundColor)

	tty, err := vt100.NewTTY()
	if err != nil {
		return false, err
	}
	defer tty.Close()

	tty.SetTimeout(2 * time.Millisecond)

	var (
		sigChan = make(chan os.Signal, 1)
		running = true
	)

	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			resizeMut.Lock()
			// Create a new canvas, with the new size
			nc := c.Resized()
			if nc != nil {
				c.Clear()
				vt100.Clear()
				c.Draw()
				c = nc
			}
			// Inform the Lua script about the resize
			if err := game.Resize(int(c.W()), int(c.H())); err != nil {
				fmt.Fprintln(os.Stderr, "resize error: "+err.Error())
			}
			resizeMut.Unlock()
		}
	}()

	vt100.Init()
	vt100.EchoOff()
	defer vt100.Close()

	// The loop time that is aimed for
	var (
		loopDuration = time.Millisecond * 10
		start        = time.Now()
		key          int
	)

	for running {
		c.Clear()

		resizeMut.RLock()
		if err := game.Draw(); err != nil {
			fmt.Fprintln(os.Stderr, "draw error: "+err.Error())
		}
		resizeMut.RUnlock()

		// Update the canvas
		c.Draw()

		// Wait a bit
		end := time.Now()
		passed := end.Sub(start)
		if passed < loopDuration {
			remaining := loopDuration - passed
			time.Sleep(remaining)
		}
		start = time.Now()

		// Handle events
		key = tty.Key()
		switch key {
		case 27: // ESC
			running = false
		case 17: // ctrl-q
			return true, nil
		}

		if err := game.Update(key); err != nil {
			fmt.Fprintln(os.Stderr, "update error: "+err.Error())
		}
	}

	return false, nil
}

// saveHighScore will save the given high score to a file
func saveHighScore(highScore uint) error {
	if noWriteToCache {
		return nil
	}
	// First create the folders, if needed
	folderPath := filepath.Dir(highScoreFile)
	os.MkdirAll(folderPath, os.ModePerm)
	// Prepare the file
	f, err := os.OpenFile(highScoreFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	// Write the contents, ignore the number of written bytes
	_, err = f.WriteString(fmt.Sprintf("%d\n", highScore))
	return err
}

// loadHighScore will load the current high score from the highScoreFile
func loadHighScore() (uint, error) {
	data, err := os.ReadFile(highScoreFile)
	if err != nil {
		return 0, err
	}
	highScoreString := string(bytes.TrimSpace(data))
	highScore, err := strconv.ParseUint(highScoreString, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(highScore), nil
}

// Game runs the game using the "feedgame.lua" script
func Game() (bool, error) {
	return RunGame("feedgame.lua")
}
