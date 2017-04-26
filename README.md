# goflip
goflip is a Go/golang package for controlling a pinball machine. It's purpose is to run the game rules as well as communicate to the peripheral arduino devices.

## Installing

`go get github.com/jfleitz/goflip`

Check out the Mike Bossy pinball project on how to use:
http://www.github.com/jfleitz/Bossy

## Required
goflip utilizes the serial ports to communicate with the arduino peripherals. The following are required to be connected when using goflip:
* 8x8 Switch Matrix (Arduino nano)
* Lamp Driver Unit (LDU - Arduino nano)
* Solenoid Driver Unit (SDU - Arduino nano)


### Future
* Display driver support
* LDU - short message support (currenly 4 byte messages)
* Remote monitoring (web interface)
