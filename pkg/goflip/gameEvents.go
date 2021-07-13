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

//GameStart is called when a game is started (when the first player gets a credit)
func (g *GoFlip) GameStart() {
	log.Infoln("gameEvents:GameStart()")

	if g.TestMode {
		return
	}

	//Reset game stats
	g.BallInPlay = 0 //nextup will queue this up
	g.NumOfPlayers = 0
	g.CurrentPlayer = 0
	g.ClearScores()

	for _, f := range g.Observers {
		f.GameStart()
	}

}

//GameOver should be called when it is game over.
func (g *GoFlip) GameOver() {
	g.SetBallInPlayDisp(blankScore)

	log.Infoln("gameEvents:GameOver()")

	if g.TestMode {
		return
	}

	g.BallInPlay = 0

	for _, f := range g.Observers {
		f.GameOver()
	}

}

func (g *GoFlip) BallDrained() {
	log.Infoln("BallDrained() called")

	if g.TestMode {
		return
	}

	for _, f := range g.Observers {
		f.BallDrained()
	}
}

func (g *GoFlip) PlayerFinish() {
	log.Infoln("PlayerFinish() called")

	if g.TestMode {
		return
	}

	for _, f := range g.Observers {
		f.PlayerFinish(g.CurrentPlayer)
	}
}

func (g *GoFlip) PlayerEnd() {
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
		g.ChangePlayerState(PlayerUp)
	}()
}

func (g *GoFlip) PlayerUp() {
	log.Infoln("PlayerUp() called")

	if g.TestMode {
		return
	}

	g.BallScore = 0 //reset before any points are added

	defer func() {
		if g.gameState == GameStart {
			//	g.BlinkOnlyOneDisplay(g.CurrentPlayer - 1)
			for _, f := range g.Observers {
				f.PlayerUp(g.CurrentPlayer)
			}
			log.Infoln("GoFlip: PlayerUp. IsGameInPlay. BallInPlay, CurrentPlayer:", g.BallInPlay, g.CurrentPlayer)
		} else {
			//	g.BlinkOnlyOneDisplay(4) //credit display
			//	g.BlinkDisplay(4, false)
			log.Infoln("GoFlip: PlayerUp. IsGameInPlay = false")
		}

	}()

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
			g.GameOver()
			return
		}
	}

	g.SetBallInPlayDisp(int8(g.BallInPlay))

	if g.BallInPlay == 1 {
		for _, f := range g.Observers {
			f.PlayerStart(g.CurrentPlayer)
		}
		return
	}
}

func (g *GoFlip) AddPlayer() {
	log.Infoln("AddPlayer() called")

	if g.TestMode {
		return
	}

	//sanity check, only can add a player if on ball 1
	if g.BallInPlay > 1 {
		return
	}

	if g.NumOfPlayers < g.MaxPlayers {
		g.NumOfPlayers++

		g.ShowDisplay(g.NumOfPlayers, true)
		log.Infof("ShowDisplay was called passing: %d, true", g.NumOfPlayers)

		for _, f := range g.Observers {
			f.PlayerAdded(g.NumOfPlayers)
		}
	}
}
