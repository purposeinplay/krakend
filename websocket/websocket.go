package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	router "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"
	"github.com/go-contrib/uuid"
	"golang.org/x/net/websocket"
	melody "gopkg.in/olahol/melody.v1"
)

const (
	Namespace = "github.com/ProvablyFair/krakend/websocket"

	ClientIntroduction    = `{"msg":"KrakenD WS proxy starting"}`
	ClientIntroductionACK = "OK"
)

func HandlerFactory(ctx context.Context, logger logging.Logger, next router.HandlerFactory) router.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		opt, ok := getOptions(cfg).(*options)
		if !ok || opt == nil {
			logger.Debug("WS: ignoring endpoint", cfg.Endpoint)
			return next(cfg, p)
		}
		logger.Debug("WS: endpoint", cfg.Endpoint, *opt)
		return newHandler(ctx, logger, *opt)
	}
}

type Message struct {
	URL     string            `json:"url,omitempty"`
	Session map[string]string `json:"session,omitempty"`
	Body    []byte            `json:"body"`
}

func newHandler(ctx context.Context, logger logging.Logger, opt options) gin.HandlerFunc {
	m := melody.New()

	client, err := newClient(ctx, logger, opt, func(b []byte) { processResponse(logger, m, b) })
	if err != nil {
		logger.Fatal(err)
	}

	m.HandleMessage(processRequest(logger, client))

	w := &handlerFactory{
		l: logger,
		c: client,
		m: m,
	}
	return w.Handler()
}

type handlerFactory struct {
	l logging.Logger
	m *melody.Melody
	c *client
}

func (w *handlerFactory) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		keys := map[string]interface{}{
			"uuid": uuid.NewV4().String(),
		}
		for _, param := range c.Params {
			keys[strings.Title(param.Key)] = param.Value
		}
		w.m.HandleRequestWithKeys(c.Writer, c.Request, keys)
	}
}

func processRequest(logger logging.Logger, c *client) func(*melody.Session, []byte) {
	return func(s *melody.Session, msg []byte) {
		ses := make(map[string]string, len(s.Keys))
		for k, v := range s.Keys {
			ses[k], _ = v.(string)
		}
		req, err := json.Marshal(Message{
			URL:     s.Request.URL.Path,
			Session: ses,
			Body:    msg,
		})
		if err != nil {
			logger.Warning("WS Preparing request:", err.Error())
			return
		}

		if _, err := c.Write(req); err != nil {
			logger.Warning("WS Writing request:", err.Error())
			return
		}
	}
}

func processResponse(logger logging.Logger, m *melody.Melody, msg []byte) error {
	r := &Message{}
	if err := json.Unmarshal(msg, r); err != nil {
		logger.Debug("WS backend sent a response without an envelope:", err.Error())
		return m.Broadcast(msg)
	}

	return m.BroadcastFilter(r.Body, broadcastFilter(r))
}

func broadcastFilter(r *Message) func(*melody.Session) bool {
	return func(q *melody.Session) bool {
		if r.URL != "" && r.URL != q.Request.URL.Path {
			return false
		}
		if len(r.Session) > len(q.Keys) {
			return false
		}
		for k, expected := range r.Session {
			raw, ok := q.Keys[k]
			if !ok {
				return false
			}
			s, ok := raw.(string)
			if !ok {
				return false
			}
			if s != expected {
				return false
			}
		}
		return true
	}
}

type options struct {
	URL    string
	Origin string
}

func getOptions(e *config.EndpointConfig) interface{} {
	if len(e.Backend) != 1 {
		return nil
	}
	if _, ok := e.ExtraConfig[Namespace]; !ok {
		return nil
	}

	u, err := url.Parse(e.Backend[0].Host[0])
	if err != nil {
		log.Fatal(err)
	}
	u.Path = path.Join(u.Path, e.Backend[0].URLPattern)

	return &options{
		URL:    u.String(),
		Origin: e.Endpoint,
	}
}

func newClient(ctx context.Context, logger logging.Logger, opt options, h func([]byte)) (*client, error) {
	ws, err := newConn(logger, opt)
	if err != nil {
		return nil, err
	}

	c := &client{
		conn:   ws,
		logger: logger,
		opt:    opt,
	}

	go c.run(ctx, h)

	return c, nil
}

type client struct {
	conn   io.ReadWriteCloser
	logger logging.Logger
	opt    options
}

func (c *client) Write(in []byte) (int, error) {
	n, err := c.conn.Write(in)
	if err == nil {
		return n, nil
	}
	newConn, err := newConn(c.logger, c.opt)
	if err != nil {
		return n, err
	}
	c.conn.Close()
	c.conn = newConn
	return c.conn.Write(in)
}

func (c *client) run(ctx context.Context, h func([]byte)) {
	buf := make([]byte, 512)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := c.conn.Read(buf)
		if err != nil {
			c.logger.Warning("WS client:", err.Error())
			newConn, err := newConn(c.logger, c.opt)
			if err != nil {
				return
			}
			c.conn.Close()
			c.conn = newConn
			continue
		}

		h(buf[:n])
	}
}

func newConn(logger logging.Logger, opt options) (*Conn, error) {
	ws, err := websocket.Dial(opt.URL, "", opt.Origin)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		ws:     ws,
		logger: logger,
		once:   new(sync.Once),
	}
	c.init()
	return c, nil
}

type Conn struct {
	ws     *websocket.Conn
	once   *sync.Once
	logger logging.Logger
}

func (c *Conn) init() error {
	var err error
	c.once.Do(func() {
		if _, localErr := c.ws.Write([]byte(ClientIntroduction)); localErr != nil {
			err = localErr
			return
		}
		resp := make([]byte, 512)
		n, localErr := c.ws.Read(resp)
		if localErr != nil {
			err = localErr
			return
		}
		c.logger.Debug("WS received msg:", string(resp[:n]))
		if string(resp[:n]) != ClientIntroductionACK {
			err = errors.New("WS backend did not ack the proxy introduction")
			return
		}
	})

	return err
}

func (c *Conn) Write(in []byte) (int, error) {
	if c.ws == nil {
		return 0, errors.New("empty connection")
	}
	return c.ws.Write(in)
}

func (c *Conn) Read(out []byte) (int, error) {
	if c.ws == nil {
		return 0, errors.New("empty connection")
	}
	return c.ws.Read(out)
}

func (c *Conn) Close() error {
	if c.ws == nil {
		return errors.New("empty connection")
	}
	return c.ws.Close()
}
