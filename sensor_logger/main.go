package main

import (
	"log"
	"log/syslog"
	"time"

	"github.com/d2r2/go-dht"
	logger "github.com/d2r2/go-logger"
	"github.com/gofrs/flock"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)

// ========== Settings =========
const maxRetry = 3

// update interval in minutes
const updateInterval = 10

// GPIO number for DHT temperature sensor
const gpioTemp = 4

// google sheet credential json file
const googleSheetCredential = "/home/starryalley/.secret/google_sheet_credentials.json"

// =============================

func getTempHum(lock *flock.Flock) (float32, float32, error) {
	logger.ChangePackageLogLevel("dht", logger.ErrorLevel)
	var temperature, humidity float32
	for {
		locked, err := lock.TryLock()
		if err != nil {
			log.Printf("unable to lock for DHT22:%v\n", err)
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

func main() {
	// configure logger to write to syslog
	logwriter, err := syslog.New(syslog.LOG_NOTICE, "SensorLogger")
	if err != nil {
		log.Printf("Unable to configure logger to write to syslog:%s\n", err)
		return
	}
	log.SetOutput(logwriter)
	log.SetFlags(0)

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
			temp, hum, err := getTempHum(fileLockTemp)
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
