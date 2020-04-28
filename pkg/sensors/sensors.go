package sensors

import (
	"log"

	logger "github.com/d2r2/go-logger"
	"github.com/gofrs/flock"
	"github.com/starryalley/go-dht"
)

// GPIO number for DHT temperature sensor
const gpioTemp = 4

// GetTempHum returns temperature and humidity from DHT sensor
func GetTempHum(lock *flock.Flock) (float32, float32, error) {
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
