package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/d2r2/go-dht"
	logger "github.com/d2r2/go-logger"
	"github.com/gofrs/flock"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)

// ========== Settings =========
// update interval in seconds
const UpdateInterval = 60 * 5

// GPIO number for DHT temperature sensor
const GPIOTemp = 4

// Pin names for LED R,G,B pins
const PinR = "11"
const PinG = "13"
const PinB = "15"

// max/min temperature in the room
const MaxTemp = 25
const MinTemp = 12

// max brightness 100
const LEDBrightness = 100

// =============================

type color struct {
	R, G, B uint8
}

var Colors = [...]color{
	{255, 0, 255}, //purple
	{0, 0, 255},   //blue
	{0, 255, 255}, //cyan
	{0, 255, 0},   //green
	{255, 255, 0}, //yellow
	{255, 0, 0},   //red
}

const NumColor = len(Colors)

func interpolateV(x, y uint8, dx float64) uint8 {
	return uint8((1-dx)*float64(x) + dx*float64(y))
}

func interpolate(c1, c2 color, dx float64) color {
	return color{
		interpolateV(c1.R, c2.R, dx),
		interpolateV(c1.G, c2.G, dx),
		interpolateV(c1.B, c2.B, dx),
	}
}

func (c color) mul(s float32) color {
	return color{
		uint8(float32(c.R) * s),
		uint8(float32(c.G) * s),
		uint8(float32(c.B) * s),
	}
}

// ref: https://github.com/lilspikey/arduino_sketches/blob/master/nightlight/nightlight.h
func temperatureToColor(t float32) color {
	if t < MinTemp {
		return Colors[0]
	} else if t > MaxTemp {
		return Colors[NumColor-1]
	}
	col := float64(t-MinTemp) / (MaxTemp - MinTemp) * float64(NumColor-1)
	colLow := int(math.Floor(col))
	colHigh := int(math.Ceil(col))
	dx := float64(colHigh) - col
	return interpolate(Colors[colHigh], Colors[colLow], dx).mul(float32(LEDBrightness) / 100)
}

func getTempHum() (float32, float32, error) {
	logger.ChangePackageLogLevel("dht", logger.ErrorLevel)
	temperature, humidity, _, err :=
		dht.ReadDHTxxWithRetry(dht.DHT22, GPIOTemp, false, 10)
	return temperature, humidity, err
}

func main() {
	// for concurrent access to light sensor
	fileLock := flock.New("/var/lock/tsl2561.lock")

	r := raspi.NewAdaptor()
	led := gpio.NewRgbLedDriver(r, PinR, PinG, PinB)
	lux := i2c.NewTSL2561Driver(r, i2c.WithBus(0), i2c.WithAddress(0x39), i2c.WithTSL2561Gain1X)
	// for updating to google sheet
	sheet, err := InitGoogleSheet("client_secret.json")
	if err != nil {
		panic(err)
	}
	work := func() {
		gobot.Every(UpdateInterval*time.Second, func() {
			temp, hum, err := getTempHum()
			if err != nil {
				log.Printf("read temperature failed:%v\n", err)
				return
			}
			c := temperatureToColor(temp)
			var broadband, ir uint16
			var now time.Time
			for {
				locked, err := fileLock.TryLock()
				if err != nil {
					log.Printf("Unable to lock:%v\n", err)
					return
				}
				if locked {
					defer fileLock.Unlock()
					// turn off LED to get accurate lux
					led.Off()
					time.Sleep(500 * time.Millisecond)
					now = time.Now()
					broadband, ir, err = lux.GetLuminocity()
					if err != nil {
						log.Printf("read luminocity failed:%v\n", err)
						return
					}
					break
				}
			}
			light := lux.CalculateLux(broadband, ir)

			// now turn on LED
			led.SetRGB(c.R, c.G, c.B)
			log.Printf("Temperature:%v*C, Humidity:%v%% (LED: %v,%v,%v) BB:%v, IR:%v, Lux:%v\n",
				temp, hum, c.R, c.G, c.B, broadband, ir, light)

			// update to google sheet in a goroutine
			go func() {
				WriteRowToSheet(sheet, []string{
					now.Format("2006.01.02 15:04:05"),
					fmt.Sprintf("%.1f", temp),
					fmt.Sprintf("%.1f", hum),
					fmt.Sprintf("%d", broadband),
					fmt.Sprintf("%d", ir),
					fmt.Sprintf("%d", light),
				})
			}()
		})

	}

	robot := gobot.NewRobot("temperatureBot",
		[]gobot.Connection{r},
		[]gobot.Device{led, lux},
		work,
	)

	robot.Start()
}
