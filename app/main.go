package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

var (
	store = make(map[string]string)
	mutex = sync.RWMutex{}
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		command, err := parseRESPArray(reader)
		if err != nil {
			fmt.Printf("Error parsing command: %v\n", err)
			return
		}

		if len(command) == 0 {
			continue
		}

		cmd := strings.ToUpper(command[0])
		switch cmd {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "ECHO":
			if len(command) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'echo' command\r\n"))
			} else {
				response := fmt.Sprintf("$%d\r\n%s\r\n", len(command[1]), command[1])
				conn.Write([]byte(response))
			}
		case "SET":
			if len(command) < 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
			} else {
				key := command[1]
				value := command[2]

				mutex.Lock()
				store[key] = value
				mutex.Unlock()

				conn.Write([]byte("+OK\r\n"))
			}
		case "GET":
			if len(command) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
			} else {
				key := command[1]

				mutex.RLock()
				value, exists := store[key]
				mutex.RUnlock()

				if exists {
					response := fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
					conn.Write([]byte(response))
				} else {
					conn.Write([]byte("$-1\r\n"))
				}
			}
		default:
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

func parseRESPArray(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("invalid RESP array format")
	}

	arrayLen, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}

	result := make([]string, arrayLen)

	for i := 0; i < arrayLen; i++ {
		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		lengthLine = strings.TrimSpace(lengthLine)
		if len(lengthLine) == 0 || lengthLine[0] != '$' {
			return nil, fmt.Errorf("invalid bulk string format")
		}

		strLen, err := strconv.Atoi(lengthLine[1:])
		if err != nil {
			return nil, err
		}

		data := make([]byte, strLen)
		_, err = reader.Read(data)
		if err != nil {
			return nil, err
		}

		reader.ReadString('\n')

		result[i] = string(data)
	}

	return result, nil
}
