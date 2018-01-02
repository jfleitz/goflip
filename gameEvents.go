package goflip

import log "github.com/Sirupsen/logrus"

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
	//Reset game stats
	g.BallInPlay = 0 //nextup will queue this up
	g.NumOfPlayers = 0
	g.CurrentPlayer = 0

	g.GameRunning = true

	for _, f := range g.Observers {
		f.GameStart()
	}

}

//GameOver should be called when it is game over.
func (g *GoFlip) GameOver() {
	log.Infoln("gameEvents:GameOver()")

	g.BallInPlay = 0

	for _, f := range g.Observers {
		f.GameOver()
	}

	g.GameRunning = false
}

func (g *GoFlip) BallDrained() {
	log.Infoln("BallDrained() called")

	for _, f := range g.Observers {
		f.BallDrained()
	}
}

func (g *GoFlip) PlayerFinish() {
	log.Infoln("PlayerFinish() called")

	for _, f := range g.Observers {
		f.PlayerFinish(g.CurrentPlayer)
	}
}

func (g *GoFlip) PlayerEnd() {
	log.Infoln("PlayerEnd() called")

	for _, f := range g.Observers {
		f.PlayerEnd(g.CurrentPlayer)
	}

	g.PlayerUp() //call for the next ball or player if there is one
}

func (g *GoFlip) PlayerUp() {
	log.Infoln("PlayerUp() called")
	g.BallScore = 0 //reset before any points are added

	defer func() {
		if g.GameRunning {
			for _, f := range g.Observers {
				f.PlayerUp(g.CurrentPlayer)
			}
			log.Infoln("GoFlip: PlayerUp. IsGameInPlay. BallInPlay, CurrentPlayer:", g.BallInPlay, g.CurrentPlayer)
		} else {
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
			g.SetBallInPlayDisp(0)
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
	//sanity check, only can add a player if on ball 1
	if g.BallInPlay > 1 {
		return
	}

	if g.NumOfPlayers < g.MaxPlayers {
		g.NumOfPlayers++
		g.ShowDisplay(g.NumOfPlayers, true)
	}
}
