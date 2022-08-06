package protocol

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"nothing-cli/display"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/net/http2"
)

const (
	http2HeaderLen = 9
	http2Preface   = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

type GrpcInterop struct{}

func (g *GrpcInterop) Dump(r io.Reader, source string, id int, quiet bool) {
	g.readPrefect(r, source, id)

	data := make([]byte, bufferSize)
	for {
		n, err := r.Read(data)
		if n > 0 && !quiet {
			var buf strings.Builder
			buf.WriteString(color.HiGreenString("from %s [%d]\n", source, id))

			var index int
			for index < n {
				frameInfo, moreInfo, offset := g.explain(data[index:n])
				buf.WriteString(fmt.Sprintf("%s%s%s\n", color.HiBlueString("%s:(", grpcProtocol), color.HiYellowString(frameInfo), color.HiBlueString(")")))
				end := index + offset
				if end > n {
					end = n
				}
				buf.WriteString(fmt.Sprint(hex.Dump(data[index:end])))
				if len(moreInfo) > 0 {
					buf.WriteString(fmt.Sprintf("\n%s\n\n", strings.TrimSpace(moreInfo)))
				}
				index += offset
			}
			display.PrintfWithTime("%s\n\n", strings.TrimSpace(buf.String()))
		}
		if err != nil && err != io.EOF {
			fmt.Printf("unable to read data %v", err)
		}
		if n == 0 {
			break
		}
	}
}

func (g *GrpcInterop) explain(b []byte) (string, string, int) {
	if len(b) < http2HeaderLen {
		return "", "", len(b)
	}

	frame, err := http2.ReadFrameHeader(bytes.NewReader(b[:http2HeaderLen]))
	if err != nil {
		return "", "", len(b)
	}

	frameLen := http2HeaderLen + int(frame.Length)
	switch frame.Type {
	case http2.FrameSettings:
		switch frame.Flags {
		case http2.FlagSettingsAck:
			return "http2:settings:ack", "", frameLen
		default:
			return g.explainSettings(b[http2HeaderLen:frameLen]), "", frameLen
		}
	}

	return "http2:" + strings.ToLower(frame.Type.String()), "", frameLen
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
