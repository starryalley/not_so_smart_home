package colors

import "math"

// Color represents a Color used for LED
type Color struct {
	R, G, B uint8
}

var definedColors = [...]Color{
	{255, 0, 255}, //purple
	{0, 0, 255},   //blue
	{0, 255, 255}, //cyan
	{0, 255, 0},   //green
	{255, 255, 0}, //yellow
	{255, 0, 0},   //red
}

const numColor = len(definedColors)

// max/min temperature in the room
const maxTemp = 32
const minTemp = 8

// max brightness 100
const ledBrightness = 100

func interpolateV(x, y uint8, dx float64) uint8 {
	return uint8((1-dx)*float64(x) + dx*float64(y))
}

func interpolate(c1, c2 Color, dx float64) Color {
	return Color{
		interpolateV(c1.R, c2.R, dx),
		interpolateV(c1.G, c2.G, dx),
		interpolateV(c1.B, c2.B, dx),
	}
}

func (c Color) mul(s float32) Color {
	return Color{
		uint8(float32(c.R) * s),
		uint8(float32(c.G) * s),
		uint8(float32(c.B) * s),
	}
}

// TemperatureToColor gets a temperature and returns a Color which represents this air temperature
// ref: https://github.com/lilspikey/arduino_sketches/blob/master/nightlight/nightlight.h
func TemperatureToColor(t float32) Color {
	if t < minTemp {
		return definedColors[0]
	} else if t > maxTemp {
		return definedColors[numColor-1]
	}
	col := float64(t-minTemp) / (maxTemp - minTemp) * float64(numColor-1)
	colLow := int(math.Floor(col))
	colHigh := int(math.Ceil(col))
	dx := float64(colHigh) - col
	return interpolate(definedColors[colHigh], definedColors[colLow], dx).mul(float32(ledBrightness) / 100)
}

// AQIToColor gets a AQI value and returns a Color which represents this AQI
func AQIToColor(idx float64) Color {
	if idx <= 50 {
		// green
		return Color{0, 255, 0}
	} else if idx <= 100 {
		// yellow
		return Color{255, 255, 0}
	} else if idx <= 150 {
		// orange
		return Color{255, 127, 0}
	} else if idx <= 200 {
		// red
		return Color{255, 0, 0}
	} else if idx <= 300 {
		// purple
		return Color{255, 0, 255}
	} else {
		// brown #7E0023
		return Color{126, 0, 35}
	}
}
