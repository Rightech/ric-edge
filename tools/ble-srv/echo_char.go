package main

import (
	"log"
	"sync"
	"time"

	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
)

// NewEchoChar ...
func NewEchoChar() *ble.Characteristic {
	e := &echoChar{m: make(map[string]chan []byte)}
	c := ble.NewCharacteristic(ble.MustParse("00020000-0002-1000-8000-00805F9B34FB"))
	c.HandleWrite(ble.WriteHandlerFunc(e.written))
	c.HandleNotify(ble.NotifyHandlerFunc(e.echo))
	c.HandleIndicate(ble.NotifyHandlerFunc(e.echo))
	return c
}

type echoChar struct {
	sync.Mutex
	m map[string]chan []byte
}

func (e *echoChar) written(req ble.Request, rsp ble.ResponseWriter) {
	e.Lock()
	e.m[req.Conn().RemoteAddr().String()] <- req.Data()
	e.Unlock()
}

func (e *echoChar) echo(req ble.Request, n ble.Notifier) {
	ch := make(chan []byte)
	e.Lock()
	e.m[req.Conn().RemoteAddr().String()] = ch
	e.Unlock()
	log.Printf("echo: Notification subscribed")
	defer func() {
		e.Lock()
		delete(e.m, req.Conn().RemoteAddr().String())
		e.Unlock()
	}()
	for {
		select {
		case <-n.Context().Done():
			log.Printf("echo: Notification unsubscribed")
			return
		case <-time.After(time.Second * 20):
			log.Printf("echo: timeout")
			return
		case msg := <-ch:
			if _, err := n.Write(msg); err != nil {
				log.Printf("echo: can't indicate: %s", err)
				return
			}
		}
	}
}
