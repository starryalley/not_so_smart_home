package logs

import (
	"log"
	"log/syslog"
)

// SetupSyslog configures log to write to syslog
func SetupSyslog(name string) error {
	logwriter, err := syslog.New(syslog.LOG_NOTICE, name)
	if err != nil {
		log.Printf("Unable to configure logger to write to syslog:%s\n", err)
		return err
	}
	log.SetOutput(logwriter)
	log.SetFlags(0)
	return nil
}
