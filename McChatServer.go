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
	Port               = ":14610"
	SHOW_USERS_COMMAND = "SHOW USERS"
	DUPLICATE_HANDLE   = "Handle Already In Use"
)

var (
	connections map[string]*UserProfile = make(map[string]*UserProfile)
)

func NewUserProfile(name string) error {
	_, ok := connections[name]
	if ok {
		return errors.New(DUPLICATE_HANDLE)
	}

	connections[name] = &UserProfile{
		handle: name,
	}
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
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, errors.New("Dialing Failed")
	}

	return bufio.NewReadWriter(bufio.NewReader(conn),
		bufio.NewWriter(conn)), nil
}

func Listen() error {
	listener, err := net.Listen("tcp", Port)
	if err != nil {
		return errors.New("Not able to accept connections atm")
	}

	for {
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

	userProfile, error := handleRegistration(rw)

	if error != nil {
		rw.WriteString(error.Error() + "\n")
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

		if cmd == SHOW_USERS_COMMAND {
			users := getAllOnlineUsers()
			rw.WriteString(users + "\n")
			rw.Flush()
		} else if ok := validChatMessage(cmd); ok {
			handleMessages(userProfile, cmd)
		} else {
			rw.WriteString("Please Enter ValidUser: Message\n")
			rw.Flush()
		}
	}
}

func Trim(str string) string {
	return strings.Trim(str, " \n")
}

func validChatMessage(message string) bool {
	tokens := strings.SplitN(message, ":", 2)

	if len(tokens) != 2 {
		return false
	}

	to, message := Trim(tokens[0]), Trim(tokens[1])
	_, ok := connections[to]

	return ok && len(message) > 0
}

func (profile *UserProfile) close() {
	delete(connections, profile.handle)
	*profile = nil
}

func handleMessages(userProfile *UserProfile, message string) {
	tokens := strings.SplitN(message, ":", 2)
	to, message := Trim(tokens[0]), Trim(tokens[1])
	toUserProfile := connections[to]

	rw := toUserProfile.rw
	rw.WriteString(userProfile.handle + " : " + message + "\n")
	rw.Flush()
}

func handleRegistration(rw *bufio.ReadWriter) (userProfile *UserProfile,
	e error) {
	cmd, err := rw.ReadString('\n')
	if err != nil {
		return nil, err
	}

	cmd = Trim(cmd)
	err = NewUserProfile(cmd)
	if err != nil {
		return nil, err
	}

	userProfile = connections[cmd]
	userProfile.rw = rw

	rw.WriteString(getAllOnlineUsers() + "\n")
	rw.Flush()

	return connections[cmd], nil
}

func clientScanner(rw *bufio.ReadWriter, done chan bool) {
	inputScanner := bufio.NewScanner(os.Stdin)
	for inputScanner.Scan() {
		select {
		case <-done:
			return
		default:
		}

		input := Trim(inputScanner.Text())
		rw.WriteString(input + "\n")
		rw.Flush()
	}
}

func client(ip string) {

	rw, _ := Open(ip + Port)
	fmt.Println("Choose a handle, be cool")

	done := make(chan bool)
	go clientScanner(rw, done)

	for {
		response, err := rw.ReadString('\n')
		response = strings.Trim(response, "\n")
		fmt.Println(response)
		if err != nil || response == DUPLICATE_HANDLE {
			done <- true
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
