package goflip

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

/*
gameEvents has handlers for the game in play.

Player Control events:
GameStart = called when a credit is added to the machine (someone presses the credit button)
AddPlayer = adds to the number of players in the game
PlayerUp = called every time that a new player is up
PlayerEnd = called every time a player loses the ball in play
PlayerFinish = called at the very end of the game for that player
GameOver = called at the very end of the game

BallInPlay events:
BallDrained = called when a ball is now found in the outhole
BallInPlay = called when a ball is launched
*/

// GameStart is called when a game is started (when the first player gets a credit)
func GameStart() {
	log.Debugln("gameEvents:GameStart()")

	g := GetMachine()

	if g.TestMode {
		return
	}

	//Reset game stats
	g.BallInPlay = 0 //nextup will queue this up
	g.NumOfPlayers = 0
	g.CurrentPlayer = 0
	ClearScores()

	for _, f := range g.Observers {
		f.GameStart()
	}

}

// GameOver should be called when it is game over.
func GameOver() {
	g := GetMachine()
	SetBallInPlayDisp(blankScore)

	log.Debugln("gameEvents:GameOver()")

	if g.TestMode {
		return
	}

	g.BallInPlay = 0

	for _, f := range g.Observers {
		f.GameOver()
	}

	ChangePlayerState(NoPlayer)
}

func BallDrained() {
	log.Debugln("BallDrained() called")

	g := GetMachine()

	if g.TestMode {
		return
	}

	for _, f := range g.Observers {
		f.BallDrained()
	}
}

func PlayerFinish() {
	log.Debugln("PlayerFinish() called")

	g := GetMachine()

	if g.TestMode {
		return
	}

	for _, f := range g.Observers {
		f.PlayerFinish(g.CurrentPlayer)
	}
}

func PlayerEnd() {
	g := GetMachine()
	if g.TestMode {
		return
	}

	var wait sync.WaitGroup
	wait.Add(len(g.Observers))

	for _, f := range g.Observers {
		f.PlayerEnd(g.CurrentPlayer, &wait)
	}

	go func() {
		wait.Wait() //need to wait for all observers to be done with any goroutines.
		ChangePlayerState(UpPlayer)
	}()
}

func PlayerUp() {
	g := GetMachine()

	if g.gameState != InProgress {
		log.Warnln("PlayerUp called, but game is not started")
		return
	}

	log.Debugln("PlayerUp() called")

	if g.TestMode {
		return
	}

	g.BallScore = 0 //reset before any points are added

	if g.BallInPlay == 0 {
		//first time we are playing
		//g.BallInPlay = 1
		g.CurrentPlayer = 1
	}

	if g.CurrentPlayer < g.NumOfPlayers {
		g.CurrentPlayer++ //we are staying on the same ball
	} else {
		//next ball
		if g.BallInPlay < g.TotalBalls {
			g.BallInPlay++
			g.CurrentPlayer = 1
		} else {
			//game over
			ChangeGameState(GameEnded)
			return
		}
	}

	SetBallInPlayDisp(int8(g.BallInPlay))

	if g.BallInPlay == 1 {
		for _, f := range g.Observers {
			f.PlayerStart(g.CurrentPlayer)
		}
	}

	for _, f := range g.Observers {
		f.PlayerUp(g.CurrentPlayer)
	}
}

func AddPlayer() {
	log.Debugln("AddPlayer() called")
	g := GetMachine()

	if g.TestMode {
		return
	}

	//sanity check, only can add a player if on ball 1
	if g.BallInPlay > 1 {
		return
	}

	if g.NumOfPlayers < g.MaxPlayers {
		g.NumOfPlayers++

		ShowDisplay(g.NumOfPlayers, true)
		log.Debugf("ShowDisplay was called passing: %d, true", g.NumOfPlayers)

		for _, f := range g.Observers {
			f.PlayerAdded(g.NumOfPlayers)
		}
	}
}
