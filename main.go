package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	users      = make(map[net.Conn]string, 10)
	m          sync.Mutex
	counter    = 0
	messages   = make(chan message)
	leaving    = make(chan message)
	connecting = make(chan message)
)

var fileName *os.File

func main() {
	port := "8989"
	input := os.Args[1:]
	if len(input) > 1 {
		fmt.Println("[USAGE]: ./TCPChat $port")
	} else if len(input) == 1 {
		port = input[0]
	}

	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		return
	}
	fileName, err = createTxt()
	if err != nil {
		log.Println(err)
		return
	}
	go broadcastConn(messages, fileName)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		iterateCounter()
		m.Lock()
		if counter > 10 {
			fmt.Fprintln(conn, "Sorry, the chat is full")
			counter--
			conn.Close()
			m.Unlock()
			continue
		}
		m.Unlock()
		go connectUser(conn, messages)
	}
}

// increases the counter by one
func iterateCounter() {
	m.Lock()
	counter++
	m.Unlock()
}

// decreases the counter by one
func downCounter() {
	m.Lock()
	counter--
	m.Unlock()
}

// adds the user to the map
func connectUser(conn net.Conn, messages chan message) {
	for {
		welcomeMessage(conn)
		username, err := readLine(conn)
		if err != nil {
			conn.Close()
			return
		}
		m.Lock()
		if !checkLogins(conn, username) {
			m.Unlock()
			downCounter()
			continue
		} else {
			users[conn] = username
			m.Unlock()
			printHistory(conn)
			dt := time.Now().Format("2006-01-02 15:04:05")
			connecting <- writeMessage(username+" joined chat", conn, dt)
		}
		handleConn(conn, messages, username)
	}
}

func checkLogins(conn net.Conn, username string) bool {
	for _, val := range users {
		if val == username {
			fmt.Fprintln(conn, "Username is taken, please enter a new one")
			return false
		}
	}
	if username == " " || username == "\n" || username == "" {
		fmt.Fprintln(conn, "Username is invalid, please enter a new one")
		return false
	}
	return true
}

func handleConn(conn net.Conn, messages chan message, login string) {
	defer conn.Close()
	for {
		dt := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(conn, "[%s][%s]: ", dt, login)
		incomingMsg, err := readLine(conn)
		if err != nil {
			downCounter()
			m.Lock()
			toSend := login + " has left chat..."
			delete(users, conn)
			m.Unlock()
			dt := time.Now().Format("2006-01-02 15:04:05")
			leaving <- writeMessage(toSend, conn, dt)
			return
		}
		if incomingMsg == "\n" || incomingMsg == "" || incomingMsg == " " {
			continue
		}
		toSend := "[" + dt + "][" + login + "]: " + incomingMsg
		dt = time.Now().Format("2006-01-02 15:04:05")
		messages <- writeMessage(toSend, conn, dt)
	}
}

func writeMessage(inputMessage string, conn net.Conn, dt string) message {
	return message{
		text:    inputMessage,
		address: conn,
		time:    "[" + dt + "]",
	}
}

type message struct {
	text    string
	address net.Conn
	time    string
}

func printHistory(conn net.Conn) {
	content, err := os.ReadFile("txtfiles/data.txt")
	if err != nil {
		return
	}
	fmt.Fprint(conn, string(content))
}

func readLine(conn net.Conn) (string, error) {
	input := bufio.NewReader(conn)
	incomingMsg, err := input.ReadString('\n')
	if err != nil {
		return "", err
	}
	incomingMsg = strings.TrimSpace(incomingMsg)
	return incomingMsg, err
}

func welcomeMessage(conn net.Conn) {
	content, _ := os.ReadFile("txtfiles/welcome.txt")
	fmt.Fprint(conn, strings.TrimSuffix(string(content), "\n"))
}

func broadcastConn(my chan message, logfile *os.File) {
	for {
		select {
		case newMsg := <-messages:
			logfile.WriteString(newMsg.text + "\n")
			m.Lock()
			for key, val := range users {
				if key == newMsg.address {
					continue
				} else {
					fmt.Fprintln(key, "\n"+newMsg.text)
					fmt.Fprint(key, newMsg.time+"["+val+"]: ")
				}
			}
			m.Unlock()
		case newMsg := <-connecting:
			logfile.WriteString(newMsg.text + "\n")
			m.Lock()
			for key, val := range users {
				if key == newMsg.address {
					continue
				} else {
					fmt.Fprintln(key, "\n"+newMsg.text)
					fmt.Fprint(key, newMsg.time+"["+val+"]: ")
				}
			}
			m.Unlock()
		case newMsg := <-leaving:
			logfile.WriteString(newMsg.text + "\n")
			m.Lock()
			for key, val := range users {
				if key == newMsg.address {
					continue
				} else {
					fmt.Fprintln(key, "\n"+newMsg.text)
					fmt.Fprint(key, newMsg.time+"["+val+"]: ")
				}
			}
			m.Unlock()
		}
	}
}

func createTxt() (*os.File, error) {
	fileName, err := os.Create("txtfiles/data.txt")
	return fileName, err
}
