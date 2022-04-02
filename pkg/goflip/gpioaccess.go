package goflip

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/host"
	"periph.io/x/periph/host/rpi"
)

var _disp [5][7]byte //this holds what we want to show on the display. Bytes are in terms of what the 74ls48 supports (0x0f is blank)
var _sound byte

const (
	blank           byte = 0x0f //what is sent to the 7448 on the display board to blank the 7 seg disp
	blankScore           = -1   //the numeric number that can be passed in as an integer to clear the display
	creditMatchDisp      = 4    //the number in the display array for the credit display
	creditLSD            = 6    //position in the display array for the 1's credit disp digit
	creditMSD            = 0    //position in the display array for the 10's credit disp digit

	pinDataClk     = 15
	pinLatchClk    = 16
	noSound        = 15
	ballInPlayDisp = 5
	creditDisp     = 6
)

//const blank byte = 0x0f
//const blankScore = -1
//const creditMatchDisp = 4

type displayMessage struct {
	display int
	value   int32
}

type soundMessage struct {
	soundID byte
}

type pwmMessage struct {
	portID int
	value  int
}

var endLoop bool

func (g *GoFlip) gpioInit() {
	clearDisplays()
	endLoop = false
	_sound = noSound
	go runGPIO()
}

func (g *GoFlip) gpioSubscriber() {
	err := initGPIO()
	if err != nil {
		return
	}

	log.Debugln("Starting gpio subscribing")
	for {
		select {
		case dspMsg := <-g.DisplayControl:
			if dspMsg.display > 0 && dspMsg.display <= 4 {
				setScore(dspMsg.display-1, dspMsg.value)
			} else {
				switch dspMsg.display {
				case ballInPlayDisp:
					setBallInPlay(int8(dspMsg.value))
				case creditDisp:
					setCredits(int8(dspMsg.value))
				}
			}
		case sndMsg := <-g.SoundControl:
			go func() {
				_sound = sndMsg.soundID
				time.Sleep(time.Millisecond * 100)
				_sound = noSound
			}() //doing this so that we can retrigger another sound of the same right after

		case pwmMessage := <-g.PWMControl:
			log.Debugf("PWM request for %d:%d", pwmMessage.portID, pwmMessage.value)
		case <-time.After(time.Millisecond * 100):
			if endLoop {
				log.Debugln("gpioSubscriber has ended")
				break
			}
		}
	}
}

func initGPIO() error {
	if !rpi.Present() {
		return errors.New("not running on raspberry pi")
	}

	if _, err := host.Init(); err != nil {
		log.Errorln("gpioAccess:Could not initialize periph gpio")
		return err
	}

	initPorts()

	return nil
}

func initPorts() {
	rpi.P1_13.Out(gpio.Low)
	rpi.P1_15.Out(gpio.Low)
	rpi.P1_16.Out(gpio.Low)
}

func clearDisplays() {
	for i := 0; i < len(_disp); i++ {
		blankDisplay(i)
	}
}

//dspOut, sends all of the bytes out for controlling the displays
//Data needs to be first followed by clock followed by digits
//MSB needs to be sent first as well
func dspOut(digits byte, clock byte, dspData byte, sndData byte) {
	thirdReg := sndData<<4 | dspData&0x0f //no need to mask the dsp data really, but just in case
	shiftOut(thirdReg)
	shiftOut(clock)
	shiftOut(digits)
	pulse(pinLatchClk) //latch output of shift registers
}

//shifOut sends value "val" passed in to the '595 and latches the output
func shiftOut(val byte) {

	var a byte

	a = 0x80 //msb first

	for b := 1; b <= 8; b++ {
		if val&a > 0 {
			rpi.P1_13.Out(gpio.High)
		} else {
			rpi.P1_13.Out(gpio.Low)
		}

		pulse(pinDataClk) //pulse clock line
		a >>= 1
	}
}

func pulse(pin int) {
	switch pin {
	case pinDataClk:
		rpi.P1_15.Out(gpio.High)
	case pinLatchClk:
		rpi.P1_16.Out(gpio.High)
	}

	//delay here
	//time.Sleep(1 * time.Microsecond) //this should be enough for a HC595 I think?

	switch pin {
	case pinDataClk:
		rpi.P1_15.Out(gpio.Low)
	case pinLatchClk:
		rpi.P1_16.Out(gpio.Low)
	}
}

func setDisplay(dispNum int, digits []byte) {
	for i, d := range digits {
		_disp[dispNum][i] = d
	}
}

func blankDisplay(dispNum int) {
	_disp[dispNum] = [...]byte{blank, blank, blank, blank, blank, blank, blank} //initialize to blank disp
}

func numToArray(number int32) ([]byte, error) {
	var scoreArr []byte

	var remainder int32
	tmpScore := number

	for {
		remainder = tmpScore % 10
		scoreArr = append(scoreArr, byte(remainder))
		tmpScore /= 10

		if tmpScore == 0 {
			break
		}
	}

	return scoreArr, nil
}

//assumption is 7 digit display, so we will blank all remaining digits the score passed in didn't set
func setScore(dispNum int, score int32) {
	scoreArr, _ := numToArray(score)

	_disp[dispNum] = [...]byte{blank, blank, blank, blank, blank, blank, blank} //initialize to blank disp

	if score != blankScore {
		//copy the score into the display array
		for i, num := range scoreArr {
			_disp[dispNum][i] = num
		}
	}
}

//pretty sure match and ball in play are the same display (digits 4 and 3), Credit is 0 and 6
func setBallInPlay(ball int8) {
	ballDisp := _disp[creditMatchDisp][3:5]
	if ball == blankScore {
		ballDisp[0] = blank
		ballDisp[1] = blank
		return
	}

	ballArr, _ := numToArray(int32(ball))

	if len(ballArr) == 2 {
		ballDisp[0] = ballArr[0]
		ballDisp[1] = ballArr[1]
	} else {
		ballDisp[0] = ballArr[0]
		ballDisp[1] = blank
	}
}

//for some reason GamePlan uses digit 6 and 0
func setCredits(credit int8) {

	if credit == blankScore {
		_disp[creditMatchDisp][creditMSD] = blank
		_disp[creditMatchDisp][creditLSD] = blank
		return
	}

	creditArr, _ := numToArray(int32(credit))

	if len(creditArr) == 2 {
		_disp[creditMatchDisp][creditLSD] = creditArr[0]
		_disp[creditMatchDisp][creditMSD] = creditArr[1]
	} else {
		_disp[creditMatchDisp][creditLSD] = creditArr[0]
		_disp[creditMatchDisp][creditMSD] = blank
	}
}

func runGPIO() {
	var digit, display, data, digitStrobe byte

	for {
		digitStrobe = 0x01
		//data = 0x06

		for digit = 0; digit < 7; digit++ {
			var clkOut byte = 0x01
			for display = 0; display < 4; display++ {
				data = _disp[display][digit]

				dspOut(0, clkOut, data, _sound)
				clkOut <<= 1
			}

			//strobing the digit here, which is why we took it out of the other for loop
			data = _disp[creditMatchDisp][digit]
			dspOut(digitStrobe, clkOut, data, _sound)
			digitStrobe <<= 1                  //shifting over for the next digit
			time.Sleep(100 * time.Microsecond) //230 ms should be 120hz to the displays?
		}

		if endLoop {
			break
		}

		//loop forever
	}
}

func (g *GoFlip) SetDisplay(display int, value int32) {
	var msg displayMessage
	msg.display = display
	msg.value = value

	g.DisplayControl <- msg
}

func (g *GoFlip) ShowDisplay(display int, on bool) {
	if on {
		g.SetDisplay(display, 0)
		log.Debugf("ShowDisplay called setting disp on for %d\n", display)
	} else {
		g.SetDisplay(display, blankScore)
	}
}

func (g *GoFlip) SetCreditDisp(value int8) {
	g.SetDisplay(creditDisp, int32(value))
}

func (g *GoFlip) SetBallInPlayDisp(value int8) {
	g.SetDisplay(ballInPlayDisp, int32(value))
}

func (g *GoFlip) PlaySound(soundID byte) {
	var msg soundMessage
	msg.soundID = soundID

	g.SoundControl <- msg

}

func (g *GoFlip) DebugOutDisplays() {
	for i, val := range _disp {
		log.Debugf("Display Array %d: ", i)
		log.Debugln(val)
	}
}
