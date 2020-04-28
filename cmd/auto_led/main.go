package main

import (
	"log"
	"time"

	"github.com/gofrs/flock"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/raspi"

	"github.com/starryalley/smart_home/pkg/colors"
	"github.com/starryalley/smart_home/pkg/logs"
	"github.com/starryalley/smart_home/pkg/sensors"
)

const (
	updateInterval = 60   // update interval in seconds
	pinR           = "11" // Pin names for LED R pins
	pinG           = "13" // Pin names for LED G pins
	pinB           = "15" // Pin names for LED B pins
)

var (
	lastTemp      float32
	lastTempColor colors.Color
	lastAqiColor  colors.Color
)

func updateAQI() {
	aqi, err := getAQI()
	if err != nil {
		log.Println("get AQI error:", err)
		return
	}
	lastAqiColor = colors.AQIToColor(aqi)
}

func updateTemperature(fileLockTemp *flock.Flock) {
	temp, _, err := sensors.GetTempHum(fileLockTemp)
	if err != nil {
		log.Printf("read temperature failed:%v\n", err)
		return
	}
	// temperature changes
	if lastTemp != temp {
		lastTempColor = colors.TemperatureToColor(temp)
		lastTemp = temp
		log.Printf("Temperature:%.01f°C\n", lastTemp)
	}
}

func main() {
	logs.SetupSyslog("AutoLED")

	// possible multi-process access
	fileLockTemp := flock.New("/var/lock/dht22.lock")

	r := raspi.NewAdaptor()
	led := gpio.NewRgbLedDriver(r, pinR, pinG, pinB)

	work := func() {
		// update temperature and LED every 1 min
		gobot.Every(updateInterval*time.Second, func() {
			updateTemperature(fileLockTemp)
			go func() {
				// alternating between AQI and temperature color for some time
				//log.Printf("Set AQI RGB LED:%v,%v,%v\n", lastAqiColor.R, lastAqiColor.G, lastAqiColor.B)
				for i := 0; i < 10; i++ {
					// set to AQI color
					led.SetRGB(lastAqiColor.R, lastAqiColor.G, lastAqiColor.B)
					time.Sleep(500 * time.Millisecond)
					// set to temperature color
					led.SetRGB(lastTempColor.R, lastTempColor.G, lastTempColor.B)
					time.Sleep(500 * time.Millisecond)
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
