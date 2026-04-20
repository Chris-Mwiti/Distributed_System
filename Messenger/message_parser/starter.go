package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Message struct {
	Src string `json:"src"`
	Dest string `json:"dest"`
	Body map[string]interface{} `json:"body"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()

		var msg Message

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Println(os.Stderr, "Error parsing JSON: ", err)
			continue
		}

		fmt.Printf("PARSED: %s|%s|%s", msg.Src, msg.Dest, msg.Body["type"])

	}
}
