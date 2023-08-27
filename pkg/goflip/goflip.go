package goflip

//to build: env GOOS=linux GOARCH=arm GOARM=5 go build
import (
	"encoding/json"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Moving methods that should really be internal
var lampControl chan deviceMessage
var solenoidControl chan deviceMessage
var displayControl chan displayMessage
var soundControl chan soundMessage
var pWMControl chan pwmMessage

type GoFlip struct {
	devices        arduinos
	scores         [4]int32
	BallInPlay     int //If no ball, then 0. //used
	ExtraBall      bool
	TotalBalls     int       //used
	Credits        int       //used
	MaxPlayers     int       //max players supported by the game //used
	NumOfPlayers   int       //number of players playing
	PWMPortConfig  PWMConfig //used
	switchStates   []bool
	lampStates     map[int]int
	Observers      []Observer //used
	CurrentPlayer  int        //used
	observerEvents chan SwitchEvent
	//GameRunning      bool  //Whether a game is going on = true, or game is over = false
	BallScore        int32    //current score for the ball in play
	TestMode         bool     //states whether we are in Test Mode or not //used
	DiagObserver     Observer //used
	playerEndChannel chan bool
	gameState        GState
	playerState      PState
	Quitting         bool //Notifies all go routines that the running application is quitting //used
	ConsoleMode      bool //Signifies that goFlip is being used for running in a console vs an actual machine //used
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

// PWMConfig holds the configuration for the gpio PWM port to be used to control a servo
type PWMConfig struct {
	ArcRange      int
	PulseMin      float32
	PulseMax      float32
	DeviceAddress string
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
	UpPlayer
	EndPlayer
	FinishedPlayer
)

const QUIT = -1 //ID passed to channels to let goRoutines to quit

type GState int

const (
	Init GState = iota
	InProgress
	GameEnded
)

type deviceMessage struct {
	id    int
	value int //set to one of the constants
}

// Init is Called just one time in the beginning to Initialize the game
func (g *GoFlip) Init(m func(SwitchEvent)) bool {

	go StartServer()
	g.gameState = Init
	g.playerState = NoPlayer

	log.AddHook(MsgHook{})
	g.playerEndChannel = make(chan bool)

	lampControl = make(chan deviceMessage)
	solenoidControl = make(chan deviceMessage)
	g.observerEvents = make(chan SwitchEvent, 100)

	displayControl = make(chan displayMessage)
	soundControl = make(chan soundMessage)
	pWMControl = make(chan pwmMessage)

	//These can be overriddden after Init, before Start is called
	g.MaxPlayers = 4
	g.TotalBalls = 3
	g.BallScore = 0
	g.TestMode = false
	g.switchStates = make([]bool, 64)
	log.Println("!!!Setting LampStates!!")
	g.lampStates = make(map[int]int)

	gpioInit()

	//moved this before subbscribers try to connect and write
	if !g.ConsoleMode {
		connected := false
		for i := 0; i < 5; i++ {
			if !g.devices.Connect() {
				log.Warningf("Devices were unable to connect, Try %d\n", i)
			} else {
				connected = true
				break
			}

			//try to reconnect every second
			time.Sleep(1 * time.Second)
		}

		if !connected {
			log.Errorln("Devices were unable to connect. Check USB connections")
			return false
		}
	}

	log.Println("Starting LampSubscriber()")

	go LampSubscriber()
	go SolenoidSubscriber()
	go gpioSubscriber()

	for _, f := range g.Observers {
		f.Init()
	}

	//This broadcasts anything to all Observers.
	go func() {
		for {
			select {
			case sw := <-g.observerEvents:
				g.DiagObserver.SwitchHandler(sw)

				if !g.TestMode {
					//call individual feature Switch Handling too.
					for _, f := range g.Observers {
						f.SwitchHandler(sw)
					}
				}

				if sw.SwitchID == QUIT {
					return
				}

			}
		}
	}()

	//handler for calling switch event routine:

	go func() {
		log.Debugln("Starting switch monitoring")
		g.devices.switchMatrix.consoleMode = g.ConsoleMode
		for {
			//buf := make([]byte, 16) //shouldn't be over 1 byte really
			buf := g.devices.switchMatrix.ReadSwitch()
			log.Debugf("Received %d switch events", len(buf))

			//we should never receive 0 switch events... so if we do, maybe we stop and reinitialize??

			for _, sw := range buf {
				g.switchStates[sw.SwitchID] = sw.Pressed
				m(sw) //main switch eventHandler called

				g.observerEvents <- sw
			}
		}

	}()

	return true
}

func BroadcastEvent(sw SwitchEvent) {
	g := GetMachine()
	g.observerEvents <- sw
}

// IsGameInPlay returns true if a game is going on. False if not.
func IsGameInPlay() bool {
	g := GetMachine()
	return g.BallInPlay > 0
}

func AddScore(points int) {
	g := GetMachine()
	if g.CurrentPlayer < 1 {
		return
	}
	g.scores[g.CurrentPlayer-1] += int32(points)
	g.BallScore += int32(points)
	log.Debugf("goFlip:BallScore = %d, total = %d\n", g.BallScore, g.scores[g.CurrentPlayer-1])

	//refresh display
	SetDisplay(g.CurrentPlayer, g.scores[g.CurrentPlayer-1])
}

func ClearScores() {
	g := GetMachine()
	for i := range g.scores {
		g.scores[i] = 0
		ShowDisplay(i+1, false)
	}
}

func PlayerScore(playerNumber int) int32 {
	g := GetMachine()
	return g.scores[playerNumber-1]
}

func SendStats() {
	g := GetMachine()
	stat := GameStats{}
	stat.BallInPlay = g.BallInPlay
	stat.Credits = 0
	stat.Display1 = 0
	stat.Display2 = 0
	stat.Display3 = 0
	stat.Display4 = 0
	stat.Match = 0
	stat.TotalBalls = g.TotalBalls
	i := len(g.scores)

	if i > 0 {
		stat.Player1Score = g.scores[0]
		stat.Player2Score = g.scores[1]
		stat.Player3Score = g.scores[2]
		stat.Player4Score = g.scores[3]
	}
	statb, err := json.Marshal(stat)

	if err != nil {
		log.Errorln("Error in marshalling:", err)
		return
	}
	//log.Debugln("Sending json:", string(statb))
	Broadcast("stat", string(statb))

}

func GetPlayerState() PState {
	g := GetMachine()
	return g.playerState
}

func ChangePlayerState(newState PState) bool {
	g := GetMachine()
	if g.playerState == newState {
		//already at this state, so don't change
		return false
	}

	g.playerState = newState
	switch g.playerState {
	case UpPlayer:
		PlayerUp()
	case EndPlayer:
		PlayerEnd()
	case FinishedPlayer:
		PlayerFinish()
	}
	return true

}

func GetGameState() GState {
	g := GetMachine()
	return g.gameState
}

func ChangeGameState(newState GState) bool {
	g := GetMachine()
	if g.gameState == newState {
		//already in the current state
		return false
	}

	g.gameState = newState

	switch g.gameState {
	case GameEnded:
		GameOver()
	case InProgress:
		GameStart()
	}

	return true
}

// Quit tells all channels and go routines that the application is ending, to attempt to
// nicely disconnect to all peripherals, etc.
func Quit() {
	g := GetMachine()
	g.Quitting = true

	var msg deviceMessage
	msg.id = QUIT
	msg.value = 0

	lampControl <- msg
	solenoidControl <- msg
	BroadcastEvent(SwitchEvent{SwitchID: QUIT, Pressed: true})
}

var machineInstance *GoFlip
var lock sync.Mutex

func GetMachine() *GoFlip {
	lock.Lock()

	defer lock.Unlock()

	if machineInstance == nil {
		machineInstance = new(GoFlip)
	}

	return machineInstance
}
