package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Node struct {
	NodeID    string
	NodeIDs   []string
	NextMsgID int
	mu        sync.Mutex
	outMu     sync.Mutex
}

type Message struct {
	Src  string                 `json:"src"`
	Dest string                 `json:"dest"`
	Body map[string]interface{} `json:"body"`
}

func (n *Node) Send(dest string, body map[string]interface{}) {
	n.mu.Lock()
	body["msg_id"] = n.NextMsgID
	n.NextMsgID++
	n.mu.Unlock()

	msg := Message{Src: n.NodeID, Dest: dest, Body: body}
	output, err := json.Marshal(msg)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while marshalling res: ", err)
		return
	}
	
	n.outMu.Lock()
	fmt.Println(string(output))
	n.outMu.Unlock()
}

func (n *Node) Reply(request Message, body map[string]interface{}) {
	if msgID, ok := request.Body["msg_id"].(float64); ok {
		body["in_reply_to"] = int(msgID)
	}
	n.Send(request.Src, body)
}

func (n *Node) HandleMessages(msg Message) {

	msgType, _ := msg.Body["type"].(string)
	switch msgType {
	case "init":

		n.mu.Lock()
		n.NodeID, _ = msg.Body["node_id"].(string)
		if ids, ok := msg.Body["node_ids"].([]interface{}); ok {
			for _, id := range ids {
				n.NodeIDs = n.NodeIDs[:0]
				n.NodeIDs = append(n.NodeIDs, id.(string))
			}
		}
		n.mu.Unlock()
		n.Reply(msg, map[string]interface{}{"type": "init_ok"})
	case "echo":
		// TODO: Handle echo message
		// Reply with echo_ok containing the same echo value
		body := make(map[string]interface{})

		body["type"] = "echo_ok"
		body["echo"] = msg.Body["echo"]
		n.Reply(msg, body)
	
	default:
		fmt.Println("Unknown command")
	}


}

func main() {
	node := &Node{}
	scanner := bufio.NewScanner(os.Stdin)

	var wg sync.WaitGroup
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			continue
		}

		wg.Add(1)
		go func(msg Message){
			defer wg.Done()	
			node.HandleMessages(msg)
		}(msg)
		wg.Wait()
	}

}

