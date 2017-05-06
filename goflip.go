package goflip

import (
	log "github.com/Sirupsen/logrus"
)

type GoFlip struct {
	devices         arduinos
	Scores          []int
	BallInPlay      int //If no ball, then 0.
	ExtraBall       bool
	TotalBalls      int
	MaxPlayers      int
	LampControl     chan deviceMessage
	SolenoidControl chan deviceMessage
	SwitchEvents    chan SwitchEvent
	switchStates    []bool
	lampStates      map[int]int
	Observers       []Observer
	CurrentPlayer   int
	ObserverEvents  chan SwitchEvent
}

type Observer interface {
	Init()                     //Called from the beginning when the game is first turned on
	GameStart()                //Called when a game starts
	PlayerAdded(playerID int)  //Called when a player is added to the current game
	PlayerUp(int)              //called when a new player is up (passing the player number in as well.. zero based)
	PlayerEnd(int)             //called when a player ends. Same as BallDrained, unless there is a ball save
	SwitchHandler(SwitchEvent) //called every time a switch event occurs
	BallDrained()              //calls when a ball is drained
	GameOver()                 //called when a game is over
}

type SwitchEvent struct {
	SwitchID int
	Pressed  bool
}

const (
	Off       = 0    //can be used for Solenoids or Lamp
	On        = 1    //can be used for Solenoids or Lamp
	SlowBlink = 2    //lamp Only
	FastBlink = 3    //lamp Only
	ack       = iota //when used, it doesn't matter what ID is.
)

const consoleMode bool = false

type deviceMessage struct {
	id    int
	value int //set to one of the constants
}

//Init is Called just one time in the beginning to Initialize the game
func (g *GoFlip) Init(m func(SwitchEvent)) bool {

	if !consoleMode {
		if !g.devices.Connect() {
			log.Errorln("Devices were unable to connect. Check USB connections")
			//return false make this configurable.
		}
	}

	g.LampControl = make(chan deviceMessage, 100)
	g.SolenoidControl = make(chan deviceMessage, 100)
	g.SwitchEvents = make(chan SwitchEvent, 100)
	g.ObserverEvents = make(chan SwitchEvent, 100)

	//These can be overriddden after Init, before Start is called
	g.MaxPlayers = 2
	g.TotalBalls = 3
	g.switchStates = make([]bool, 64)
	g.lampStates = make(map[int]int)

	for _, f := range g.Observers {
		f.Init()
	}

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

				g.ObserverEvents <- sw
			}
		}

	}()

	//This broadcasts anything to all Observers.
	go func() {
		for {
			select {
			case sw := <-g.ObserverEvents:
				//call individual feature Switch Handling too.
				for _, f := range g.Observers {
					f.SwitchHandler(sw)
				}
			}
		}
	}()

	return true
}

func (g *GoFlip) BroadcastEvent(sw SwitchEvent) {
	g.ObserverEvents <- sw
}

//IsGameInPlay returns true if a game is going on. False if not.
func (g *GoFlip) IsGameInPlay() bool {
	return g.BallInPlay > 0
}

func (g *GoFlip) AddScore(points int) {
	g.Scores[g.CurrentPlayer] += points
}
