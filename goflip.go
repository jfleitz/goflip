package goflip

import log "github.com/Sirupsen/logrus"

type GoFlip struct {
	devices         arduinos
	Scores          []int
	BallInPlay      int
	ExtraBall       bool
	TotalBalls      int
	MaxPlayers      int
	LampControl     chan deviceMessage
	SolenoidControl chan deviceMessage
	SwitchEvents    chan SwitchEvent
	switchStates    []bool
}

type SwitchEvent struct {
	SwitchID int
	Pressed  bool
}

const (
	off       = 0    //can be used for Solenoids or Lamp
	on        = 1    //can be used for Solenoids or Lamp
	slowBlink = 2    //lamp Only
	fastBlink = 3    //lamp Only
	ack       = iota //when used, it doesn't matter what ID is.
)

const consoleMode bool = false

type deviceMessage struct {
	id    int
	value int //set to one of the constants
}

func (g *GoFlip) Init() bool {

	if !consoleMode {
		if !g.devices.Connect() {
			log.Errorln("Devices were unable to connect. Check USB connections")
			//return false make this configurable.
		}
	}

	g.LampControl = make(chan deviceMessage, 100)
	g.SolenoidControl = make(chan deviceMessage, 100)
	g.SwitchEvents = make(chan SwitchEvent, 100)

	//read from a config..but for now, we hard code.
	g.MaxPlayers = 2
	g.TotalBalls = 3
	g.switchStates = make([]bool, 64)
	return true
}

func (g *GoFlip) Start(m func(SwitchEvent)) bool {
	go g.LampSubscriber() //-temp
	go g.SolenoidSubscriber()

	//handler for calling switch event routine:

	go func() {
		log.Infoln("Starting switch monitoring")
		for {
			//buf := make([]byte, 16) //shouldn't be over 1 byte really
			buf := g.devices.switchMatrix.ReadSwitch()
			log.Infof("Received %d switch events", len(buf))

			//we should never receive 0 switch events... so if we do, maybe we stop and reinitialize??

			for _, sw := range buf {
				g.switchStates[sw.SwitchID] = sw.Pressed
				m(sw)
			}
		}

	}()

	return true
}

func (g *GoFlip) LampSubscriber() {
	log.Infoln("Starting LDU subscribing")
	for {

		msg := <-g.LampControl
		//	log.Infoln("received message")

		//select {
		//case msg := <-_lmpControl:
		//format the message and send to the LDU
		//{[lampID][ControlID]} where ControlID is 0 = 0 off,1 = on,2 = slow,3 = fast
		switch msg.value {
		case on:
			g.devices.ldu.SendMessage(msg)
		case off:
			g.devices.ldu.SendMessage(msg)
		case slowBlink:
			g.devices.ldu.SendMessage(msg)
		case fastBlink:
			g.devices.ldu.SendMessage(msg)

		default:
			log.Errorf("Invalid message value received for Lamp Control: %d", msg.value)
		}

		//	}
	}
}

func (g *GoFlip) SolenoidSubscriber() {
	log.Infoln("Starting Solenoid subscribing")

	for {

		msg := <-g.SolenoidControl
		log.Infoln("received message")
		//select {
		//case msg := <-g.SolenoidControl:
		//format the message and send to the LDU
		//{[lampID][ControlID]} where ControlID is 0 = 0 off,1 = on,2 = slow,3 = fast
		switch msg.value {
		case on:
			g.devices.sdu.SendShortMessage(msg)
		case off:
			g.devices.sdu.SendShortMessage(msg)
			break
		default:
			if msg.value >= 255 {
				log.Errorf("Invalid message value received for Solenoid Control: %d", msg.value)
			} else {
				g.devices.sdu.SendShortMessage(msg)
			}

		}

		//}
	}
}

func (g *GoFlip) LampOn(lampID int) {
	var msg deviceMessage
	msg.id = lampID
	msg.value = on

	g.LampControl <- msg
}

func (g *GoFlip) LampOff(lampID int) {
	var msg deviceMessage
	msg.id = lampID
	msg.value = off

	g.LampControl <- msg
}

func (g *GoFlip) LampSlowBlink(lampID int) {
	var msg deviceMessage
	msg.id = lampID
	msg.value = slowBlink

	g.LampControl <- msg
}

func (g *GoFlip) LampFlastBlink(lampID int) {

	var msg deviceMessage
	msg.id = lampID
	msg.value = fastBlink

	g.LampControl <- msg
}

func (g *GoFlip) SolenoidOff(solID int) {
	var msg deviceMessage
	msg.id = solID
	msg.value = off

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
