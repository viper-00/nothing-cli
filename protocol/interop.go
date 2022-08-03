package protocol

import (
	"encoding/hex"
	"fmt"
	"io"
	"nothing-cli/display"
)

const (
	ServerSide   = "SERVER"
	ClientSide   = "CLIENT"
	grpcProtocol = "grpc"
	bufferSize   = 1 << 20
)

var interop defaultInterop

type Interop interface {
	Dump(r io.Reader, source string, id int, quiet bool)
}

func CreateInterop(protocol string) Interop {
	switch protocol {
	case grpcProtocol:
		return new(GrpcInterop)
	default:
		return interop
	}
}

type defaultInterop struct{}

func (d defaultInterop) Dump(r io.Reader, source string, id int, quiet bool) {
	data := make([]byte, bufferSize)
	for {
		n, err := r.Read(data)
		if n > 0 && !quiet {
			display.PrintfWithTime("from %s [%d]:\n", source, id)
			fmt.Println(hex.Dump(data[:n]))
		}
		if err != nil && err != io.EOF {
			fmt.Printf("Unable to read data %v", err)
			break
		}
		if n == 0 {
			break
		}
	}
}
