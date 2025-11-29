package funcs

import (
	"context"
	"encoding/json"

	"zombiezen.com/go/sqlite"

	"github.com/horos/gopage/pkg/sse"
)

// RegisterSSEFunctions registers SSE-related SQL functions
func (r *Registry) RegisterSSEFunctions() {
	r.Register(&FuncDef{
		Name:          "sse_notify",
		NumArgs:       2,
		Deterministic: false,
		Description:   "Send SSE notification (event, data)",
		ScalarFunc:    funcSSENotify,
	})

	r.Register(&FuncDef{
		Name:          "sse_notify_channel",
		NumArgs:       3,
		Deterministic: false,
		Description:   "Send SSE notification to channel (channel, event, data)",
		ScalarFunc:    funcSSENotifyChannel,
	})

	r.Register(&FuncDef{
		Name:          "sse_broadcast",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Broadcast message to all SSE clients",
		ScalarFunc:    funcSSEBroadcast,
	})

	r.Register(&FuncDef{
		Name:          "sse_client_count",
		NumArgs:       0,
		Deterministic: false,
		Description:   "Returns number of connected SSE clients",
		ScalarFunc:    funcSSEClientCount,
	})

	r.Register(&FuncDef{
		Name:          "sse_channel_count",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Returns number of clients in a channel",
		ScalarFunc:    funcSSEChannelCount,
	})

	r.Register(&FuncDef{
		Name:          "sse_notify_json",
		NumArgs:       2,
		Deterministic: false,
		Description:   "Send SSE notification with JSON data (event, json_data)",
		ScalarFunc:    funcSSENotifyJSON,
	})
}

// funcSSENotify sends an SSE notification
func funcSSENotify(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	event := args[0].Text()
	data := args[1].Text()

	sse.Notify(event, data)
	return 1, nil
}

// funcSSENotifyChannel sends an SSE notification to a specific channel
func funcSSENotifyChannel(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 3 {
		return nil, nil
	}

	channel := args[0].Text()
	event := args[1].Text()
	data := args[2].Text()

	sse.NotifyChannel(channel, event, data)
	return 1, nil
}

// funcSSEBroadcast broadcasts a message to all clients
func funcSSEBroadcast(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	data := args[0].Text()
	sse.Notify("message", data)
	return 1, nil
}

// funcSSEClientCount returns the number of connected clients
func funcSSEClientCount(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	return int64(sse.GetHub().ClientCount()), nil
}

// funcSSEChannelCount returns the number of clients in a channel
func funcSSEChannelCount(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return int64(0), nil
	}

	channel := args[0].Text()
	return int64(sse.GetHub().ChannelClientCount(channel)), nil
}

// funcSSENotifyJSON sends a notification with JSON data
func funcSSENotifyJSON(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	event := args[0].Text()
	jsonStr := args[1].Text()

	// Parse JSON to verify it's valid
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// If not valid JSON, send as string
		sse.Notify(event, jsonStr)
	} else {
		sse.NotifyJSON(event, data)
	}

	return 1, nil
}
