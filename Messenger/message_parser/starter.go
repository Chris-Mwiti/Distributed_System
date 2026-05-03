package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Node struct {
	NodeID    string
	NodeIDs   []string
	NextMsgID int
	ProxyRegistry map[string]interface{}
	mu        sync.Mutex
	outMu     sync.Mutex
}

type Message struct {
	Src  string                 `json:"src"`
	Dest string                 `json:"dest"`
	Body map[string]interface{} `json:"body"`
}

func (n *Node) Send(dest string, body map[string]interface{}) (*Message, error){
	n.mu.Lock()
	body["msg_id"] = n.NextMsgID
	n.NextMsgID++
	n.mu.Unlock()

	msg := Message{Src: n.NodeID, Dest: dest, Body: body}
	output, err := json.Marshal(msg)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while marshalling res: ", err)
		return nil, err
	}
	
	n.outMu.Lock()
	fmt.Println(string(output))
	n.outMu.Unlock()

	return &msg, nil
}

func (n *Node) Reply(request Message, body map[string]interface{})(*Message, error) {
	if msgID, ok := request.Body["msg_id"].(float64); ok {
		body["in_reply_to"] = int(msgID)
	}
	res, err := n.Send(request.Src, body)
	
	if err != nil {
		fmt.Fprintln(os.Stderr, "error while sending request", err)
		return nil, err
	}

	return res, nil
}

func (n *Node) Init(msg Message) (*Message, error){
	n.mu.Lock()
	n.NodeID, _ = msg.Body["node_id"].(string)
	if ids, ok := msg.Body["node_ids"].([]interface{}); ok {
		for _, id := range ids {
			n.NodeIDs = n.NodeIDs[:0]
			n.NodeIDs = append(n.NodeIDs, id.(string))
		}
	}
	n.mu.Unlock()
	res, err := n.Reply(msg, map[string]interface{}{"type": "init_ok"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error while reply request: ", err)
		return nil, err
	}
	return res, nil
}

func (n *Node) Echo(msg Message) (*Message, error){
	// TODO: Handle echo message
	// Reply with echo_ok containing the same echo value
	body := make(map[string]interface{})

	body["type"] = "echo_ok"
	body["echo"] = msg.Body["echo"]

	res, err := n.Reply(msg, map[string]interface{}{"type": "init_ok"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error while reply request: ", err)
		return nil, err
	}
	return res, nil
}

func (n *Node) sync_rpc(msg Message, ctx context.Context, ch chan bool) (*Message, error){
	innerReq := make(map[string]interface{})
	innerReq = msg.Body["inner"].(map[string]interface{})
	innerReq["msg_id"] = msg.Body["msg_id"]

	//patched request
	var reqMsg Message
	reqMsg.Src = n.NodeID
	//@todo: check if the target node for the proxy exists within the registry
	reqMsg.Dest = msg.Body["target"].(string)	
	reqMsg.Body = innerReq

	return n.HandleMessages(reqMsg, ctx, ch)	
}

func (n *Node) HandleMessages(msg Message, ctx context.Context, ch chan bool) (*Message, error){

	msgType, _ := msg.Body["type"].(string)
	var globalRes *Message
	var globalErr error
	switch msgType {
	case "init":
		res, err := n.Init(msg)
		globalRes = res
		globalErr = err
		if err != nil {
			fmt.Fprintln(os.Stderr, "error while initiating node: ", err)
		}
		ch <- true
	case "echo":
		res, err := n.Init(msg)
		globalRes = res
		globalErr = err

		if err != nil {
			fmt.Fprintln(os.Stderr, "error while initiating node: ", err)
		}

		ch <- true
	case "proxy":
		select{
		case <- ctx.Done(): 
			return nil, nil
		default: 
	    key := msg.Body["msg_id"].(string)	
			res, err := n.sync_rpc(msg, ctx, ch)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error while sync rpc: ", err)
				ch <- true
			}
			n.ProxyRegistry[key] = *res
			globalRes= res
			globalErr = err
			ch <- true
		}
	default:
		fmt.Println("Unknown command")
	}

	return globalRes, globalErr

}

func main() {
	node := &Node{}
	scanner := bufio.NewScanner(os.Stdin)
	ctx, cancel := context.WithTimeout(context.Background(), 60 * time.Second)
	eventCh := make(chan bool)

	defer cancel()

	var wg sync.WaitGroup
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			continue
		}

		wg.Add(1)
		go func(msg Message, ctx context.Context, ch chan bool){
			defer wg.Done()	
			node.HandleMessages(msg, ctx, ch)
			<-eventCh
		}(msg, ctx, eventCh)
		wg.Wait()
	}

}

