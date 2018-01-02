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

## Events
### Player Control events:
* GameStart = called when a credit is added to the machine (someone * presses the credit button)
* AddPlayer = adds to the number of players in the game
* PlayerUp = called every time that a new player is up
* PlayerEnd = called every time a player loses the ball in play
* PlayerFinish = called at the very end of the game for that player
* GameOver = called at the very end of the game

### BallInPlay events:
* BallDrained = called when a ball is now found in the outhole
* BallInPlay = called when a ball is launched

### Typical playout of the events
A player walks up and puts a coin in the machine (or freeplay and they don't)
* Credit button is pressed
    * GameStart event is callled
    * AddPlayer event is called (since credit button was pressed)
    * PlayerUp event is called (since we have player 1 in the game now)
* Another credit button is pressed and on Ball 1 still
    * AddPlayer event is called to add another player

Player X starts playing and launches the ball
* BallInPlay event is called

Player X loses the ball (and no ball save or after ball save)
* BallDrained is called
    * PlayerEnd is callled
        * If last ball was played for that player, then PlayerFinish is called
        * If more players or not the last ball, PlayerUp is called with next player
        * If no more balls left, then GameOver is called


### Future
* Display driver support
* LDU - short message support (currenly 4 byte messages)
* Remote monitoring (web interface)
