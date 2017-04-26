package goflip

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"go.bug.st/serial.v1"
)

const (
	unknownUSB      = iota
	swmatrixArduino = iota
	lduArduino      = iota
	sduArduino      = iota
)

func inittest() {
	log.SetOutput(os.Stdout)
}

func maintest() {
	var lduPort = "unknown"
	var sduPort = "unknown"
	var swPort = "unknown"

	log.Info("Starting GoFlip!")

	arduinos := findArduinos()

	for _, s := range arduinos {
		id, err := identifyArduino(s)
		if err == nil {
			switch id {
			case swmatrixArduino:
				swPort = s
				break
			case lduArduino:
				lduPort = s
				break
			case sduArduino:
				sduPort = s
				break
			default:
				log.Infof("Unknown connected device: %v", s)
			}
		}
	}

	log.Infof("SwitchMatrix Arduino connected at: %s", swPort)
	log.Infof("LDU Arduino connected at: %s", lduPort)
	log.Infof("SDU Arduino connected at: %s", sduPort)

	sendCommand(lduPort, "p2") //slow flash the LDU led
	sendCommand(sduPort, "@y") //turn the LED on the SDU board for awhile

}

func sendCommand(port string, command string) error {
	whatToSend := fmt.Sprintf("{%s}", command)

	//[]byte(whatToSend)
	write := []byte(whatToSend)

	if len(write) > 0 {
		c := &serial.Mode{BaudRate: 57600}
		s, _ := serial.Open(port, c)
		//wait 3 secs
		time.Sleep(3 * time.Second)
		defer s.Close()

		_, err := s.Write(write)
		if err != nil {
			return err
		}

	}
	return nil
}

//find what we devices we have...

//Must have at least a switch matrix input min
//Output perpipherals (sdu/ldu/etc) are optional..but probably necessary

// findArduino looks for the file that represents the Arduino
// serial connection. Returns the fully qualified path to the
// device if we are able to find a likely candidate for an
// Arduino, otherwise an empty string if unable to find
// something that 'looks' like an Arduino device.
func findArduinos() []string {
	var found []string

	contents, _ := ioutil.ReadDir("/dev")

	// Look for what is mostly likely the Arduino device
	for _, f := range contents {
		if strings.Contains(f.Name(), "tty.usbserial") ||
			strings.Contains(f.Name(), "ttyUSB") {
			//call and ack the device.
			val := "/dev/" + f.Name()
			found = append(found, val)
		}
	}

	// Have not been able to find a USB device that 'looks'
	// like an Arduino.
	return found
}

func connectToArduino(port string) {
	c := &serial.Mode{BaudRate: 57600}
	s, _ := serial.Open(port, c)

	defer s.Close()

}

func identifyArduino(port string) (ret int, err error) {
	c := &serial.Mode{BaudRate: 57600}
	s, _ := serial.Open(port, c)
	//wait 3 secs
	time.Sleep(3 * time.Second)
	defer s.Close()

	_, err = s.Write([]byte("|"))

	if err != nil {
		log.Errorf("Error in identifyArduino: %v", err)
		return 0, err
	}

	buf := make([]byte, 128)
	n, err := s.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	//last character should be what we got back.
	switch buf[n-1] {
	case 'a':
		return swmatrixArduino, nil

	case 'b':
		return lduArduino, nil

	case 'c':
		return sduArduino, nil

	}
	//    log.Print("%q", buf[:n])

	return unknownUSB, nil
}
