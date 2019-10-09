package main

import (
	"encoding/binary"
	"time"

	"github.com/Rightech/ric-edge/third_party/go-ble/ble"
	log "github.com/sirupsen/logrus"
)

func nfHandler(typ string) ble.NotifyHandler {
	return ble.NotifyHandlerFunc(func(req ble.Request, n ble.Notifier) {
		cnt := uint32(0)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		log.Printf("count: %s subscribed\n", typ)
		for {
			select {
			case <-n.Context().Done():
				log.Printf("count: %s unsubscribed\n", typ)
				return
			case <-ticker.C:
				log.Printf("count: %s: %d", typ, cnt)

				bs := make([]byte, 4)
				binary.LittleEndian.PutUint32(bs, cnt)

				if _, err := n.Write(bs); err != nil {
					// Client disconnected prematurely before unsubscription.
					log.WithError(err).Errorf("count: Failed to %s", typ)
					return
				}
				cnt++
			}
		}
	})
}

// NewCountChar ...
func NewCountChar() *ble.Characteristic {
	n := uint32(0)

	c := ble.NewCharacteristic(ble.MustParse("00010000-0002-1000-8000-00805F9B34FB"))
	c.HandleRead(ble.ReadHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
		log.Printf("count: Read %d", n)

		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, n)

		_, err := rsp.Write(bs)
		if err != nil {
			log.WithError(err).Error("read: write err")
			rsp.SetStatus(ble.ErrInvalidHandle)
			return
		}

		rsp.SetStatus(ble.ErrSuccess)
	}))

	c.HandleWrite(ble.WriteHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
		val := binary.LittleEndian.Uint32(req.Data())

		log.Printf("count: Write %d", val)

		n = val

		rsp.SetStatus(ble.ErrSuccess)
	}))

	c.HandleNotify(nfHandler("notify"))
	c.HandleIndicate(nfHandler("indicate"))

	return c
}
