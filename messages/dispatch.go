package messages

import (
	"fmt"
	"time"
)

const MaxAsyncMessageProcCount = 1024

type MessageProcFunc = func(*Message) (error)

type Dispatcher struct {
	fns map[string] MessageProcFunc
	syncChannel chan *Message
	asyncCount int
}

type MessageObject interface {
	GetMappedTypes() (map[int] MessageProcFunc)
}

func CreateDispatcher() (*Dispatcher) {
	sor := new(Dispatcher)
	sor.fns = make(map[string] MessageProcFunc)
	sor.syncChannel = make(chan *Message, 102400)
	sor.asyncCount = 0
	return sor
}

func (receiver *Dispatcher) MessageMapped(scheme, tag, tp int, fn MessageProcFunc) {
	key := fmt.Sprintf("%d-%d-%d", scheme, tag, tp)
	receiver.fns[key] = fn
}


func (receiver *Dispatcher) MessageObjectMapped(scheme, tag int, o MessageObject) {
	fns := o.GetMappedTypes()
	for k,v := range fns {
		receiver.MessageMapped(scheme,tag,k,v)
	}
}

func (receiver *Dispatcher) execMessageProc(message *Message, count bool) {
	for _, tag := range message.messageTag {
		key := fmt.Sprintf("%d-%d-%d", message.messageScheme, tag, message.messageType)
		fn, ok := receiver.fns[key]
		if ok {
			if count {
				receiver.asyncCount += 1
			}
			fn(message)
			if count {
				receiver.asyncCount -= 1
			}
		}
	}
}

func (receiver *Dispatcher) Dispatch() {
	go func() {
		syncMessage, ok := <- receiver.syncChannel
		if ok {
			receiver.execMessageProc(syncMessage, false)
		}
	}()

	go func() {
		for {
			if receiver.asyncCount >= MaxAsyncMessageProcCount {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			message := GlobalMessageQueue.Pop()
			if message == nil {
				break
			}

			if message.messageSync {
				go func() {
					receiver.syncChannel <- message
				}()
			} else {
				go func() {
					receiver.execMessageProc(message, true)
				}()
			}
		}
	}()
}

var GlobalDispatcher = CreateDispatcher()