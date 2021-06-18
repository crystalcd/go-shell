package webssh

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"time"
)

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"
)

type SshWsSessionConfig struct {
	cols      int
	rows      int
	sshClient *ssh.Client
	wsConn    *websocket.Conn
}

type SshWsSession struct {
	stdinPipe   io.WriteCloser
	comboOutput *safeBuffer //ssh 终端混合输出
	session     *ssh.Session
	wsConn      *websocket.Conn
}

type wsMsg struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type safeBuffer struct { // 如果直接使用bytes.Buffer会出现输出上一次的结果
	buffer bytes.Buffer
}

func (w *safeBuffer) Write(p []byte) (int, error) {
	return w.buffer.Write(p)
}
func (w *safeBuffer) Bytes() []byte {
	return w.buffer.Bytes()
}
func (w *safeBuffer) Reset() {
	w.buffer.Reset()
}

func NewSshWsSession(swsc *SshWsSessionConfig) (*SshWsSession, error) {
	sshSession, err := swsc.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}
	comboWriter := new(safeBuffer)
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("xterm", swsc.rows, swsc.cols, modes); err != nil {
		return nil, err
	}
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	return &SshWsSession{
		stdinPipe:   stdinP,
		comboOutput: comboWriter,
		session:     sshSession,
		wsConn:      swsc.wsConn,
	}, nil
}

func (sws *SshWsSession) Start(quitChan chan bool) {
	go sws.receiveWsMsg(quitChan)
	go sws.sendComboOutput(quitChan)
}

func (sws *SshWsSession) Wait(quitChan chan bool) {
	defer setQuit(quitChan)
	if err := sws.session.Wait(); err != nil {
		log.WithError(err).Error("ssh session wait failed")
	}
}

func (sws *SshWsSession) receiveWsMsg(exitCh chan bool) {
	wsConn := sws.wsConn
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				log.WithError(err).Error("reading webSocket message failed")
				return
			}
			//unmashal bytes into struct
			msgObj := wsMsg{}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				log.WithError(err).WithField("wsData", string(wsData)).Error("unmarshal websocket message failed")
			}
			switch msgObj.Type {
			case wsMsgResize:
				//handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := sws.session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						log.WithError(err).Error("ssh pty change windows size failed")
					}
				}
			case wsMsgCmd:
				// seng message to pty
				sws.sendWsInput2SshStdinPipe([]byte(msgObj.Cmd))
			}
		}
	}
}

func (sws *SshWsSession) sendComboOutput(exitCh chan bool) {
	wsConn := sws.wsConn
	//tells other go routine quit
	defer setQuit(exitCh)

	tick := time.NewTicker(time.Millisecond * time.Duration(10))
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if sws.comboOutput == nil {
				return
			}
			bs := sws.comboOutput.Bytes()
			if len(bs) > 0 {
				//log.Debugf("comboOutput value=%s",sws.comboOutput)
				//log.Debugf("stdout value=%s",sws.session.Stdout)
				//log.Debugf("err value=%s",sws.session.Stderr)
				err := wsConn.WriteMessage(websocket.TextMessage, bs)
				if err != nil {
					log.WithError(err).Error("ssh sending combo output to webSocket failed")
				}
				sws.comboOutput.Reset()
				//log.Debugf("---------清空后-------------")
				//log.Debugf("comboOutput value=%s",sws.comboOutput)
				//log.Debugf("stdout value=%s",sws.session.Stdout)
				//log.Debugf("err value=%s",sws.session.Stderr)
			}
		case <-exitCh:
			return
		}
	}
}

func (sws *SshWsSession) sendWsInput2SshStdinPipe(cmdBytes []byte) {
	if _, err := sws.stdinPipe.Write(cmdBytes); err != nil {
		log.WithError(err).Error("ws cmd bytes write to ssh.stdin pipe failed")
	}
}

func (sws *SshWsSession) Close() {
	if sws.session != nil {
		sws.session.Close()
	}

	if sws.comboOutput != nil {
		sws.comboOutput = nil
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
