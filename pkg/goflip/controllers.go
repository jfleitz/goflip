package goflip

//JAF TODO... need to send a keepalive to each arduino

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const KeepAliveMS = 250

func (g *GoFlip) LampSubscriber() {
	log.Debugln("Starting LDU subscribing")
	for {
		select {
		case msg := <-g.LampControl:
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

func (g *GoFlip) SolenoidSubscriber() {
	log.Debugln("Starting Solenoid subscribing")

	for {

		msg := <-g.SolenoidControl
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

func (g *GoFlip) SetLampState(lampID int, state int) {
	if _, ok := g.lampStates[lampID]; !ok {
		g.lampStates[lampID] = state
	} else {
		g.lampStates[lampID] = state
	}

	var msg deviceMessage
	msg.id = lampID
	msg.value = state

	g.LampControl <- msg
}

func (g *GoFlip) LampOn(lampID ...int) {
	for _, l := range lampID {
		g.SetLampState(l, On)
	}

}

func (g *GoFlip) LampOff(lampID ...int) {
	for _, l := range lampID {
		g.SetLampState(l, Off)
	}
}

func (g *GoFlip) LampSlowBlink(lampID ...int) {
	for _, l := range lampID {
		g.SetLampState(l, SlowBlink)
	}
}

func (g *GoFlip) LampFastBlink(lampID ...int) {

	for _, l := range lampID {
		g.SetLampState(l, FastBlink)
	}
}

func (g *GoFlip) SolenoidOff(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = Off

	g.SolenoidControl <- msg
}
func (g *GoFlip) SolenoidFire(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = 2 //should be about a 100ms pule when at 2

	g.SolenoidControl <- msg
}

func (g *GoFlip) SolenoidAlwaysOn(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = 0x07

	g.SolenoidControl <- msg
}

func (g *GoFlip) FlipperControl(on bool) {
	var msg deviceMessage
	msg.id = 0x0f
	if on {
		msg.value = 0x03
	} else {
		msg.value = 0x02
	}
	g.SolenoidControl <- msg
}

func (g *GoFlip) SolenoidOnDuration(solID int, duration int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = duration

	g.SolenoidControl <- msg
}

func (g *GoFlip) SwitchPressed(swID int) bool {
	return g.switchStates[swID]
}

func (g *GoFlip) GetLampState(lampID int) int {
	if state, ok := g.lampStates[lampID]; ok {
		return state
	}
	return Off

}
