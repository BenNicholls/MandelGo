package main

import (
	"errors"
	"fmt"
	"math"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

var window *sdl.Window
var event sdl.Event
var renderer *sdl.Renderer
var texture *sdl.Texture
var format *sdl.PixelFormat

var xDim, yDim int32
var bound, line int
var x, y, w, h, xStep, yStep, ratio, minX, minY, magnify float64
var updating, running bool

var colours []uint32
var pixelBuffers [8][]uint32
var time uint64

var black uint32

var calculators chan lineData
var num_calculators int = 0

var colourCycleSpeed float64 = 10
var colourOffset int = 0

func main() {
	running = true
	xDim, yDim = 1920, 1080
	// x, y = -0.5, 0
	// magnify = 1
	// bound = 50
	x, y = -0.772403, -0.124375
	magnify = 16384
	bound = 550

	calculators = make(chan lineData, 20)

	err := setup_sdl()
	if err != nil {
		fmt.Println(err)
		return
	}

	colours = make([]uint32, 1000)
	black = sdl.MapRGB(format, 0, 0, 0)
	generatePalette()
	resize()

	for running {
		for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				if t.Type == sdl.KEYUP {
					switch t.Keysym.Sym {
					case sdl.K_KP_PLUS:
						bound += 100
						setup()
					case sdl.K_KP_MINUS:
						if bound > 100 {
							bound -= 100
							setup()
						}
					case sdl.K_UP:
						colourOffset = (colourOffset + len(colours)/10) % len(colours)
						setup()
					case sdl.K_DOWN:
						colourOffset = (colourOffset - len(colours)/10)
						if colourOffset < 0 {
							colourOffset += len(colours)
						}
						setup()
					case sdl.K_LEFT:
						if colourCycleSpeed > 1 {
							colourCycleSpeed--
							setup()
						}
					case sdl.K_RIGHT:
						colourCycleSpeed++
						setup()
					case sdl.K_PAGEUP:
						magnify = magnify * 2
						setup()
					case sdl.K_PAGEDOWN:
						magnify = magnify / 2
						setup()
					}
				}
			case *sdl.MouseButtonEvent:
				if t.Type == sdl.MOUSEBUTTONUP && t.Button == sdl.BUTTON_LEFT {
					x = minX + float64(t.X)*xStep
					y = minY + float64(t.Y)*yStep
					setup()
				}
			case *sdl.WindowEvent:
				if t.Event == sdl.WINDOWEVENT_RESIZED || t.Event == sdl.WINDOWEVENT_SIZE_CHANGED {
					fmt.Println("window resized")
					resize()
				}
			}
		}

		if updating {
			select {
			case line_completed := <-calculators:
				num_calculators--
				r := &sdl.Rect{int32(0), int32(line_completed.num), xDim, int32(1)}
				texture.Update(r, unsafe.Pointer(&pixelBuffers[line_completed.bufferIndex][0]), int(xDim)*4)
				if line < int(yDim)-1 {
					num_calculators++
					line_completed.num = line
					go calcLine(line_completed, bound)
					line++
				}
				renderer.Copy(texture, r, r)
				if line%20 == 0 {
					renderer.Present()
				}

				if num_calculators == 0 {
					output_stats()
					renderer.Present()
					updating = false
				}
			default:
				//sdl.Delay(1)
			}
		} else {
			sdl.Delay(50)
		}
	}

	// texture.Destroy()
	// renderer.Destroy()
	// window.Destroy()
}

func setup_sdl() (err error) {
	window, err = sdl.CreateWindow("MandelGo!", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, int32(xDim), int32(yDim), sdl.WINDOW_RESIZABLE|sdl.WINDOW_INPUT_FOCUS)
	if window == nil {
		return errors.New("Failed to create window: " + sdl.GetError().Error())
	}

	renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return errors.New("Failed to create renderer: " + sdl.GetError().Error())
	}

	f, err := window.GetPixelFormat()
	format, err = sdl.AllocFormat(uint(f))

	texture, err = renderer.CreateTexture(f, sdl.TEXTUREACCESS_STREAMING, int32(xDim), int32(yDim))
	if err != nil {
		return errors.New("No texture: " + sdl.GetError().Error())
	}

	return
}

func resize() (err error) {
	xDim, yDim = window.GetSize()
	for buffer_index := range pixelBuffers {
		pixelBuffers[buffer_index] = make([]uint32, xDim)
	}

	f, err := window.GetPixelFormat()
	if texture != nil {
		texture.Destroy()
	}
	texture, err = renderer.CreateTexture(f, sdl.TEXTUREACCESS_STREAMING, int32(xDim), int32(yDim))
	if err != nil {
		return errors.New("No texture: " + sdl.GetError().Error())
	}

	setup()

	return
}

func setup() {
	for num_calculators > 0 {
		<-calculators
		num_calculators--
	}

	ratio = float64(xDim) / float64(yDim)
	w = 3 * ratio / magnify
	h = 3 / magnify
	minX = x - w/2
	minY = y - h/2
	xStep = w / float64(xDim)
	yStep = h / float64(yDim)
	line = 0
	time = sdl.GetTicks64()
	begin_calculating()
}

func begin_calculating() {
	updating = true
	for i := range pixelBuffers {
		num_calculators++
		go calcLine(lineData{line, i}, bound)
		line++
	}
}

func evalPoint(r, i float64, max_iterations int) uint32 {
	q := (r-0.25)*(r-0.25) + i*i
	if 4*q*(q+(r-0.25)) < i*i || (r+1)*(r+1)+i*i < 1.0/16 {
		return black
	}

	n := 0
	r2, i2 := r*r, i*i
	for c_r, c_i := r, i; r2+i2 <= 4 && n < max_iterations; n++ {
		r, i = r2-i2+c_r, (r+r)*i+c_i
		r2, i2 = r*r, i*i
	}

	if n >= max_iterations {
		return black
	} else {
		k := math.Abs((float64(n) - math.Log(math.Log(math.Hypot(r, i)))/math.Log(2)) * colourCycleSpeed)
		return colours[(int(k)+colourOffset)%len(colours)]
	}
}

type lineData struct {
	num         int
	bufferIndex int
}

func calcLine(current_line lineData, max_iterations int) {
	currentY := minY + float64(current_line.num)*yStep
	currentX := minX
	for i := int32(0); i < xDim; i++ {
		pixelBuffers[current_line.bufferIndex][i] = evalPoint(currentX, currentY, max_iterations)
		currentX += xStep
	}

	calculators <- current_line
}

func generatePalette() {
	r1, g1, b1 := 25, 25, 122
	r2, g2, b2 := 205, 133, 0
	r3, g3, b3 := 255, 255, 255
	r4, g4, b4 := 180, 205, 205

	k := len(colours) / 4

	for i := 0; i < k; i++ {
		colours[i] = sdl.MapRGB(format, interpC(r1, r2, i, k), interpC(g1, g2, i, k), interpC(b1, b2, i, k))
		colours[i+k] = sdl.MapRGB(format, interpC(r2, r3, i, k), interpC(g2, g3, i, k), interpC(b2, b3, i, k))
		colours[i+k*2] = sdl.MapRGB(format, interpC(r3, r4, i, k), interpC(g3, g4, i, k), interpC(b3, b4, i, k))
		colours[i+k*3] = sdl.MapRGB(format, interpC(r4, r1, i, k), interpC(g4, g1, i, k), interpC(b4, b1, i, k))
	}
}

func interpC(c1, c2, i, k int) uint8 {
	return uint8(c1 + (i * (c2 - c1) / k))
}

func output_stats() {
	render_time := sdl.GetTicks64() - time
	fmt.Printf("Size: %dx%d (%d total pixels)\n", xDim, yDim, xDim*yDim)
	fmt.Printf("Position: (%f, %f)\n", x, y)
	fmt.Printf("Magnification: %.2f, Max Iterations: %d\n", magnify, bound)
	fmt.Printf("Output complete in %dms (%d kp/s)\n", render_time, uint64(xDim)*uint64(yDim)/render_time)
}
