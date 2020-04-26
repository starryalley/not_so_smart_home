package main

import (
	"fmt"
	"log"
	"path"
	"time"

	"github.com/scotow/notigo"

	"github.com/starryalley/smart_home/pkg/cmds"
	"github.com/starryalley/smart_home/pkg/logs"
)

// for RPi
//const binPath = "/usr/local/lib/nodejs/bin/"
const binPath = "/usr/local/bin/"

// rear door sensor IDs: enter your MIIO device ID
const doorSensorID = "158d0002676aec"

// update sensor state in this interval
const checkInterval = 30 * time.Second

// how long if door is left open is considered a warning
const doorOpenWarningTimeout = 2 * time.Minute

// IFTTT: enter your IFTTT webhook key and event name below
const iftttKey = "your_ifttt_key"
const iftttEventName = "your_ifttt_webhook_event"

// true if door is opened, false if closed
var doorOpened bool

func getMagnetSensorContact(sensorID string) (bool, error) {
	outs, err := cmds.RunCmdWithResult(fmt.Sprintf("%s %s control %s contact", path.Join(binPath, "node"), path.Join(binPath, "miio"), doorSensorID))
	if err != nil {
		return false, err
	}
	if len(outs) == 3 {
		//log.Printf("magnet sensor contact:%s\n", outs[1])
		if string(outs[1]) == "true" {
			return true, nil
		}
		if string(outs[1]) == "false" {
			return false, nil
		}
		return false, fmt.Errorf("Unexpected sensor output:%v", outs[1])
	}
	return false, fmt.Errorf("Unexpected miio command output:%v", outs)
}

func updateSensorState(eventCh chan<- string, quit <-chan struct{}) {
	log.Printf("door sensor updater started\n")
	for {
		select {
		case <-quit:
			log.Printf("door sensor updater exited\n")
			return
		case <-time.After(checkInterval):
			closed, err := getMagnetSensorContact(doorSensorID)
			if err != nil {
				log.Printf("Error getting sensor state:%s\n", err)
				// ignore for now
				continue
			}
			// when sensor state is different
			if doorOpened == closed {
				if doorOpened {
					eventCh <- "door_closed"
				} else {
					eventCh <- "door_opened"
				}
				doorOpened = !doorOpened
			}
		}
	}
}

func sendNotification(title, message string) error {
	notification := notigo.NewNotification(title, message)
	key := notigo.Key(iftttKey)

	err := key.SendEvent(notification, iftttEventName)
	if err != nil {
		return err
	}

	log.Println("Notification sent through IFTTT")
	return nil
}

func monitorDoor(quit <-chan struct{}) {
	select {
	case <-time.After(doorOpenWarningTimeout):
		if err := sendNotification("Rear Door Warning", "Door left open for too long"); err != nil {
			log.Printf("Error sending notification:%s\n", err)
		}
	case <-quit:
	}
	log.Println("door monitor exited")
}

func main() {
	logs.SetupSyslog("DoorMonitor")

	eventCh := make(chan string)
	quitCh := make(chan struct{})
	defer close(quitCh)

	// start sensor updater
	go updateSensorState(eventCh, quitCh)

	// wait for event to happen
	var quitMonCh chan struct{}
	for {
		select {
		case event := <-eventCh:
			log.Printf("Event:%s\n", event)
			if event == "door_opened" {
				// start door monitoring
				if quitMonCh == nil {
					quitMonCh = make(chan struct{})
				}
				go monitorDoor(quitMonCh)
			} else if event == "door_closed" {
				// stop door monitoring
				if quitMonCh != nil {
					close(quitMonCh)
					quitMonCh = nil
				}
			}
		}
	}
}
