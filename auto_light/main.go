package main

import (
	"log"
	"log/syslog"
	"os/exec"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/kelvins/sunrisesunset"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)

// Ringwood, VIC, Australia (AEST)
const Latitude = -37.8114
const Longitude = 145.2306
const UTC = 10

// calculated sunrise and sunset
var sunriseTime time.Time
var sunsetTime time.Time

// coming midnight
var midnight time.Time

func runCmd(cmd string) {
	cmds := strings.Split(cmd, " ")
	if err := exec.Command(cmds[0], cmds[1:]...).Run(); err != nil {
		log.Printf("Error running shell command:%v\n", err)
	}
}

var lightOn bool = false

func checkLight() (bool, error) {
	cmds := strings.Split("/usr/local/lib/nodejs/bin/node /usr/local/lib/nodejs/bin/miio control 158d0002498b8e power", " ")
	out, err := exec.Command(cmds[0], cmds[1:]...).Output()
	if err != nil {
		return false, err
	}
	outs := strings.Split(string(out), "\n")
	log.Printf("Light On:%s", outs[1])
	if string(outs[1]) == "true" {
		return true, nil
	}
	return false, nil
}

func turnOnLight() {
	if !lightOn {
		log.Println("Turning on light")
		runCmd("/usr/local/lib/nodejs/bin/node /usr/local/lib/nodejs/bin/miio control 158d0002498b8e power true")
		lightOn = true
	}
}

func turnOffLight() {
	if lightOn {
		log.Println("Turning off light")
		runCmd("/usr/local/lib/nodejs/bin/node /usr/local/lib/nodejs/bin/miio control 158d0002498b8e power false")
		lightOn = false
	}
}

func updateSunTime() {
	now := time.Now()
	p := sunrisesunset.Parameters{
		Latitude:  Latitude,
		Longitude: Longitude,
		UtcOffset: UTC,
		Date:      time.Now(),
	}

	// calculate the sunrise and sunset times
	sunrise, sunset, err := p.GetSunriseSunset()
	if err != nil {
		log.Fatal(err)
	}

	// set current date to sunrise/sunset time
	sunrise = time.Date(now.Year(), now.Month(), now.Day(),
		sunrise.Hour(), sunrise.Minute(), sunrise.Second(), 0, now.Location())
	sunset = time.Date(now.Year(), now.Month(), now.Day(),
		sunset.Hour(), sunset.Minute(), sunset.Second(), 1, now.Location())
	log.Printf("Sunrise: %v, Sunset: %v\n", sunrise.Format("15:04:05"), sunset.Format("15:04:05"))
	sunriseTime, sunsetTime = sunrise, sunset

	// create a time representing the coming midnight
	midnight = now.Add(24 * time.Hour)
	midnight = time.Date(midnight.Year(), midnight.Month(), midnight.Day(),
		0, 0, 0, 0, now.Location())
	log.Printf("Coming midnight: %v\n", midnight.Format("Mon Jan 2 15:04:05 MST 2006"))

	lightOn, err = checkLight()
	if err != nil {
		log.Printf("Check light failed:%v\n", err)
	}
}

// check if current time is during day
func isBright() bool {
	now := time.Now()
	if now.After(sunriseTime) && now.Before(sunsetTime) {
		return true
	}
	return false
}

func main() {
	// configure logger to write to syslog
	logwriter, err := syslog.New(syslog.LOG_NOTICE, "AutoLight")
	if err == nil {
		log.SetOutput(logwriter)
		log.SetFlags(0)
	}
	// for concurrent access to light sensor
	fileLock := flock.New("/var/lock/tsl2561.lock")

	r := raspi.NewAdaptor()
	lux := i2c.NewTSL2561Driver(r, i2c.WithBus(0), i2c.WithAddress(0x39), i2c.WithTSL2561Gain1X)

	// do the first sunrise/sunset calculation
	updateSunTime()

	work := func() {
		gobot.Every(10*time.Second, func() {
			// check if sun already sets
			if !isBright() {

				// if now is past midnight, let's turn off light
				//  and then do nothing until the next sunset
				if time.Now().After(midnight) {
					go turnOffLight()
					updateSunTime()
					return
				}

				var broadband, ir uint16
				for {
					locked, err := fileLock.TryLock()
					if err != nil {
						log.Printf("Unable to lock:%v\n", err)
						return
					}
					if locked {
						defer fileLock.Unlock()
						// get current light measurement
						broadband, ir, err = lux.GetLuminocity()
						if err != nil {
							log.Printf("read luminocity failed:%v\n", err)
							return
						}
						break
					}
				}
				light := lux.CalculateLux(broadband, ir)

				// check if light is off
				if light <= 15 {
					// light isn't on, let's turn on
					log.Printf("BB: %v, IR: %v, Lux: %v => Now it's dark after sunset!\n",
						broadband, ir, light)
					// wait until light is turned on
					turnOnLight()
				} else if light > 100 {
					// too bright, turn off light
					turnOffLight()
				}
			}
		})
	}

	robot := gobot.NewRobot("auto_light_on",
		[]gobot.Connection{r},
		[]gobot.Device{lux},
		work,
	)

	robot.Start()
}
