package goflip

import (
	"errors"
	"time"

	"github.com/googolgl/go-i2c"
	"github.com/googolgl/go-pca9685"
	log "github.com/sirupsen/logrus"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/rpi"
)

var _disp [5][7]byte //this holds what we want to show on the display. Bytes are in terms of what the 74ls48 supports (0x0f is blank)
var _sound byte

var _i2cServo *i2c.Options
var _pca0 *pca9685.PCA9685
var _servo0 *pca9685.Servo

var _i2cDisplay *i2c.Options
var _dsp *I2CDisplay

const (
	blankScore      = -1 //the numeric number that can be passed in as an integer to clear the display
	creditMatchDisp = 0  //the number in the display array for the credit display
	creditLSD       = 6  //position in the display array for the 1's credit disp digit
	creditMSD       = 5  //position in the display array for the 10's credit disp digit

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
	angle int
}

var endLoop bool

func gpioInit() {
	//clearDisplays()

	endLoop = false
	_sound = noSound
	//go runGPIO()
}

func gpioSubscriber() {
	err := initGPIO()
	if err != nil {
		return
	}

	log.Debugln("Starting gpio subscribing")

subscriberloop:
	for {
		//g := GetMachine()
		select {
		case dspMsg := <-displayControl:
			if dspMsg.display > 0 && dspMsg.display <= 4 {
				_dsp.SetDisplay(int8(dspMsg.display), dspMsg.value)
			} else {
				switch dspMsg.display {
				case ballInPlayDisp:
					//TODO fix this for the i2c support
					_dsp.SetBallInPlay(int8(dspMsg.value))
				case creditDisp:
					_dsp.SetCredits(int8(dspMsg.value))
				}
			}
		case sndMsg := <-soundControl:
			go func() {
				//_sound = sndMsg.soundID
				stageSound(sndMsg.soundID)
				time.Sleep(time.Millisecond * 100)
				//_sound = noSound //???
				stageSound(noSound)

			}() //doing this so that we can retrigger another sound of the same right after

		case pwmMessage := <-pWMControl:
			//	log.Debugf("PWM angle is %v", pwmMessage.angle)
			go func() {
				_servo0.Angle(pwmMessage.angle)
			}()
		case <-time.After(time.Millisecond * 300):
			if endLoop {
				log.Debugln("gpioSubscriber has ended")
				break subscriberloop
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
	initI2C()

	return nil
}

func initI2C() {
	// Create new connection to i2c-bus on 1 line with address 0x40.
	// Use i2cdetect utility to find device address over the i2c-bus
	g := GetMachine()

	var err error

	_i2cServo, err = i2c.New(pca9685.Address, g.PWMPortConfig.DeviceAddress)
	if err != nil {
		log.Fatalf("initI2C(). DeviceAddress passed is: %v, error: %v", g.PWMPortConfig.DeviceAddress, err)
	}

	_i2cDisplay, err = i2c.New(0x11, g.PWMPortConfig.DeviceAddress)
	if err != nil {
		log.Fatalf("init failed for i2cDisplay. DeviceAddress passed is: %v, error: %v", g.PWMPortConfig.DeviceAddress, err)
	}

	o := pca9685.ServOptions{
		AcRange:  g.PWMPortConfig.ArcRange, //180
		MinPulse: g.PWMPortConfig.PulseMin, //500,
		MaxPulse: g.PWMPortConfig.PulseMax, //2500,
	}

	_pca0, err = pca9685.New(_i2cServo, nil)
	if err != nil {
		log.Fatalf("initPWM()2: %v", err)
	}

	// Sets a single PWM channel 0
	_pca0.SetChannel(0, 0, 180)

	// Servo on channel 0
	_servo0 = _pca0.ServoNew(0, &o)

	_dsp, err = NewDisplay(_i2cDisplay)
	if err != nil {
		log.Fatalf("initPWM/NewDisplay failed: %v", err)
	}
}

func initPorts() {
	//Init Ports
	rpi.P1_13.Out(gpio.Low) //Data
	rpi.P1_15.Out(gpio.Low) //Clock
	rpi.P1_16.Out(gpio.Low) //Latch

}

/*
// dspOut, sends all of the bytes out for controlling the displays
// Data needs to be first followed by clock followed by digits
// MSB needs to be sent first as well
func dspOut(digits byte, clock byte, dspData byte, sndData byte) {
	var bank0, thirdReg byte
	thirdReg = (sndData << 4) | (dspData & 0x0f) //no need to mask the dsp data really, but just in case
	bank0 = (^digits) & 0x7f                     //send the inverse now, since we have a 7416 inverter in place, also reset the enable

	rpi.P1_15.Out(gpio.Low) //clock low
	rpi.P1_16.Out(gpio.Low) //latch low
	//rpi.P1_15.Out(gpio.High) //clock high

	shiftOut(thirdReg)
	shiftOut(0)    //always shift out zero first on the clocks
	shiftOut(0x7f) //keeping the digit off for now and enable high so that we can latch in new data

	rpi.P1_15.Out(gpio.Low)  //clock low
	rpi.P1_16.Out(gpio.High) //latch High
	//rpi.P1_15.Out(gpio.High) //clock high

	//pulse(pinLatchClk) //latch output of shift registers

	time.Sleep(time.Nanosecond * 25) //time to setup 74ls379
	//-----------------
	rpi.P1_15.Out(gpio.Low) //clock low
	rpi.P1_16.Out(gpio.Low) //latch low
	//rpi.P1_15.Out(gpio.High) //clock high

	shiftOut(thirdReg)
	shiftOut(clock) //then shift out the actual clock
	shiftOut(bank0) //keep the enable low, and now have the digit turned on
	//pulse(pinLatchClk) //latch output of shift registers

	rpi.P1_15.Out(gpio.Low)  //clock low
	rpi.P1_16.Out(gpio.High) //latch High
	//rpi.P1_15.Out(gpio.High) //clock high

	time.Sleep(time.Microsecond * 10) //time to setup 74ls379 + to see the digit
}*/

// shifOut sends value "val" passed in to the '595 and latches the output
func shiftOut(val byte) {

	var a byte

	a = 0x80 //msb first

	for b := 1; b <= 8; b++ {
		rpi.P1_15.Out(gpio.Low)
		if val&a > 0 {
			rpi.P1_13.Out(gpio.High)
		} else {
			rpi.P1_13.Out(gpio.Low)
		}
		rpi.P1_15.Out(gpio.High)
		time.Sleep(time.Microsecond * 2) //time for 595
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

/*
func setDisplay(dispNum int, digits []byte) {
	for i, d := range digits {
		_disp[dispNum][i] = d
	}
}
*/

func stageSound(sndCode byte) {
	log.Debugf("******Writing sound out for %v", sndCode)

	rpi.P1_16.Out(gpio.Low) //latch low
	//rpi.P1_15.Out(gpio.High) //clock high

	shiftOut(sndCode)
	rpi.P1_16.Out(gpio.High)          //latch High
	rpi.P1_15.Out(gpio.Low)           //clock low
	time.Sleep(time.Microsecond * 20) //present sound

	//send out clear sound

}

/*
// TODO replace this...
func runGPIO() {
	var digit, display, data, digitStrobe, clkOut byte

	for {
		digitStrobe = 0x01
		//data = 0x06

		for digit = 0; digit < 7; digit++ {
			clkOut = 0x01
			for display = 0; display <= 3; display++ {
				data = _disp[display][digit]

				dspOut(0, clkOut, data, _sound) //keep digit strobe 0 until we get all display latches loaded
				clkOut <<= 1
			}

			data = _disp[4][digit]
			clkOut = 0x10 //force to 5 display
			dspOut(0, clkOut, data, _sound)
			dspOut(digitStrobe, clkOut, data, _sound)
			digitStrobe <<= 1

			time.Sleep(200 * time.Microsecond) //230 ms should be 120hz to the displays?
		}

		if endLoop {
			break
		}

		//loop forever
	}
}*/

func SetDisplay(display int, value int32) {
	var msg displayMessage
	msg.display = display
	msg.value = value

	displayControl <- msg
}

func ShowDisplay(display int, on bool) {
	if on {
		SetDisplay(display, 0)
		log.Debugf("ShowDisplay called setting disp on for %d\n", display)
	} else {
		SetDisplay(display, blankScore)
	}
}

func SetCreditDisp(value int8) {
	SetDisplay(creditDisp, int32(value))
}

func SetBallInPlayDisp(value int8) {
	SetDisplay(ballInPlayDisp, int32(value))
}

func PlaySound(soundID byte) {
	var msg soundMessage
	msg.soundID = soundID

	soundControl <- msg

}

func DebugOutDisplays() {
	for i, val := range _disp {
		log.Debugf("Display Array %d: ", i)
		log.Debugln(val)
	}
}

func ServoAngle(angle int) {
	var msg pwmMessage
	msg.angle = angle
	pWMControl <- msg
}
