package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

type UserProfile struct {
	handle string
	rw     *bufio.ReadWriter
}

const (
	Port = ":14610"
)

var (
	connections map[string]*UserProfile = make(map[string]*UserProfile)
)

func NewUserProfile(name string) error {
	_, ok := connections[name]
	if ok {
		return errors.New("Handle already in use")
	}
	connections[name] = &UserProfile{handle: name}
	return nil
}

func getAllOnlineUsers() (ret string) {
	ret = "Online Users "
	for handle := range connections {
		ret += handle + " "
	}
	return ret
}

func Open(addr string) (*bufio.ReadWriter, error) {
	log.Println("Connecting to addr ", addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, errors.New("Dialing Failed")
	}
	return bufio.NewReadWriter(bufio.NewReader(conn),
		bufio.NewWriter(conn)), nil
}

type HandleFunc func(*bufio.ReadWriter)

func Listen() error {
	listener, _ := net.Listen("tcp", Port)
	for {
		log.Println("waiting for a connection")
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting a connection")
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	rw := bufio.NewReadWriter(bufio.NewReader(conn),
		bufio.NewWriter(conn))
	defer conn.Close()

	log.Println("handling messages")

	userProfile, error := handleRegistration(rw)

	if error != nil {
		fmt.Println("we are here")
		rw.WriteString("Handle Already Taken\n")
		rw.Flush()
		return
	}
	defer userProfile.close()

	for {
		cmd, err := rw.ReadString('\n')
		if err != nil {
			log.Println("Terminated or Errored")
			return
		}
		cmd = strings.Trim(cmd, "\n")
		if cmd == "SHOW USERS" {
			users := getAllOnlineUsers()
			rw.WriteString(users + "\n")
			rw.Flush()
			continue
		}
		fmt.Println("reading", cmd)
		handleMessages(userProfile, cmd)
	}
}

func (profile *UserProfile) close() {
	delete(connections, profile.handle)
	profile = nil
}

func handleMessages(userProfile *UserProfile, message string) {

	tokens := strings.SplitN(message, ":", 2)
	if len(tokens) < 2 {
		rw := userProfile.rw
		rw.WriteString("Bad Input or user does not exist\n")
		rw.Flush()
		return
	}
	to, message := tokens[0], tokens[1]
	fmt.Println("to message ", to, message)

	toUserProfile, ok := connections[to]
	if !ok {
		rw := userProfile.rw
		rw.WriteString("Bad Input or user does not exist\n")
		rw.Flush()
		return
	}

	rw := toUserProfile.rw
	rw.WriteString(userProfile.handle + " : " + message + "\n")
	rw.Flush()
}

func handleRegistration(rw *bufio.ReadWriter) (userProfile *UserProfile, e error) {
	cmd, err := rw.ReadString('\n')
	if err != nil {
		log.Println("Terminated or Errored")
		return nil, err
	}
	cmd = strings.Trim(cmd, "\n")
	err = NewUserProfile(cmd)
	if err != nil {
		return nil, err
	}
	userProfile = connections[cmd]
	userProfile.rw = rw

	fmt.Println(connections)

	rw.WriteString(getAllOnlineUsers() + "\n")
	rw.Flush()

	return connections[cmd], nil
}

func clientScanner(rw *bufio.ReadWriter) {
	// TODO kill this when client kills itself
	inputScanner := bufio.NewScanner(os.Stdin)
	for inputScanner.Scan() {
		input := inputScanner.Text()
		rw.WriteString(input + "\n")
		rw.Flush()
	}
}

func client(ip string) {
	rw, _ := Open(ip + Port)
	fmt.Println("Choose a handle, be cool")
	go clientScanner(rw)
	for {
		response, _ := rw.ReadString('\n')
		response = strings.Trim(response, "\n")
		fmt.Println(response)
		if response == "Handle Already Taken" {
			break
		}
	}
}

func server() {
	Listen()
}

func main() {
	connect := flag.String("connect", "", "IP addr")
	flag.Parse()
	if *connect != "" {
		client(*connect)
	} else {
		server()
	}
}
