package goflip

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/googolgl/go-i2c"
)

const (
	// Address default for controller
	Address byte = 0x11
	blank   byte = 0x0f //what is sent to the 7448 on the display board to blank the 7 seg disp
)

// PCA9685 is a Driver for the PCA9685 16-channel 12-bit PWM/Servo controller
type I2CDisplay struct {
	i2c           *i2c.Options
	creditDisplay [7]byte //since this display is actually used for both credits and ball in play.
}

type Message struct {
	DisplayNumber byte
	Digits        [7]byte
}

// New creates the new PCA9685 driver with specified i2c interface and options
func NewDisplay(i2c *i2c.Options) (*I2CDisplay, error) {
	adr := i2c.GetAddr()
	if adr == 0 {
		return nil, fmt.Errorf(`I2C device is not initiated`)
	}

	disp := &I2CDisplay{
		i2c:           i2c,
		creditDisplay: blankDisplay(),
	}

	return disp, nil
}

func (dsp *I2CDisplay) ClearDisplay() error {
	//dsp.i2c.WriteBytes()

	return dsp.SetDisplay(6, 0) //hacky
}

func (dsp *I2CDisplay) SetDisplay(dspNumber int8, value int32) error {

	val := setScore(value)

	msg := Message{DisplayNumber: byte(dspNumber), Digits: val}

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg)

	i, err := dsp.i2c.WriteBytes(buf.Bytes())
	fmt.Printf("Wrote %v bytes", i)
	return err
}

func blankDisplay() [7]byte {
	return [7]byte{blank, blank, blank, blank, blank, blank, blank} //initialize to blank disp
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

// setScore assumption is 7 digit display, so we will blank all remaining digits the score passed in didn't set
func setScore(score int32) [7]byte {
	scoreArr, _ := numToArray(score)

	dsp := blankDisplay()

	//copy the score into the display array
	for i, num := range scoreArr {
		dsp[i] = num
	}
	return dsp
}

func (dsp *I2CDisplay) WriteToDisplay(dspNumber int8, digits [7]byte) error {

	msg := Message{DisplayNumber: byte(dspNumber), Digits: digits}

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg)

	i, err := dsp.i2c.WriteBytes(buf.Bytes())
	fmt.Printf("Wrote %v bytes", i)
	return err
}

// pretty sure match and ball in play are the same display (digits 4 and 3), Credit is 0 and 6
func (dsp *I2CDisplay) SetBallInPlay(ball int8) {
	if ball == blankScore {
		dsp.creditDisplay[4] = blank
		dsp.creditDisplay[5] = blank
		return
	}

	ballArr, _ := numToArray(int32(ball))

	if len(ballArr) == 2 {
		dsp.creditDisplay[4] = ballArr[0]
		dsp.creditDisplay[5] = ballArr[1]
	} else {
		dsp.creditDisplay[4] = ballArr[0]
		dsp.creditDisplay[5] = blank
	}

	dsp.WriteToDisplay(0, dsp.creditDisplay)
}

func (dsp *I2CDisplay) SetCredits(credit int8) {
	//dsp.creditDisplay[0:2]

	if credit == blankScore {
		dsp.creditDisplay[0] = blank
		dsp.creditDisplay[1] = blank
		return
	}

	creditArr, _ := numToArray(int32(credit))

	if len(creditArr) == 2 {
		dsp.creditDisplay[0] = creditArr[0]
		dsp.creditDisplay[1] = creditArr[1]
	} else {
		dsp.creditDisplay[0] = creditArr[0]
		dsp.creditDisplay[1] = blank
	}

	dsp.WriteToDisplay(0, dsp.creditDisplay)
}
