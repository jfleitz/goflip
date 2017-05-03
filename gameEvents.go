package goflip

import log "github.com/Sirupsen/logrus"

//GameStart is called when a game is started (when the first player gets a credit)
func (g *GoFlip) GameStart() {
	log.Infoln("GameStart() called")
	g.BallInPlay = 0 //nextup will queue this up
	g.CurrentPlayer = 0

	for _, f := range g.Observers {
		f.GameStart()
	}

}

//GameOver should be called when it is game over.
func (g *GoFlip) GameOver() {
	log.Infoln("GameOver() called")

	g.BallInPlay = 0

	for _, f := range g.Observers {
		f.GameOver()
	}
}

func (g *GoFlip) BallDrained() {
	log.Infoln("BallDrained() called")

	for _, f := range g.Observers {
		f.BallDrained()
	}
}

func (g *GoFlip) PlayerEnd() {
	log.Infoln("PlayerEnd() called")

	for _, f := range g.Observers {
		f.PlayerEnd(g.CurrentPlayer)
	}
}

func (g *GoFlip) NextUp() {
	log.Infoln("NextUp() called")

	defer func() {
		if g.IsGameInPlay() {
			for _, f := range g.Observers {
				f.PlayerUp(g.CurrentPlayer)
			}
			log.Infoln("GoFlip: NextUp. IsGameInPlay. BallInPlay, CurrentPlayer:", g.BallInPlay, g.CurrentPlayer)
		} else {
			log.Infoln("GoFlip: NextUp. IsGameInPlay = false")
		}

	}()

	if g.BallInPlay == 0 {
		g.BallInPlay = 1
		g.CurrentPlayer = 1
		return
	}

	if g.CurrentPlayer < g.MaxPlayers {
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

}
