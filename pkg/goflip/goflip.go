package goflip

//to build: env GOOS=linux GOARCH=arm GOARM=5 go build
import (
	"encoding/json"
	"sync"

	log "github.com/sirupsen/logrus"
)

type GoFlip struct {
	devices         arduinos
	Scores          [4]int32
	BallInPlay      int //If no ball, then 0.
	ExtraBall       bool
	TotalBalls      int
	Credits         int
	MaxPlayers      int //max players supported by the game
	NumOfPlayers    int //number of players playing
	LampControl     chan deviceMessage
	SolenoidControl chan deviceMessage
	SwitchEvents    chan SwitchEvent
	DisplayControl  chan displayMessage
	SoundControl    chan soundMessage
	PWMControl      chan pwmMessage
	switchStates    []bool
	lampStates      map[int]int
	Observers       []Observer
	CurrentPlayer   int
	ObserverEvents  chan SwitchEvent
	//GameRunning      bool  //Whether a game is going on = true, or game is over = false
	BallScore        int32 //current score for the ball in play
	TestMode         bool  //states whether we are in Test Mode or not
	DiagObserver     Observer
	PlayerEndChannel chan bool
	gameState        GState
	playerState      PState
}

type Observer interface {
	Init()                          //Called from the beginning when the game is first turned on
	GameStart()                     //Called when a game starts
	PlayerAdded(playerID int)       //Called when a player is added to the current game
	PlayerStart(int)                //Called the very first time a player is playing (their first Ball1)
	PlayerUp(int)                   //called when a new player is up (passing the player number in as well.. zero based)
	PlayerEnd(int, *sync.WaitGroup) //called after every ball is ended for the player (after ball drain)
	PlayerFinish(int)               //called after the very last ball for the player is over (after ball 3 for example)
	SwitchHandler(SwitchEvent)      //called every time a switch event occurs
	BallDrained()                   //calls when a ball is drained
	GameOver()                      //called when a game is over
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

type PState int

const (
	NoPlayer PState = iota
	PlayerUp
	PlayerEnd
	PlayerFinish
)

const consoleMode bool = false

type GState int

const (
	Init GState = iota
	GameStart
	GameOver
)

type deviceMessage struct {
	id    int
	value int //set to one of the constants
}

//Init is Called just one time in the beginning to Initialize the game
func (g *GoFlip) Init(m func(SwitchEvent)) bool {

	go StartServer()
	g.gameState = Init
	g.playerState = NoPlayer

	log.AddHook(MsgHook{})
	g.PlayerEndChannel = make(chan bool)

	g.LampControl = make(chan deviceMessage)
	g.SolenoidControl = make(chan deviceMessage)
	g.SwitchEvents = make(chan SwitchEvent, 100)
	g.ObserverEvents = make(chan SwitchEvent, 100)

	g.DisplayControl = make(chan displayMessage)
	g.SoundControl = make(chan soundMessage)
	g.PWMControl = make(chan pwmMessage)

	//These can be overriddden after Init, before Start is called
	g.MaxPlayers = 4
	g.TotalBalls = 3
	g.BallScore = 0
	g.TestMode = false
	g.switchStates = make([]bool, 64)
	g.lampStates = make(map[int]int)

	g.gpioInit()

	//moved this before subbscribers try to connect and write
	if !consoleMode {
		if !g.devices.Connect() {
			log.Warningln("RETRY:Devices were unable to connect")
			//go ahead and retry once
			if !g.devices.Connect() {
				log.Errorln("Devices were unable to connect. Check USB connections")
				//return false make this configurable.
				return true
			}
		}
	}

	go g.LampSubscriber()
	go g.SolenoidSubscriber()
	go g.gpioSubscriber()

	for _, f := range g.Observers {
		f.Init()
	}

	//This broadcasts anything to all Observers.
	go func() {
		for {
			select {
			case sw := <-g.ObserverEvents:
				g.DiagObserver.SwitchHandler(sw)

				if !g.TestMode {
					//call individual feature Switch Handling too.
					for _, f := range g.Observers {
						f.SwitchHandler(sw)
					}
				}

			}
		}
	}()

	//handler for calling switch event routine:

	go func() {
		log.Debugln("Starting switch monitoring")
		for {
			//buf := make([]byte, 16) //shouldn't be over 1 byte really
			buf := g.devices.switchMatrix.ReadSwitch()
			log.Debugf("Received %d switch events", len(buf))

			//we should never receive 0 switch events... so if we do, maybe we stop and reinitialize??

			for _, sw := range buf {
				g.switchStates[sw.SwitchID] = sw.Pressed
				m(sw) //main switch eventHandler called

				g.ObserverEvents <- sw
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
	if g.CurrentPlayer < 1 {
		return
	}
	g.Scores[g.CurrentPlayer-1] += int32(points)
	g.BallScore += int32(points)
	log.Debugf("goFlip:BallScore = %d\n", g.BallScore)

	//refresh display
	g.SetDisplay(g.CurrentPlayer, g.PlayerScore(g.CurrentPlayer))
}

func (g *GoFlip) ClearScores() {
	for i := range g.Scores {
		g.Scores[i] = 0
		g.ShowDisplay(i+1, false)
	}
}

func (g *GoFlip) PlayerScore(playerNumber int) int32 {
	return g.Scores[playerNumber-1]
}

func (g *GoFlip) SendStats() {
	stat := GameStats{}
	stat.BallInPlay = g.BallInPlay
	stat.Credits = 0
	stat.Display1 = 0
	stat.Display2 = 0
	stat.Display3 = 0
	stat.Display4 = 0
	stat.Match = 0
	stat.TotalBalls = g.TotalBalls
	i := len(g.Scores)

	if i > 0 {
		stat.Player1Score = g.Scores[0]
		stat.Player2Score = g.Scores[1]
		stat.Player3Score = g.Scores[2]
		stat.Player4Score = g.Scores[3]
	}
	statb, err := json.Marshal(stat)

	if err != nil {
		log.Errorln("Error in marshalling:", err)
		return
	}
	//log.Debugln("Sending json:", string(statb))
	Broadcast("stat", string(statb))

}

func (g *GoFlip) GetPlayerState() PState {
	return g.playerState
}

func (g *GoFlip) ChangePlayerState(newState PState) bool {
	if g.playerState == newState {
		//already at this state, so don't change
		return false
	}

	g.playerState = newState
	switch g.playerState {
	case PlayerUp:
		g.PlayerUp()
	case PlayerEnd:
		g.PlayerEnd()
	case PlayerFinish:
		g.PlayerFinish()
	}
	return true

}

func (g *GoFlip) GetGameState() GState {
	return g.gameState
}

func (g *GoFlip) ChangeGameState(newState GState) bool {
	if g.gameState == newState {
		//already in the current state
		return false
	}

	g.gameState = newState

	switch g.gameState {
	case GameOver:
		g.GameOver()
	case GameStart:
		g.GameStart()
	}

	return true
}
