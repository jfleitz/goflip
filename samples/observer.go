/*This is an example of an observer class that is used for
receiving calls from the goflip engine. Each function is required to be
defined so that it implements the goflip.observer interface.

The goflip engine does not encapsulate these calls into a go routine to give
more control and flexibility to the developer. Keep this in mind as these
routines are time sensitive, with the most important being the SwitchHandler
method. The order of the method calls by goflip are in sequence with each Observer
added to the goflip.observer array.

All observers needs to be added to the goflip.Observers array during the main()
main routine. Example:
func main() {
	var o *sampleObserver
	o = new(sampleObserver)
	game.Observers = []goflip.Observer{o}

	..
	..
	..
}

*/

package sample //this will probably be package main in your app

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jfleitz/goflip"
)

type sampleObserver struct {
	//add your variables for the observer here
}

/*the following line should be called to ensure that your observer DOES
implement the goflip.Observer interface:
*/
var _ goflip.Observer = (*sampleObserver)(nil)

/*Init is called by goflip when the application is first started (Init). This
is called only once:
*/
func (p *sampleObserver) Init() {
	/*using logrus package for logging. Best practice to call logging when
	only necessary and not in routines that are called a lot*/
	log.Infoln("sampleObserver:Init called")

}

/*SwitchHandler is called any time a switch event is received by goflip. This
routine must be kept as fast as possible. Make use of go routines when necessary
Any delay in this routine can cause issues with latency
*/
func (p *sampleObserver) SwitchHandler(sw goflip.SwitchEvent) {
}

/*BallDrained is called whenever a ball is drained on the playfield (Before PlayerEnd)*/
func (p *sampleObserver) BallDrained() {

}

/*PlayerUp is called after the ball is launched from the Ball Trough for the next ball up
playerID is the player that is now up*/
func (p *sampleObserver) PlayerUp(playerID int) {

}

/*PlayerEnd is called after BallDrained. In a multiball game, this would be after the last
BallDrained method call*/
func (p *sampleObserver) PlayerEnd(playerID int) {

}

/*PlayerAdded is called after a player is added by the credit button, and after the GameStart event*/
func (p *sampleObserver) PlayerAdded(playerID int) {

}

/*GameOver is called after the last player of the last ball is drained, before the game goes
into the GameOver mode*/
func (p *sampleObserver) GameOver() {

}

/*GameStart is called whenever a new game is started*/
func (p *sampleObserver) GameStart() {

}
