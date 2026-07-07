package tcpserver

import (
	"context"
	"fmt"
)

// statChans holds the channels that the sys_stat handler fans out to.
// They are initialised once in StartTcpServer and shared across all
// connection goroutines via the closure in RegisterHandlers.
type statChans struct {
	stat    chan Msg
	monitor chan Msg
}

// HandleSysStat is called for every "sys_stat" message from an agent.
// It fans the message out to the storage channel and the alert monitor
// channel. Both sends are non-blocking: if a channel is full the message
// is dropped and a warning is logged rather than stalling the TCP loop.
func HandleSysStat(chans statChans) HandlerFunc {
	return func(ctx context.Context, c *ConnContext, msg Msg) {
		if c.NodeID == 0 {
			fmt.Printf("HandleSysStat: dropping stat from %s — handshake not complete\n", c.RemoteAddr)
			return
		}

		// Stamp the correct node ID in case the agent sent a stale value.
		msg.NodeId = int32(c.NodeID)

		// Fan out to storage
		select {
		case chans.stat <- msg:
		default:
			fmt.Printf("HandleSysStat: stat channel full, dropping message for node %d\n", c.NodeID)
		}

		// Fan out to alert monitor
		select {
		case chans.monitor <- msg:
		default:
			fmt.Printf("HandleSysStat: monitor channel full, dropping message for node %d\n", c.NodeID)
		}
	}
}
