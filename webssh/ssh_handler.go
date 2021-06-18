package webssh

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"net/http"
	"strconv"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Webssh(c *gin.Context) {
	wsConn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)
	defer wsConn.Close()
	cols, err := strconv.Atoi(c.DefaultQuery("cols", "100"))
	if err != nil {
		log.Error(err)
		return
	}
	rows, err := strconv.Atoi(c.DefaultQuery("rows", "50"))
	if err != nil {
		log.Error(err)
		return
	}

	sshAddr := c.DefaultQuery("sshaddr", "localhost:22")
	sshPassword := c.DefaultQuery("sshpassword", "password")
	sshUser := c.DefaultQuery("sshuser", "root")

	sshClient, _ := CreateSSHClient(sshUser, sshPassword, sshAddr)
	defer sshClient.Close()
	sshWsSessionConfig := &SshWsSessionConfig{
		cols:      cols,
		rows:      rows,
		sshClient: sshClient,
		wsConn:    wsConn,
	}
	sws, _ := NewSshWsSession(sshWsSessionConfig)
	defer sws.Close()
	quitChan := make(chan bool, 3)
	sws.Start(quitChan)
	go sws.Wait(quitChan)

	<-quitChan

}

func CreateSSHClient(sshUser, sshPassword, sshAddr string) (*ssh.Client, error) {
	if sshUser == "" {
		return nil, errors.New("sshUser must not be empty")
	}
	config := &ssh.ClientConfig{
		Timeout:         time.Second * 3,
		User:            sshUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{ssh.Password(sshPassword)},
	}
	return ssh.Dial("tcp", sshAddr, config)
}
