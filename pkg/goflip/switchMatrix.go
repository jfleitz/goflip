package goflip

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

const swBufferSize = 100
const swAckNakTime = 500
const initCommand = 1 //recd byte value of 1, means that the Switch Matrix just initialized and is ready
const ackCommand = 2  //send byte value of 2, we should get a NAK back
const nakCommand = 3  //recd byte value of 3, means we got a response from our transmission

//SwitchValue contains what was received from the SwitchMatrix board. value = true means just Pressed. value = false means just released
type SwitchValue struct {
	SwitchID int8
	value    bool
}

//SwitchMatrixHandler communicates with the SwitchMatrix interface board using the port provided
func SwitchMatrixHandler(serialPort io.ReadWriteCloser, gameIn chan SwitchValue) error {
	if serialPort == nil {
		return nil
	}

	sin := make(chan byte, swBufferSize)
	go func() {
		for {
			var readVal []byte
			n, err := serialPort.Read(readVal)
			if err != nil {
				log.Errorf("SW Serial Error: %v", err)
			} else {
				for i := 0; i < n; i++ {
					sin <- readVal[i] //send each sep
				}
			}
		}
	}()

	//JAF TODO... What was I thinking???? This seems like it could be replaced with just a sleep return on the case statement below. Check this
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(swAckNakTime * time.Millisecond)
		timeout <- true
	}()

	for {
		select {
		case swVal := <-sin:
			switch swVal {
			case nakCommand:
			case initCommand:
			default:
				//send to GameProcess channel
				swid := int8(swVal >> 1)
				v := false
				if swVal&0xfe > 0 {
					v = true
				}
				gameIn <- SwitchValue{
					SwitchID: swid,
					value:    v}
			}
			//If valid, send on to the Game Handler
		case <-timeout:
			//need to send an ack
		}
	}

}
