package main

import (
	"log"
	"time"

	"github.com/gofrs/flock"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"

	"github.com/starryalley/smart_home/pkg/logs"
	"github.com/starryalley/smart_home/pkg/sensors"
)

const (
	maxRetry              = 3
	updateInterval        = 10                                                        // update interval in minutes
	googleSheetCredential = "/home/starryalley/.secret/google_sheet_credentials.json" // google sheet credential json file
)

// =============================

func main() {
	logs.SetupSyslog("SensorLogger")

	// possible multi-process access to those hardware
	fileLockLight := flock.New("/var/lock/tsl2561.lock")
	fileLockTemp := flock.New("/var/lock/dht22.lock")

	// initialise google sheet
	service, err := InitGoogleSheet(googleSheetCredential)
	if err != nil {
		log.Fatal(err)
	}

	// setup gobot
	r := raspi.NewAdaptor()
	lux := i2c.NewTSL2561Driver(r, i2c.WithBus(0), i2c.WithAddress(0x39), i2c.WithTSL2561Gain1X)

	work := func() {
		gobot.Every(updateInterval*time.Minute, func() {
			now := time.Now()
			temp, hum, err := sensors.GetTempHum(fileLockTemp)
			if err != nil {
				log.Printf("read temperature failed:%v\n", err)
				return
			}
			var broadband, ir uint16
			for {
				locked, err := fileLockLight.TryLock()
				if err != nil {
					log.Printf("unable to lock for light sensor:%v\n", err)
					time.Sleep(500 * time.Millisecond)
					continue
				}
				if locked {
					broadband, ir, err = lux.GetLuminocity()
					fileLockLight.Unlock()
					if err != nil {
						log.Printf("read luminocity failed:%v\n", err)
						return
					}
					break
				}
			}
			light := lux.CalculateLux(broadband, ir)

			log.Printf("T:%.01f°C H:%.01f%% BB:%v IR:%v Lux:%v\n",
				temp, hum, broadband, ir, light)

			// update to google sheet in a goroutine
			go func() {
				for i := 0; i < maxRetry; {
					row := []interface{}{
						now, //.Format("2006.01.02 15:04:05"),
						temp,
						hum,
						broadband,
						ir,
						light,
					}
					err = PrependRow(service, "15Zyy0_swv2YazuL9UdZ4YYkPfaIwTpPNtPHLAlsLtcY", "RawData!A2:F2", row)
					if err != nil {
						log.Println(err)
						i++
					} else {
						break
					}
				}
			}()

		})
	}

	robot := gobot.NewRobot("SensorLoggerBot",
		[]gobot.Connection{r},
		[]gobot.Device{lux},
		work,
	)

	robot.Start()
}
