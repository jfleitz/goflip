package goflip

/*This is the websocket api interface for getting score data */
import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/rsms/gotalk"
	log "github.com/sirupsen/logrus"
)

var (
	connections = make(map[*gotalk.Sock]int)
	mut         sync.RWMutex
)

//GameStats tells the stats of the game in play (what you would see from the backbox mostly)
type GameStats struct {
	Player1Score int32
	Player2Score int32
	Player3Score int32
	Player4Score int32
	Match        int16
	TotalBalls   int
	BallInPlay   int
	Display1     int32
	Display2     int32
	Display3     int32
	Display4     int32
	Credits      int16
}

//GameEvent is passed whenever an internal event occurs (Switch Event for example)
type GameEvent struct {
	EventType string
	EventText string
}

type MsgHook struct {
}

func (h MsgHook) Fire(e *log.Entry) error {
	msg, err := json.Marshal(e)
	if err != nil {
		return err
	}
	go func() {
		Broadcast("msg", string(msg))
	}()
	return nil
}

func (MsgHook) Levels() []log.Level {
	return []log.Level{
		log.PanicLevel,
		log.FatalLevel,
		log.ErrorLevel,
		log.WarnLevel,
		log.InfoLevel,
		log.DebugLevel,
	}
}

func onAccept(s *gotalk.Sock) {
	// Keep track of connected sockets
	mut.Lock()
	defer mut.Unlock()
	connections[s] = 1

	s.CloseHandler = func(s *gotalk.Sock, _ int) {
		mut.Lock()
		defer mut.Unlock()
		delete(connections, s)
	}
}

func Broadcast(name string, in interface{}) {
	mut.RLock()
	defer mut.RUnlock()

	for s := range connections {
		s.Notify(name, in)
	}
}

func StartServer() {
	ws := gotalk.WebSocketHandler()
	ws.OnAccept = onAccept

	folder := `/goflip/web`
	http.Handle("/socket/", ws)

	http.Handle("/",
		http.FileServer(
			http.Dir(folder),
		),
	)

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		jdata := make(map[string]string)

		jdata["version"] = ".1"

		js, err := json.Marshal(jdata)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		w.Write(js)
	})

	var port = ":8080"

	log.Infof("Server listening - http://%s%s", "127.0.0.1", port)

	err := http.ListenAndServe(port, nil)

	if err != nil {
		log.Error(err.Error())
	}
}
