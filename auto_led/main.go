package main

import (
	"log"
	"log/syslog"
	"math"
	"time"

	"github.com/d2r2/go-dht"
	logger "github.com/d2r2/go-logger"
	"github.com/gofrs/flock"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"
)

// ========== Settings =========
// update interval in seconds
const updateInterval = 60

// GPIO number for DHT temperature sensor
const gpioTemp = 4

// Pin names for LED R,G,B pins
const pinR = "11"
const pinG = "13"
const pinB = "15"

// max/min temperature in the room
const maxTemp = 32
const minTemp = 8

// max brightness 100
const ledBrightness = 100

// =============================

type color struct {
	R, G, B uint8
}

var definedColors = [...]color{
	{255, 0, 255}, //purple
	{0, 0, 255},   //blue
	{0, 255, 255}, //cyan
	{0, 255, 0},   //green
	{255, 255, 0}, //yellow
	{255, 0, 0},   //red
}

const numColor = len(definedColors)

var lastTemp float32
var lastTempColor color
var lastAqiColor color

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

func aqiToColor(idx float64) color {
	if idx <= 50 {
		// green
		return color{0, 255, 0}
	} else if idx <= 100 {
		// yellow
		return color{255, 255, 0}
	} else if idx <= 150 {
		// orange
		return color{255, 127, 0}
	} else if idx <= 200 {
		// red
		return color{255, 0, 0}
	} else if idx <= 300 {
		// purple
		return color{255, 0, 255}
	} else {
		// brown #7E0023
		return color{126, 0, 35}
	}
}

func getTempHum(lock *flock.Flock) (float32, float32, error) {
	logger.ChangePackageLogLevel("dht", logger.ErrorLevel)
	var temperature, humidity float32
	for {
		locked, err := lock.TryLock()
		if err != nil {
			log.Printf("Unable to lock for DHT22:%v\n", err)
			return 0, 0, err
		}
		if locked {
			temperature, humidity, _, err =
				dht.ReadDHTxxWithRetry(dht.DHT22, gpioTemp, false, 30)
			if err != nil {
				lock.Unlock()
				return 0, 0, err
			}
			lock.Unlock()
			break
		}
	}
	return temperature, humidity, nil
}

func updateAQI() {
	aqi, err := getAQI()
	if err != nil {
		log.Println("get AQI error:", err)
		return
	}
	lastAqiColor = aqiToColor(aqi)
}

func updateTemperature(fileLockTemp *flock.Flock) {
	temp, _, err := getTempHum(fileLockTemp)
	if err != nil {
		log.Printf("read temperature failed:%v\n", err)
		return
	}
	// temperature changes
	if lastTemp != temp {
		lastTempColor = temperatureToColor(temp)
		lastTemp = temp
		log.Printf("Temperature:%.01f°C\n", lastTemp)
	}
}

func main() {
	// configure logger to write to syslog
	logwriter, err := syslog.New(syslog.LOG_NOTICE, "AutoLED")
	if err != nil {
		log.Printf("Unable to configure logger to write to syslog:%s\n", err)
		return
	}
	log.SetOutput(logwriter)
	log.SetFlags(0)

	// possible multi-process access
	fileLockTemp := flock.New("/var/lock/dht22.lock")

	r := raspi.NewAdaptor()
	led := gpio.NewRgbLedDriver(r, pinR, pinG, pinB)

	work := func() {
		// update temperature and LED every 1 min
		gobot.Every(updateInterval*time.Second, func() {
			updateTemperature(fileLockTemp)
			go func() {
				// blink AQI for some time
				//log.Printf("Set AQI RGB LED:%v,%v,%v\n", lastAqiColor.R, lastAqiColor.G, lastAqiColor.B)
				led.SetRGB(lastAqiColor.R, lastAqiColor.G, lastAqiColor.B)
				for i := 0; i < 8; i++ {
					time.Sleep(time.Second)
					led.Toggle() // off
					time.Sleep(200 * time.Millisecond)
					led.Toggle() // on
				}
				// solid RGB for temperature
				//log.Printf("Set Temperature RGB LED:%v,%v,%v\n", lastTempColor.R, lastTempColor.G, lastTempColor.B)
				led.SetRGB(lastTempColor.R, lastTempColor.G, lastTempColor.B)
			}()
		})
		// update AQI every 1 hour
		gobot.Every(time.Hour, func() {
			updateAQI()
		})
	}

	robot := gobot.NewRobot("temperatureBot",
		[]gobot.Connection{r},
		[]gobot.Device{led},
		work,
	)

	// get initial AQI
	updateAQI()

	robot.Start()
}
