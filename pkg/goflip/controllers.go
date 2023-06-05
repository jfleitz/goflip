package goflip

//JAF TODO... need to send a keepalive to each arduino

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const KeepAliveMS = 250

func LampSubscriber() {
	g := GetMachine()
	log.Debugln("Starting LDU subscribing")

	g.devices.ldu.consoleMode = g.ConsoleMode

	for {
		select {
		case msg := <-lampControl:
			if msg.id == QUIT {
				return
			}

			if msg.value > FastBlink {
				log.Errorf("Invalid message value received for Lamp Control: %d", msg.value)
				return
			}
			msg.value += 1 //hate to do this, but have to so that the constants btw arduino and goflip are compatible for now. Fix later

			g.devices.ldu.SendMessage(msg)
		case <-time.After(time.Millisecond * KeepAliveMS):
			msg := deviceMessage{id: 0, value: 0}
			g.devices.ldu.SendMessage(msg)
		}
	}
}

func SolenoidSubscriber() {
	g := GetMachine()
	log.Debugln("Starting Solenoid subscribing")

	for {

		msg := <-solenoidControl
		if msg.id == QUIT {
			return
		}

		log.Debugf("Solenoid Msg id:%d value:%d\n", msg.id, msg.value)
		//select {
		//case msg := <-g.SolenoidControl:
		//format the message and send to the LDU
		//{[lampID][ControlID]} where ControlID is 0 = 0 off,1 = on,2 = slow,3 = fast
		switch msg.value {
		case On:
			g.devices.sdu.SendShortMessage(msg, 3)
		case Off:
			g.devices.sdu.SendShortMessage(msg, 3)
		default:
			if msg.value >= 255 {
				log.Errorf("Invalid message value received for Solenoid Control: %d", msg.value)
			} else {
				g.devices.sdu.SendShortMessage(msg, 3)
			}

		}

		//}
	}
}

func SetLampState(lampID int, state int) {
	g := GetMachine()
	if _, ok := g.lampStates[lampID]; !ok {
		g.lampStates[lampID] = state
	} else {
		g.lampStates[lampID] = state
	}

	var msg deviceMessage
	msg.id = lampID
	msg.value = state

	lampControl <- msg
}

func LampOn(lampID ...int) {
	for _, l := range lampID {
		SetLampState(l, On)
	}
}

func LampOff(lampID ...int) {
	for _, l := range lampID {
		SetLampState(l, Off)
	}
}

func LampSlowBlink(lampID ...int) {
	for _, l := range lampID {
		SetLampState(l, SlowBlink)
	}
}

func LampFastBlink(lampID ...int) {

	for _, l := range lampID {
		SetLampState(l, FastBlink)
	}
}

func SolenoidOff(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = Off

	solenoidControl <- msg
}
func SolenoidFire(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = 2 //should be about a 100ms pule when at 2

	solenoidControl <- msg
}

func SolenoidAlwaysOn(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = 0x07

	solenoidControl <- msg
}

func FlipperControl(on bool) {
	var msg deviceMessage
	msg.id = 0x0f
	if on {
		msg.value = 0x03
	} else {
		msg.value = 0x02
	}

	solenoidControl <- msg
}

func SolenoidOnDuration(solID int, duration int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = duration
	solenoidControl <- msg
}

func SwitchPressed(swID int) bool {
	g := GetMachine()
	return g.switchStates[swID]
}

func GetLampState(lampID int) int {
	g := GetMachine()
	if state, ok := g.lampStates[lampID]; ok {
		return state
	}
	return Off

}
