package notification

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/meleket/server/v2/api"
	"github.com/meleket/server/v2/auth"
	"github.com/meleket/server/v2/mode"
	"github.com/meleket/server/v2/model"
)

type API struct {
	clients     map[uint][]*client
	lock        sync.RWMutex
	pingPeriod  time.Duration
	pongTimeout time.Duration
	upgrader    *websocket.Upgrader
	clientAPI   *api.ClientAPI
}

func New(pingPeriod, pongTimeout time.Duration, allowedWebSocketOrigins []string) *API {
	return &API{
		clients:     make(map[uint][]*client),
		pingPeriod:  pingPeriod,
		pongTimeout: pingPeriod + pongTimeout,
		upgrader:    newUpgrader(allowedWebSocketOrigins),
	}
}

func (a *API) CollectConnectedClientTokens() []string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var clients []string
	for _, cs := range a.clients {
		for _, c := range cs {
			clients = append(clients, c.token)
		}
	}
	return uniq(clients)
}

func (a *API) NotifyDeletedClient(userID uint, token string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if clients, ok := a.clients[userID]; ok {
		for i := len(clients) - 1; i >= 0; i-- {
			client := clients[i]
			if client.token == token {
				client.Close()
				clients = append(clients[:i], clients[i+1:]...)
			}
		}
		a.clients[userID] = clients
	}
}

func (a *API) NotifyClient(userID uint, clientToken string, msg *model.NotificationMessageExternal) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if clients, ok := a.clients[userID]; ok {
		for _, c := range clients {
			if c.token == clientToken {
				c.write <- msg
				return
			}
		}
	}
}

func (a *API) remove(remove *client) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if userIDClients, ok := a.clients[remove.userID]; ok {
		for i, client := range userIDClients {
			if client == remove {
				a.clients[remove.userID] = append(userIDClients[:i], userIDClients[i+1:]...)
				break
			}
		}
	}
}

func (a *API) register(client *client) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.clients[client.userID] = append(a.clients[client.userID], client)
}

func (a *API) Handle(ctx *gin.Context) {
	clientToken := getClientTokenParam(ctx, "clientToken")
	// if flag := a.clientAPI.ClientExists(clientToken); flag != true {
	// ctx.AbortWithError(404, fmt.Errorf("client with token %s doesnot exist", clientToken))
	// }
	conn, err := a.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.Error(err)
		return
	}

	notifyClient := newClient(conn, auth.GetUserID(ctx), clientToken, a.remove)
	a.register(notifyClient)
	go notifyClient.startReading(a.pongTimeout)
	go notifyClient.startWriteHandler(a.pingPeriod)
}

func (a *API) Close() {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, clients := range a.clients {
		for _, client := range clients {
			client.Close()
		}
	}
	for k := range a.clients {
		delete(a.clients, k)
	}
}

func uniq[T comparable](s []T) []T {
	m := make(map[T]struct{})
	for _, v := range s {
		m[v] = struct{}{}
	}
	var r []T
	for k := range m {
		r = append(r, k)
	}
	return r
}

func isAllowedOrigin(r *http.Request, allowedOrigins []*regexp.Regexp) bool {
	origin := r.Header.Get("origin")
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if strings.EqualFold(u.Host, r.Host) {
		return true
	}

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin.MatchString(strings.ToLower(u.Hostname())) {
			return true
		}
	}

	return false
}

func newUpgrader(allowedWebSocketOrigins []string) *websocket.Upgrader {
	compiledAllowedOrigins := compileAllowedWebSocketOrigins(allowedWebSocketOrigins)
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if mode.IsDev() {
				return true
			}
			return isAllowedOrigin(r, compiledAllowedOrigins)
		},
	}
}

func compileAllowedWebSocketOrigins(allowedOrigins []string) []*regexp.Regexp {
	var compiledAllowedOrigins []*regexp.Regexp
	for _, origin := range allowedOrigins {
		compiledAllowedOrigins = append(compiledAllowedOrigins, regexp.MustCompile(origin))
	}

	return compiledAllowedOrigins
}

func getClientTokenParam(ctx *gin.Context, name string) string {
	return ctx.Param(name)
}
