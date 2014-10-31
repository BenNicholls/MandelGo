package main

import "github.com/veandco/go-sdl2/sdl"

import "unsafe"
import "fmt"
import "math"
import "math/cmplx"

var window *sdl.Window
var event sdl.Event
var renderer *sdl.Renderer
var texture *sdl.Texture
var format *sdl.PixelFormat

var xDim, yDim, bound, line int
var x, y, w, h, xStep, yStep, ratio, minX, minY, magnify float64
var updating, running bool

var colours []uint32
var pixels []uint32
var time uint32

var black uint32

func main() {
	var err error
	running = true
	xDim, yDim = 1200, 600
	x, y = -0.5, 0
	magnify = 1
	bound = 50

	window = sdl.CreateWindow("MandelGo!", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, xDim, yDim, sdl.WINDOW_SHOWN)
	if window == nil {
		fmt.Println("Failed to create window: %s\n", sdl.GetError())
		return
	}

	renderer = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if renderer == nil {
		fmt.Println("Failed to create renderer: %s\n", sdl.GetError())
		return
	}

	format, err = sdl.AllocFormat(uint(window.GetPixelFormat()))
	if err != nil {
		fmt.Println("No pixelformat: %s\n", sdl.GetError())
		return
	}

	texture = sdl.CreateTexture(renderer, window.GetPixelFormat(), sdl.TEXTUREACCESS_STREAMING, xDim, yDim)
	if texture == nil {
		fmt.Println("No texture: %s\n", sdl.GetError())
		return
	}

	pixels = make([]uint32, xDim)

	colours = make([]uint32, 1000)
	black = sdl.MapRGB(format, 0, 0, 0)
	generatePalette()
	setup()

	for running {
		for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyUpEvent:
				switch t.Keysym.Sym {
				case sdl.K_UP:
					bound += 100
					setup()
				case sdl.K_DOWN:
					if bound > 100 {
						bound -= 100
						setup()
					}
				case sdl.K_PAGEUP:
					magnify = magnify * 2
					setup()
				case sdl.K_PAGEDOWN:
					magnify = magnify / 2
					setup()
				}
			case *sdl.MouseButtonEvent:
				if t.Type == sdl.MOUSEBUTTONUP && t.Button == sdl.BUTTON_LEFT {
					x = minX + float64(t.X)*xStep
					y = minY + float64(t.Y)*yStep
					setup()
				}
			}
		}

		if updating {
			calcLine()
			r := &sdl.Rect{int32(0), int32(line), int32(xDim), int32(1)}
			texture.Update(r, unsafe.Pointer(&pixels[0]), xDim*4)
			renderer.Copy(texture, nil, nil)
			renderer.Present()
			
			if line == yDim-1 {
				updating = false
				
				fmt.Println(uint32(xDim*yDim)/(sdl.GetTicks() - time), "kp/s")
			} else {
				line++
			}
		}
	}

	// texture.Destroy()
	// renderer.Destroy()
	// window.Destroy()
}

func setup() {
	ratio = float64(xDim) / float64(yDim)
	w = 3 * ratio / magnify
	h = 3 / magnify
	minX = x - w/2
	minY = y - h/2
	xStep = w / float64(xDim)
	yStep = h / float64(yDim)
	line = 0
	updating = true
	time = sdl.GetTicks()

}

func evalPoint(r, i float64) uint32 {
	p := math.Sqrt(math.Pow(r-0.25, 2) + math.Pow(i, 2))
	if r < p-2*math.Pow(p, 2)+0.25 || math.Pow(r+1, 2)+math.Pow(i, 2) < 1.0/16 {
		return black
	} else {
		n := 0
		z := complex(r, i)
		for c := z; real(z)*real(z) + imag(z)*imag(z) < 9 && n < bound; n++ {
			z = c + complex(real(z)*real(z) - imag(z)*imag(z), 2*real(z)*imag(z))
		}
		if n == bound {
			return black
		} else {
			k := math.Abs((float64(n) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)) * 20)
			return colours[int(k)%len(colours)]
		}
	}
}

func calcLine() {
	currentY := minY + float64(line)*yStep
	currentX := minX
	for i := 0; i < xDim; i++ {
		pixels[i] = evalPoint(currentX, currentY)
		currentX += xStep
	}
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
