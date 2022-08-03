package protocol

import (
	"encoding/hex"
	"fmt"
	"io"
	"nothing-cli/display"
	"strings"

	"github.com/fatih/color"
)

const (
	http2Preface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

type GrpcInterop struct{}

func (g *GrpcInterop) Dump(r io.Reader, source string, id int, quiet bool) {
	g.readPrefect(r, source, id)
}

func (g *GrpcInterop) readPrefect(r io.Reader, source string, id int) {
	if source != ClientSide {
		return
	}

	preface := make([]byte, len(http2Preface))
	n, err := r.Read(preface)
	if err != nil || n < len(http2Preface) {
		return
	}

	fmt.Println()
	var builder strings.Builder
	builder.WriteString(color.HiGreenString("from %s [%d]\n", source, id))
	builder.WriteString(fmt.Sprintf("%s%s%s\n", color.HiBlueString("%s:(", grpcProtocol), color.YellowString("http2:preface"), color.HiBlueString(")")))
	builder.WriteString(fmt.Sprint(hex.Dump(preface)))
	display.PrintfWithTime(builder.String())
}
