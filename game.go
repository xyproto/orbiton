package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/xyproto/vt100"
)

// Every editor should include a small game, right?

const (
	pelletRune      = 'a'
	bobRuneLarge    = 'O'
	bobRuneSmall    = 'o'
	evilGobblerRune = '€'
	bubbleRune      = '°'
	gobblerRune     = '@'
	gobblerDeadRune = 'T'
	bobWonRune      = 'Y'
	bobLostRune     = 'n'
)

var (
	highScoreFile = filepath.Join(userCacheDir, "o/highscore.txt")

	bobColor         = vt100.LightCyan
	bobWonColor      = vt100.LightCyan
	bobLostColor     = vt100.LightCyan
	evilGobblerColor = vt100.LightRed
	gobblerColor     = vt100.Green
	gobblerDeadColor = vt100.DarkGray
	bubbleColor      = vt100.LightMagenta
	pelletColor1     = vt100.LightBlue
	pelletColor2     = vt100.White
	statusTextColor  = vt100.Yellow
)

type Bob struct {
	x, y       int                  // current position
	oldx, oldy int                  // previous position
	state      rune                 // looks
	color      vt100.AttributeColor // foreground color
}

func NewBob(c *vt100.Canvas) *Bob {
	//var startingWidth = int(c.W())
	return &Bob{
		x:     5,
		y:     10,
		oldx:  5,
		oldy:  10,
		state: bobRuneSmall,
		color: bobColor,
	}
}

func (b *Bob) ToggleState() {
	const up = bobRuneLarge
	const down = bobRuneSmall
	if b.state == up {
		b.state = down
	} else {
		b.state = up
	}
}

func (b *Bob) Draw(c *vt100.Canvas) {
	c.PlotColor(uint(b.x), uint(b.y), b.color, b.state)
}

func (b *Bob) Right(c *vt100.Canvas) bool {
	oldx := b.x
	b.x++
	if b.x >= int(c.W()) {
		b.x--
		return false
	}
	b.oldx = oldx
	b.oldy = b.y
	return true
}

func (b *Bob) Left(c *vt100.Canvas) bool {
	oldx := b.x
	if b.x-1 < 0 {
		return false
	}
	b.x--
	b.oldx = oldx
	b.oldy = b.y
	return true
}

func (b *Bob) Up(c *vt100.Canvas) bool {
	oldy := b.y
	if b.y-1 < 0 {
		return false
	}
	b.y--
	b.oldx = b.x
	b.oldy = oldy
	return true
}

func (b *Bob) Down(c *vt100.Canvas) bool {
	oldy := b.y
	b.y++
	if b.y >= int(c.H()) {
		b.y--
		return false
	}
	b.oldx = b.x
	b.oldy = oldy
	return true
}

// Terminal was resized
func (b *Bob) Resize() {
	b.color = vt100.LightMagenta
}

type Pellet struct {
	x, y       int                  // current position
	oldx, oldy int                  // previous position
	vx, vy     int                  // velocity
	state      rune                 // looks
	color      vt100.AttributeColor // foreground color
	stopped    bool                 // is the movement stopped?
	removed    bool                 // to be removed
}

func NewPellet(x, y, vx, vy int) *Pellet {
	return &Pellet{
		x:       x,
		y:       y,
		oldx:    x,
		oldy:    y,
		vx:      vx,
		vy:      vy,
		state:   '·',
		color:   pelletColor1,
		stopped: false,
		removed: false,
	}
}

func (b *Pellet) ToggleColor() {
	c1 := pelletColor1
	c2 := pelletColor2
	if b.color.Equal(c1) {
		b.color = c2
	} else {
		b.color = c1
	}
}

func (b *Pellet) ToggleState() {
	const up = '×'
	const down = '-'
	if b.state == up {
		b.state = down
	} else {
		b.state = up
	}
}

func (b *Pellet) Draw(c *vt100.Canvas) {
	c.PlotColor(uint(b.x), uint(b.y), b.color, b.state)
}

// Next moves the object to the next position, and returns true if it moved
func (b *Pellet) Next(c *vt100.Canvas, e *EvilGobbler) bool {
	if b.stopped {
		b.ToggleColor()
		return false
	}
	if b.x-b.vx < 0 {
		b.ToggleColor()
		return false
	}
	if b.y-b.vy < 0 {
		b.ToggleColor()
		return false
	}
	b.oldx = b.x
	b.oldy = b.y
	b.x += b.vx
	b.y += b.vy

	if b.x == e.x && b.y == e.y {
		e.shot = true
	}

	if b.HitSomething(c) {
		b.x = b.oldx
		b.y = b.oldy
		return false
	}
	if b.x >= int(c.W()) {
		b.x -= b.vx
		return false
	} else if b.x < 0 {
		b.x -= b.vx
		return false
	}
	if b.y >= int(c.H()) {
		b.y -= b.vy
		return false
	} else if b.y < 0 {
		b.y -= b.vy
		return false
	}
	return true
}

func (b *Pellet) Stop() {
	b.vx = 0
	b.vy = 0
	b.stopped = true
}

func (b *Pellet) HitSomething(c *vt100.Canvas) bool {
	r, err := c.At(uint(b.x), uint(b.y))
	if err != nil {
		return false
	}
	if r != rune(0) && r != ' ' {
		// Hit something. Check the next-next position too
		r2, err := c.At(uint(b.x+b.vx), uint(b.y+b.vy))
		if err != nil {
			return true
		}
		if r2 != rune(0) && r2 != ' ' {
			b.Stop()
		}
		return true
	}
	return false
}

// Terminal was resized
func (b *Pellet) Resize() {
	b.stopped = false
}

type Bubble struct {
	x, y       int                  // current position
	oldx, oldy int                  // previous position
	state      rune                 // looks
	color      vt100.AttributeColor // foreground color
}

func NewEnemies(n int) []*Bubble {
	enemies := make([]*Bubble, n)
	for i := range enemies {
		enemies[i] = NewBubble()
	}
	return enemies
}

func NewBubble() *Bubble {
	return &Bubble{
		x:     10,
		y:     10,
		oldx:  10,
		oldy:  10,
		state: bubbleRune,
		color: bubbleColor,
	}
}

func (b *Bubble) Draw(c *vt100.Canvas) {
	c.PlotColor(uint(b.x), uint(b.y), b.color, b.state)
}

func (b *Bubble) Right(c *vt100.Canvas) bool {
	oldx := b.x
	b.x++
	if b.x >= int(c.W()) {
		b.x--
		return false
	}
	b.oldx = oldx
	b.oldy = b.y
	return true
}

func (b *Bubble) Left(c *vt100.Canvas) bool {
	oldx := b.x
	if b.x-1 < 0 {
		return false
	}
	b.x--
	b.oldx = oldx
	b.oldy = b.y
	return true
}

func (b *Bubble) Up(c *vt100.Canvas) bool {
	oldy := b.y
	if b.y-1 < 0 {
		return false
	}
	b.y--
	b.oldx = b.x
	b.oldy = oldy
	return true
}

func (b *Bubble) Down(c *vt100.Canvas) bool {
	oldy := b.y
	b.y++
	if b.y >= int(c.H()) {
		b.y--
		return false
	}
	b.oldx = b.x
	b.oldy = oldy
	return true
}

// Terminal was resized
func (b *Bubble) Resize() {
	b.color = vt100.LightMagenta
}

// Next moves the object to the next position, and returns true if it moved
func (b *Bubble) Next(c *vt100.Canvas, bob *Bob) bool {
	b.oldx = b.x
	b.oldy = b.y

	// Now try to move the bubble intelligently, given the position of bob

	d := distance(bob.x, b.x, bob.y, b.y)
	if d > 10 {
		if b.x < bob.x {
			b.x++
		} else if b.x > bob.x {
			b.x--
		}
		if b.y < bob.y {
			b.y++
		} else if b.y > bob.y {
			b.y--
		}
	} else {
		for {
			dx := b.x - b.oldx
			dy := b.y - b.oldy
			b.x += int(math.Round(float64(dx*3+rand.Intn(5)-2) / float64(4))) // -2, -1, 0, 1, 2
			b.y += int(math.Round(float64(dy*3+rand.Intn(5)-2) / float64(4)))
			if b.x != b.oldx {
				break
			}
			if b.y != b.oldy {
				break
			}
		}
	}

	if b.HitSomething(c) {
		b.x = b.oldx
		b.y = b.oldy
		return false
	}

	if b.x >= int(c.W()) {
		b.x = b.oldx
	} else if b.x <= 0 {
		b.x = b.oldx
	}
	if b.y >= int(c.H()) {
		b.y = b.oldy
	} else if b.y <= 0 {
		b.y = b.oldy
	}

	return b.x != b.oldx || b.y != b.oldy
}

func (b *Bubble) HitSomething(c *vt100.Canvas) bool {
	r, err := c.At(uint(b.x), uint(b.y))
	if err != nil {
		return false
	}
	// Hit something?
	return r != rune(0) && r != ' '
}

type EvilGobbler struct {
	x, y            int                  // current position
	oldx, oldy      int                  // previous position
	state           rune                 // looks
	color           vt100.AttributeColor // foreground color
	hunting         *Gobbler             // current gobbler to hunt
	huntingDistance float64              // how far to closest gobbler
	counter         uint
	shot            bool
}

func NewEvilGobbler(c *vt100.Canvas) *EvilGobbler {
	var startingWidth = int(c.W())
	return &EvilGobbler{
		x:       startingWidth/2 + 5,
		y:       01,
		oldx:    startingWidth/2 + 5,
		oldy:    10,
		state:   evilGobblerRune,
		color:   evilGobblerColor,
		counter: 0,
		shot:    false,
	}
}

func (e *EvilGobbler) Draw(c *vt100.Canvas) {
	c.PlotColor(uint(e.x), uint(e.y), e.color, e.state)
}

func (e *EvilGobbler) Next(c *vt100.Canvas, gobblers []*Gobbler, bob *Bob) bool {
	e.oldx = e.x
	e.oldy = e.y

	var hunting *Gobbler
	var huntingDistance = 99999.9

	for _, b := range gobblers {
		if d := distance(b.x, e.x, b.y, e.y); !b.dead && d <= huntingDistance {
			hunting = b
			huntingDistance = d
		}
	}

	if hunting == nil {

		e.x += rand.Intn(3) - 1
		e.y += rand.Intn(3) - 1

	} else {

		xspeed := 1
		yspeed := 1

		if e.x < hunting.x {
			e.x += xspeed
		} else if e.x > hunting.x {
			e.x -= xspeed
		}
		if e.y < hunting.y {
			e.y += yspeed
		} else if e.y > hunting.y {
			e.y -= yspeed
		}

		if distance(bob.x, e.x, bob.y, e.y) < 3 {
			e.x = e.oldx + (rand.Intn(5) - 2)
			e.y = e.oldy + (rand.Intn(5) - 2)
		}

		if !hunting.dead && huntingDistance < 1.8 || (hunting.x == e.x && hunting.y == e.y) {
			hunting.dead = true
			e.counter++
			hunting = nil
		}
	}

	if e.x > int(c.W()) {
		e.x = e.oldx
	} else if e.x < 0 {
		e.x = e.oldx
	}

	if e.y > int(c.H()) {
		e.y = e.oldy
	} else if e.y < 0 {
		e.y = e.oldy
	}

	return (e.x != e.oldx || e.y != e.oldy)
}

// Terminal was resized
func (e *EvilGobbler) Resize() {
	e.color = vt100.White
}

type Gobbler struct {
	x, y            int                  // current position
	oldx, oldy      int                  // previous position
	state           rune                 // looks
	color           vt100.AttributeColor // foreground color
	hunting         *Pellet              // current pellet to hunt
	huntingDistance float64              // how far to closest pellet
	counter         uint
	dead            bool
}

func NewGobbler(c *vt100.Canvas) *Gobbler {
	var startingWidth = int(c.W())
	return &Gobbler{
		x:               startingWidth / 2,
		y:               10,
		oldx:            startingWidth / 2,
		oldy:            10,
		state:           gobblerRune,
		color:           gobblerColor,
		hunting:         nil,
		huntingDistance: 0,
		counter:         0,
		dead:            false,
	}
}

func NewGobblers(c *vt100.Canvas, n int) []*Gobbler {
	gobblers := make([]*Gobbler, n)
	for i := range gobblers {
		gobblers[i] = NewGobbler(c)
	}
	return gobblers
}

func (g *Gobbler) Draw(c *vt100.Canvas) {
	c.PlotColor(uint(g.x), uint(g.y), g.color, g.state)
}

func (g *Gobbler) Next(c *vt100.Canvas, pellets []*Pellet, bob *Bob) bool {
	if g.dead {
		g.state = gobblerDeadRune
		g.color = gobblerDeadColor
		return true
	}

	g.oldx = g.x
	g.oldy = g.y

	// Move to the nearest pellet and eat it
	if len(pellets) == 0 {

		g.x += rand.Intn(5) - 2
		g.y += rand.Intn(5) - 2

	} else {

		if g.hunting == nil || g.hunting.removed == true {
			var minDistance = 99999.9
			var closestPellet *Pellet
			for _, b := range pellets {
				if d := distance(b.x, g.x, b.y, g.y); !b.removed && d <= minDistance {
					closestPellet = b
					minDistance = d
				}
			}
			if closestPellet != nil {
				g.hunting = closestPellet
				g.huntingDistance = minDistance
			}
		} else {
			g.huntingDistance = distance(g.hunting.x, g.x, g.hunting.y, g.y)
		}

		if g.hunting == nil {

			g.x += rand.Intn(5) - 2
			g.y += rand.Intn(5) - 2

		} else {

			xspeed := 1
			yspeed := 1

			if abs(g.hunting.x-g.x) >= abs(g.hunting.y-g.y) {
				// Longer away along x than along y
				if g.huntingDistance > 20 {
					xspeed = 3
					yspeed = 2
				} else if g.huntingDistance > 10 {
					xspeed = 2 + rand.Intn(2)
					yspeed = 2
				}
			} else {
				// Longer away along x than along y
				if g.huntingDistance > 20 {
					xspeed = 2
					yspeed = 3
				} else if g.huntingDistance > 10 {
					xspeed = 2
					yspeed = 2 + rand.Intn(2)
				}
			}

			if g.x < g.hunting.x {
				g.x += xspeed
			} else if g.x > g.hunting.x {
				g.x -= xspeed
			}
			if g.y < g.hunting.y {
				g.y += yspeed
			} else if g.y > g.hunting.y {
				g.y -= yspeed
			}

			if distance(bob.x, g.x, bob.y, g.y) < 15 {
				g.x = g.oldx + (rand.Intn(3) - 1)
				g.y = g.oldy + (rand.Intn(3) - 1)
			}

			if !g.hunting.removed && g.huntingDistance < 2 || (g.hunting.x == g.x && g.hunting.y == g.y) {
				g.hunting.removed = true
				g.counter++
				g.hunting = nil
			}
		}
	}

	if g.x > int(c.W()) {
		g.x = g.oldx
	} else if g.x < 0 {
		g.x = g.oldx
	}

	if g.y > int(c.H()) {
		g.y = g.oldy
	} else if g.y < 0 {
		g.y = g.oldy
	}

	return (g.x != g.oldx || g.y != g.oldy)
}

// Terminal was resized
func (g *Gobbler) Resize() {
	g.color = vt100.White
}

func abs(a int) int {
	if a >= 0 {
		return a
	}
	return -a
}

func distance(x1, x2, y1, y2 int) float64 {
	return math.Sqrt((float64(x1)*float64(x1) - float64(x2)*float64(x2)) + (float64(y1)*float64(y1) - float64(y2)*float64(y2)))
}

func saveHighScore(highScore uint) error {
	// First create the folders, if needed
	folderPath := filepath.Dir(highScoreFile)
	os.MkdirAll(folderPath, os.ModePerm)
	// Prepare the file
	f, err := os.OpenFile(highScoreFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer f.Close()
	// Write the contents, ignore the number of written bytes
	_, err = f.WriteString(fmt.Sprintf("%d\n", highScore))
	return err
}

func loadHighScore() (uint, error) {
	data, err := ioutil.ReadFile(highScoreFile)
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

func Game() error {
	rand.Seed(time.Now().UnixNano())

	// Try loading the highscore from the file, but ignore any errors
	highScore, _ := loadHighScore()

	c := vt100.NewCanvas()
	//c.FillBackground(vt100.Blue)

	tty, err := vt100.NewTTY()
	if err != nil {
		panic(err)
	}
	defer tty.Close()

	// Mutex used when the terminal is resized
	resizeMut := &sync.RWMutex{}

	var (
		bob         = NewBob(c)
		sigChan     = make(chan os.Signal, 1)
		evilGobbler = NewEvilGobbler(c)
		gobblers    = NewGobblers(c, 10)
		pellets     = make([]*Pellet, 0)
		enemies     = NewEnemies(7)
		score       = uint(0)
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

			// Inform all elements that the terminal was resized
			// TODO: Use a slice of interfaces that can contain all elements
			for _, pellet := range pellets {
				pellet.Resize()
			}
			for _, bubble := range enemies {
				bubble.Resize()
			}
			for _, gobbler := range gobblers {
				gobbler.Resize()
			}
			bob.Resize()
			evilGobbler.Resize()
			resizeMut.Unlock()
		}
	}()

	vt100.Init()
	defer vt100.Close()

	// The loop time that is aimed for
	loopDuration := time.Millisecond * 10
	start := time.Now()

	running := true
	paused := false
	var statusText string

	// Don't output keypress terminal codes on the screen
	tty.NoBlock()

	var key int

	for running {

		// Draw elements in their new positions
		c.Clear()
		//c.Draw()

		resizeMut.RLock()
		for _, pellet := range pellets {
			pellet.Draw(c)
		}
		for _, bubble := range enemies {
			bubble.Draw(c)
		}
		evilGobbler.Draw(c)
		for _, gobbler := range gobblers {
			gobbler.Draw(c)
		}
		bob.Draw(c)
		c.Write(5, 1, statusTextColor, vt100.BackgroundDefault, statusText)
		resizeMut.RUnlock()

		//vt100.Clear()

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

		// Has the player moved?
		moved := false

		// Handle events
		key = tty.Key()
		switch key {
		case 253, 119: // Up or w
			resizeMut.Lock()
			moved = bob.Up(c)
			resizeMut.Unlock()
		case 255, 115: // Down or s
			resizeMut.Lock()
			moved = bob.Down(c)
			resizeMut.Unlock()
		case 254, 100: // Right or d
			resizeMut.Lock()
			moved = bob.Right(c)
			resizeMut.Unlock()
		case 252, 97: // Left or a
			resizeMut.Lock()
			moved = bob.Left(c)
			resizeMut.Unlock()
		case 27: // ESC
			running = false
		case 32: // Space
			// Check if the place to the right is available
			r, err := c.At(uint(bob.x+1), uint(bob.y))
			if err != nil {
				// No free place to the right
				break
			}
			if r == rune(0) || r == ' ' {
				// Fire a new pellet
				pellets = append(pellets, NewPellet(bob.x, bob.y, bob.x-bob.oldx, bob.y-bob.oldy))
			}
		}

		if !paused {
			// Change state
			resizeMut.Lock()
			for _, pellet := range pellets {
				pellet.Next(c, evilGobbler)
			}
			for _, bubble := range enemies {
				bubble.Next(c, bob)
			}
			for _, gobbler := range gobblers {
				gobbler.Next(c, pellets, bob)
			}
			evilGobbler.Next(c, gobblers, bob)
			if moved {
				bob.ToggleState()
			}
			resizeMut.Unlock()
		}

		// Erase all previous positions not occupied by current items
		c.Plot(uint(bob.oldx), uint(bob.oldy), ' ')
		c.Plot(uint(evilGobbler.oldx), uint(evilGobbler.oldy), ' ')
		for _, pellet := range pellets {
			c.Plot(uint(pellet.oldx), uint(pellet.oldy), ' ')
		}
		for _, bubble := range enemies {
			c.Plot(uint(bubble.oldx), uint(bubble.oldy), ' ')
		}
		for _, gobbler := range gobblers {
			c.Plot(uint(gobbler.oldx), uint(gobbler.oldy), ' ')
		}

		if !paused {

			// Clean up removed pellets
			filteredPellets := make([]*Pellet, 0, len(pellets))
			for _, pellet := range pellets {
				if !pellet.removed {
					filteredPellets = append(filteredPellets, pellet)
				} else {
					c.Plot(uint(pellet.x), uint(pellet.y), ' ')
				}
			}
			pellets = filteredPellets

			gobblersAlive := false
			for _, gobbler := range gobblers {
				score += gobbler.counter
				(*gobbler).counter = 0
				if !gobbler.dead {
					gobblersAlive = true
				}
			}
			// evilGobbler.counter
			if gobblersAlive {
				statusText = fmt.Sprintf("Score: %d", score)
			} else if evilGobbler.shot {
				paused = true
				statusText = "You won!"

				// The player can still move around bob
				bob.state = bobWonRune
				bob.color = bobWonColor

				if score > highScore {
					statusText = fmt.Sprintf("You won! New highscore: %d", score)
					saveHighScore(score)
				} else if score > 0 {
					statusText = fmt.Sprintf("You won! Score: %d", score)
				}
			} else {
				paused = true
				statusText = "Game over"

				// The player can still move around bob
				bob.state = bobLostRune
				bob.color = bobLostColor

				if score > highScore {
					statusText = fmt.Sprintf("Game over! New highscore: %d", score)
					saveHighScore(score)
				} else if score > 0 {
					statusText = fmt.Sprintf("Game over! Score: %d", score)
				}
			}
		}
	}
	return nil
}
