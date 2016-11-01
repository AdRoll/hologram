// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ldap

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/nmcclain/asn1-ber"
)

const (
	MessageQuit     = 0
	MessageRequest  = 1
	MessageResponse = 2
	MessageFinish   = 3
)

type messagePacket struct {
	Op        int
	MessageID uint64
	Packet    *ber.Packet
	Channel   chan *ber.Packet
}

// Conn represents an LDAP Connection
type Conn struct {
	conn          net.Conn
	isTLS         bool
	Debug         debugging
	chanConfirm   chan bool
	chanResults   map[uint64]chan *ber.Packet
	chanMessage   chan *messagePacket
	chanMessageID chan uint64
	wgSender      sync.WaitGroup
	chanDone      chan struct{}
	once          sync.Once
}

// Dial connects to the given address on the given network using net.Dial
// and then returns a new Conn for the connection.
func Dial(network, addr string) (*Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, NewError(ErrorNetwork, err)
	}
	conn := NewConn(c)
	conn.start()
	return conn, nil
}

// DialTimeout connects to the given address on the given network using net.DialTimeout
// and then returns a new Conn for the connection. Acts like Dial but takes a timeout.
func DialTimeout(network, addr string, timeout time.Duration) (*Conn, error) {
	c, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, NewError(ErrorNetwork, err)
	}
	conn := NewConn(c)
	conn.start()
	return conn, nil
}

// DialTLS connects to the given address on the given network using tls.Dial
// and then returns a new Conn for the connection.
func DialTLS(network, addr string, config *tls.Config) (*Conn, error) {
	c, err := tls.Dial(network, addr, config)
	if err != nil {
		return nil, NewError(ErrorNetwork, err)
	}
	conn := NewConn(c)
	conn.isTLS = true
	conn.start()
	return conn, nil
}

// DialTLSDialer connects to the given address on the given network using tls.DialWithDialer
// and then returns a new Conn for the connection.
func DialTLSDialer(network, addr string, config *tls.Config, dialer *net.Dialer) (*Conn, error) {
	c, err := tls.DialWithDialer(dialer, network, addr, config)
	if err != nil {
		return nil, NewError(ErrorNetwork, err)
	}
	conn := NewConn(c)
	conn.isTLS = true
	conn.start()
	return conn, nil
}

// NewConn returns a new Conn using conn for network I/O.
func NewConn(conn net.Conn) *Conn {
	return &Conn{
		conn:          conn,
		chanConfirm:   make(chan bool),
		chanMessageID: make(chan uint64),
		chanMessage:   make(chan *messagePacket, 10),
		chanResults:   map[uint64]chan *ber.Packet{},
		chanDone:      make(chan struct{}),
	}
}

func (l *Conn) start() {
	go l.reader()
	go l.processMessages()
}

// Close closes the connection.
func (l *Conn) Close() {
	l.once.Do(func() {
		close(l.chanDone)
		l.wgSender.Wait()

		l.Debug.Printf("Sending quit message and waiting for confirmation")
		l.chanMessage <- &messagePacket{Op: MessageQuit}
		<-l.chanConfirm
		close(l.chanMessage)

		l.Debug.Printf("Closing network connection")
		if err := l.conn.Close(); err != nil {
			log.Print(err)
		}
	})
	<-l.chanDone
}

// Returns the next available messageID
func (l *Conn) nextMessageID() uint64 {
	if l.chanMessageID != nil {
		if messageID, ok := <-l.chanMessageID; ok {
			return messageID
		}
	}
	return 0
}

// StartTLS sends the command to start a TLS session and then creates a new TLS Client
func (l *Conn) StartTLS(config *tls.Config) error {
	messageID := l.nextMessageID()

	if l.isTLS {
		return NewError(ErrorNetwork, errors.New("ldap: already encrypted"))
	}

	packet := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "LDAP Request")
	packet.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, messageID, "MessageID"))
	request := ber.Encode(ber.ClassApplication, ber.TypeConstructed, ApplicationExtendedRequest, nil, "Start TLS")
	request.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 0, "1.3.6.1.4.1.1466.20037", "TLS Extended Command"))
	packet.AppendChild(request)
	l.Debug.PrintPacket(packet)

	_, err := l.conn.Write(packet.Bytes())
	if err != nil {
		return NewError(ErrorNetwork, err)
	}

	packet, err = ber.ReadPacket(l.conn)
	if err != nil {
		return NewError(ErrorNetwork, err)
	}

	if l.Debug {
		if err := addLDAPDescriptions(packet); err != nil {
			return err
		}
		ber.PrintPacket(packet)
	}

	if packet.Children[1].Children[0].Value.(uint64) == 0 {
		conn := tls.Client(l.conn, config)
		l.isTLS = true
		l.conn = conn
	}

	return nil
}

func (l *Conn) closing() bool {
	select {
	case <-l.chanDone:
		return true
	default:
		return false
	}
}

func (l *Conn) sendMessage(packet *ber.Packet) (chan *ber.Packet, error) {
	if l.closing() {
		return nil, NewError(ErrorNetwork, errors.New("ldap: connection closed"))
	}
	out := make(chan *ber.Packet)
	message := &messagePacket{
		Op:        MessageRequest,
		MessageID: packet.Children[0].Value.(uint64),
		Packet:    packet,
		Channel:   out,
	}
	l.sendProcessMessage(message)
	return out, nil
}

func (l *Conn) finishMessage(messageID uint64) {
	if l.closing() {
		return
	}
	message := &messagePacket{
		Op:        MessageFinish,
		MessageID: messageID,
	}
	l.sendProcessMessage(message)
}

func (l *Conn) sendProcessMessage(message *messagePacket) bool {
	l.wgSender.Add(1)
	defer l.wgSender.Done()

	if l.closing() {
		return false
	}
	l.chanMessage <- message
	return true
}

func (l *Conn) processMessages() {
	defer func() {
		for messageID, channel := range l.chanResults {
			l.Debug.Printf("Closing channel for MessageID %d", messageID)
			close(channel)
			delete(l.chanResults, messageID)
		}
		close(l.chanMessageID)
		l.chanConfirm <- true
		close(l.chanConfirm)
	}()

	var messageID uint64 = 1
	for {
		select {
		case l.chanMessageID <- messageID:
			messageID++
		case messagePacket, ok := <-l.chanMessage:
			if !ok {
				l.Debug.Printf("Shutting down - message channel is closed")
				return
			}
			switch messagePacket.Op {
			case MessageQuit:
				l.Debug.Printf("Shutting down - quit message received")
				return
			case MessageRequest:
				// Add to message list and write to network
				l.Debug.Printf("Sending message %d", messagePacket.MessageID)
				l.chanResults[messagePacket.MessageID] = messagePacket.Channel
				// go routine
				buf := messagePacket.Packet.Bytes()

				_, err := l.conn.Write(buf)
				if err != nil {
					l.Debug.Printf("Error Sending Message: %s", err.Error())
					break
				}
			case MessageResponse:
				l.Debug.Printf("Receiving message %d", messagePacket.MessageID)
				if chanResult, ok := l.chanResults[messagePacket.MessageID]; ok {
					chanResult <- messagePacket.Packet
				} else {
					log.Printf("Received unexpected message %d", messagePacket.MessageID)
					ber.PrintPacket(messagePacket.Packet)
				}
			case MessageFinish:
				// Remove from message list
				l.Debug.Printf("Finished message %d", messagePacket.MessageID)
				close(l.chanResults[messagePacket.MessageID])
				delete(l.chanResults, messagePacket.MessageID)
			}
		}
	}
}

func (l *Conn) reader() {
	defer func() {
		l.Close()
	}()

	for {
		packet, err := ber.ReadPacket(l.conn)
		if err != nil {
			l.Debug.Printf("reader: %s", err.Error())
			return
		}
		addLDAPDescriptions(packet)
		message := &messagePacket{
			Op:        MessageResponse,
			MessageID: packet.Children[0].Value.(uint64),
			Packet:    packet,
		}
		if !l.sendProcessMessage(message) {
			return
		}

	}
}

// Use Abandon operation to perform connection keepalives
func (l *Conn) Ping() error {

	messageID := l.nextMessageID()

	packet := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "LDAP Request")
	packet.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, messageID, "MessageID"))
	abandonRequest := ber.Encode(ber.ClassApplication, ber.TypePrimitive, ApplicationAbandonRequest, nil, "Abandon Request")
	packet.AppendChild(abandonRequest)

	if l.Debug {
		ber.PrintPacket(packet)
	}

	channel, err := l.sendMessage(packet)
	if err != nil {
		return err
	}
	if channel == nil {
		return NewError(ErrorNetwork, errors.New("ldap: could not send message"))
	}
	defer l.finishMessage(messageID)

	if l.Debug {
		if err := addLDAPDescriptions(packet); err != nil {
			return err
		}
		ber.PrintPacket(packet)
	}

	return nil
}
