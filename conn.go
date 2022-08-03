package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"nothing-cli/display"
	"sync"
	"time"

	"nothing-cli/protocol"

	"github.com/fatih/color"
)

const (
	statInterval    = time.Second * 5
	useOfClosedConn = "use of closed network connection"
)

var (
	errClientCanceled = errors.New("client canceled")
	stat              Stater
)

func startListener() error {
	stat = NewStatPrinter(statInterval)
	go stat.Start()

	conn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", settings.LocalHost, settings.LocalPort))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer conn.Close()

	display.PrintfWithTime("Listening on %s...\n", conn.Addr().String())

	var connIndex int
	for {
		cliConn, err := conn.Accept()
		if err != nil {
			return fmt.Errorf("server: accept: %w", err)
		}

		connIndex++
		display.PrintlnWithTime(color.HiGreenString("[%d] Accepted from: %s", connIndex, cliConn.RemoteAddr()))
		stat.AddConn(fmt.Sprintf("%d:client", connIndex), cliConn.(*net.TCPConn))
		pconn := NewPairedConnection(connIndex, cliConn)
		go pconn.process()
	}
}

type PairedConnection struct {
	id       int
	cliConn  net.Conn
	svrConn  net.Conn
	once     sync.Once
	stopChan chan struct{}
}

func NewPairedConnection(id int, cliConn net.Conn) *PairedConnection {
	return &PairedConnection{
		id:       id,
		cliConn:  cliConn,
		stopChan: make(chan struct{}),
	}
}

func (p *PairedConnection) process() {
	defer p.stop()

	conn, err := net.Dial("tcp", settings.Remote)
	if err != nil {
		display.PrintlnWithTime(color.HiRedString("[x][%d] Couldn't connect to server: %v", p.id, err))
		return
	}

	display.PrintlnWithTime(color.HiGreenString("[%d] Connected to server: %s", p.id, conn.RemoteAddr()))

	stat.AddConn(fmt.Sprintf("%d:server", p.id), conn.(*net.TCPConn))
	p.svrConn = conn
	go p.handleServerMessage()

	p.handleClientMessage()
}

func (p *PairedConnection) stop() {
	p.once.Do(func() {
		close(p.stopChan)
		stat.DelConn(fmt.Sprintf("%d:server", p.id))
		stat.DelConn(fmt.Sprintf("%d:client", p.id))

		if p.cliConn != nil {
			display.PrintlnWithTime(color.HiBlueString("[%d] Client connection closed", p.id))
			p.cliConn.Close()
		}
		if p.svrConn != nil {
			display.PrintlnWithTime(color.HiBlueString("[%d] Server connection closed", p.id))
			p.svrConn.Close()
		}
	})
}

func (p *PairedConnection) handleServerMessage() {
	defer p.stop()

	r, w := io.Pipe()
	tee := io.MultiWriter(newDelayedWriter(p.cliConn, settings.Delay, p.stopChan), w)
	go protocol.CreateInterop(settings.Protocol).Dump(r, protocol.ServerSide, p.id, settings.Quiet)
	p.copyData(tee, p.svrConn, protocol.ServerSide)
}

func (p *PairedConnection) handleClientMessage() {

}

func (p *PairedConnection) copyData(dst io.Writer, src io.Reader, tag string) {
	_, e := io.Copy(dst, src)
	if e != nil && e != io.EOF {
		netOpError, ok := e.(*net.OpError)
		if ok && netOpError.Err.Error() != useOfClosedConn {
			reason := netOpError.Unwrap().Error()
			display.PrintlnWithTime(color.HiRedString("[%d] %s error, %s", p.id, tag, reason))
		}
	}
}
