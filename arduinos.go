/*
These are the methods for connecting to all of the supported arduino controllers, including:
SwitchMatrix
Solenoid Driver Unit (SDU)
Lamp Driver Unit (LDU)
*/
package goflip

import (
	"io"
	"io/ioutil"
	"strings"
	"time"

	"strconv"

	log "github.com/Sirupsen/logrus"
	"go.bug.st/serial.v1"
	//"github.com/huin/goserial"
)

type arduino struct {
	port        string
	conn        io.ReadWriteCloser
	consoleMode bool
}

type swarduino struct {
	arduino
}

type arduinos struct {
	switchMatrix swarduino
	ldu          arduino
	sdu          arduino
	ports        []string
}

func (a *arduinos) ReadPorts() {
	a.ports = nil

	contents, _ := ioutil.ReadDir("/dev")

	// Look for what is mostly likely the Arduino device
	for _, f := range contents {
		if strings.Contains(f.Name(), "tty.usbserial") ||
			strings.Contains(f.Name(), "ttyUSB") {
			//call and ack the device.
			val := "/dev/" + f.Name()
			a.ports = append(a.ports, val)
			log.Infof("Found arduino at %s\n", val)
		}
	}
}

func (a *arduinos) Connect() bool {
	if len(a.ports) == 0 {
		a.ReadPorts()
	}

	//for each port, we need to open a connection then see what it is, and if it fits the bill, save a ref for it
	for _, port := range a.ports {
		mode := &serial.Mode{
			BaudRate: 57600,
		}

		s, err := serial.Open(port, mode)
		if err != nil {
			log.Fatal(err)
		}

		//	c := &goserial.Config{Name: port, Baud: 57600}

		//	s, _ := goserial.OpenPort(c)
		//wait 3 secs
		time.Sleep(3 * time.Second)

		_, err = s.Write([]byte("|"))

		if err != nil {
			log.Errorf("Error in Connect: %v", err)
			return false
		}

		buf := make([]byte, 128)
		_, err = s.Read(buf)
		if err != nil {
			log.Fatal(err)
		}

		//first character should be what we got back.
		log.Infof("buf is %v", buf[0])
		switch buf[0] {
		case 'a':
			a.switchMatrix.port = port
			a.switchMatrix.conn = s
			a.switchMatrix.consoleMode = false
			log.Infof("SwitchMatrix Arduino connected at %s\n", port)
		case 'b':
			a.ldu.port = port
			a.ldu.conn = s
			a.ldu.consoleMode = false
			log.Infof("LDU Arduino connected at %s\n", port)
		case 'c':
			a.sdu.port = port
			a.sdu.conn = s
			a.sdu.consoleMode = false
			log.Infof("SDU Arduino connected at %s\n", port)
		}
	}

	if len(a.switchMatrix.port) > 0 &&
		len(a.ldu.port) > 0 &&
		len(a.sdu.port) > 0 {
		return true
	}

	return false

}

func (a *arduinos) Disconnect() {

	if !a.switchMatrix.consoleMode && a.switchMatrix.conn != nil {
		_ = a.switchMatrix.conn.Close()
	}

	if !a.ldu.consoleMode && a.ldu.conn != nil {
		_ = a.ldu.conn.Close()

	}

	if !a.sdu.consoleMode && a.sdu.conn != nil {
		_ = a.sdu.conn.Close()
	}
}

func (ard *swarduino) ReadSwitch() []SwitchEvent {
	var ret []SwitchEvent
	buf := make([]byte, 16) //shouldn't be over 1 byte really

	if ard.consoleMode {
		//str, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		str := "1 2"
		vals := strings.Split(str, " ")

		if len(vals) == 2 {
			var s SwitchEvent
			s.SwitchID, _ = strconv.Atoi(vals[0])
			log.Infof("vals[1] = %v", vals[1])
			Pressed, _ := strconv.Atoi(vals[1])
			log.Infof("Pressed = %d", Pressed)
			s.Pressed = (Pressed == 1)
			ret = make([]SwitchEvent, 1)

			log.Infof("Switch %d Pressed = %v\n", s.SwitchID, s.Pressed)
			ret[0] = s
			return ret
		}
		return nil
	}

	n, err := ard.conn.Read(buf)
	if err != nil {
		log.Errorf("Error reading switch: %v", err)
	}

	log.Infof("bytes received: %d", n)

	ret = make([]SwitchEvent, n)
	for i := 0; i < n; i++ {
		sw := buf[i]
		var s SwitchEvent
		s.Pressed = !(sw&0x01 > 0)
		s.SwitchID = int(sw >> 1)
		ret = append(ret, s)

		log.Infof("SW Received: %v, SwitchID=%d, Pressed = %v", sw, s.SwitchID, s.Pressed)
	}
	return ret
}

func (a *arduino) SendMessage(d deviceMessage) error {
	//value received is zero based. We need to convert to "arduino based" by adding 48 first

	tosend := make([]byte, 4)

	//toSend := fmt.Sprintf("{%d%d}", d.id, d.value+48)
	tosend[0] = byte('{')
	tosend[1] = byte(d.id + 48)
	tosend[2] = byte(d.value + 48)
	tosend[3] = byte('}')
	//	log.Infof("Sending arduino message for %d:%d to %s", d.id, d.value, a.port)
	_, err := a.conn.Write(tosend)
	return err
}

//Short Message format is 1 byte long. Top 5 bits is the ID, bottom 3 bits are the value
func (a *arduino) SendShortMessage(d deviceMessage) error {
	b := make([]byte, 1)
	b[0] = (byte)(d.id << 3)
	b[0] = b[0] | (byte)(0x07&d.value)

	//	log.Infof("Sending short message for %d:%d to %s", d.id, d.value, a.port)
	_, err := a.conn.Write(b)

	return err
}
