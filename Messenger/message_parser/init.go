package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Node represents a Maelstrom node
type Node struct {
	NodeID    string
	NodeIDs   []string
	NextMsgID int
	mu        sync.Mutex
}

// Send sends a message to a destination node
func (n *Node) Send(dest string, body map[string]interface{}) {
	// TODO: Implement message sending

	n.mu.Lock()
	defer n.mu.Unlock()

	body["msg_id"] = n.NextMsgID;
	//@todo: figure out why do wee need to increment
	n.NextMsgID++

	var msg Message

	msg.Src = n.NodeID
	msg.Dest = dest
	msg.Body = body

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return
	}

	fmt.Fprintln(os.Stdout,string(data))
}

// Reply sends a response to an incoming request
func (n *Node) Reply(request Message, body map[string]interface{}) {
	// TODO: Implement reply with in_reply_to
	body["in_reply_to"] = request.Body["msg_id"]
	n.Send(request.Src, body)
}

func Run() {
	node := &Node{}
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			continue
		}

		msgType, _ := msg.Body["type"].(string)
		if msgType == "init" {
			// TODO: Handle init message
			// 1. Store node_id and node_ids
			// 2. Reply with init_ok

			captureNodeIds, ok := msg.Body["node_ids"].([]string)
			
			if !ok {
				fmt.Printf("error while parsing node ids: %v", captureNodeIds)
				continue
			}

			node.mu.Lock()
			node.NodeID = msg.Body["node_id"].(string)
			for _, val := range msg.Body["node_ids"].([]interface{}) {
				node.NodeIDs = append(node.NodeIDs, val.(string))
			}
			node.NextMsgID = msg.Body["msg_id"].(int)
			node.mu.Unlock()

			respBody := make(map[string]interface{})
			respBody["type"] = "init_ok"

			node.Reply(msg, respBody)
			continue
		} else {
			fmt.Println("unsupported msg type")
			return
		}
	}
}
