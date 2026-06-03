// internal/domain/formatter.go
package domain

import (
	"encoding/base64"
	"strconv"
)

func formatEvent(evt Event) []byte {
	buf := make([]byte, 0, 32+len(evt.Message))
	buf = strconv.AppendInt(buf, evt.Timestamp, 10)
	buf = append(buf, ' ')
	buf = strconv.AppendInt(buf, int64(evt.SenderID), 10)
	buf = append(buf, ' ')
	buf = append(buf, evt.Kind.String()...)
	if len(evt.Message) > 0 {
		buf = append(buf, ' ')
		buf = append(buf, evt.Message...)
	}
	return append(buf, '\n')
}

func formatBootstrap(id MemberID, privSeed []byte) []byte {
	buf := make([]byte, 0, 66)
	buf = strconv.AppendInt(buf, int64(id), 10)
	buf = append(buf, ' ')
	buf = base64.StdEncoding.AppendEncode(buf, privSeed)
	return append(buf, '\n')
}
