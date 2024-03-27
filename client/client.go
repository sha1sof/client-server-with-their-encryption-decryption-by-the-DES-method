package main

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type Client struct {
	conn       net.Conn
	connected  bool
	serverIP   *walk.LineEdit
	serverPort *walk.LineEdit
	username   *walk.LineEdit
	mainWindow *walk.MainWindow
	chatTE     *walk.TextEdit
	sendTE     *walk.TextEdit
	connectBtn *walk.PushButton
	sendBtn    *walk.PushButton
	encryptCb  *walk.CheckBox
	keyEdit    *walk.LineEdit
}

func (c *Client) connectToServer() error {
	conn, err := net.Dial("tcp", c.serverIP.Text()+":"+c.serverPort.Text())
	if err != nil {
		return err
	}
	c.conn = conn
	c.connected = true
	c.updateUI()
	return nil
}

func (c *Client) disconnectFromServer() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.connected = false
		c.updateUI()
	}
}

func (c *Client) sendMessage() {
	if c.connected {
		msg := c.sendTE.Text()
		if msg != "" {
			if c.encryptCb.Checked() {
				key, err := parseKey(c.keyEdit.Text())
				if err != nil {
					walk.MsgBox(c.mainWindow, "Ошибка", "Неверный ключ шифрования: "+err.Error(), walk.MsgBoxIconError)
					return
				}
				msg = desEncrypt(msg, key)
			}
			_, err := c.conn.Write([]byte(c.username.Text() + ": " + msg))
			if err != nil {
				log.Println("Ошибка отправки сообщения:", err)
				return
			}
			c.sendTE.SetText("")
		}
	}
}

func (c *Client) readLoop() {
	buf := make([]byte, 2048)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Println("Ошибка чтения:", err)
			c.disconnectFromServer()
			return
		}
		msg := string(buf[:n])
		if c.encryptCb.Checked() {
			key, err := parseKey(c.keyEdit.Text())
			if err != nil {
				walk.MsgBox(c.mainWindow, "Ошибка", "Неверный ключ шифрования: "+err.Error(), walk.MsgBoxIconError)
				return
			}
			msgParts := strings.SplitN(msg, ": ", 2)
			if len(msgParts) == 2 {
				msg = msgParts[0] + ": " + desDecrypt(msgParts[1], key)
			}
		}
		c.mainWindow.Synchronize(func() {
			c.chatTE.AppendText(msg + "\r\n")
		})
	}
}

func (c *Client) updateUI() {
	if c.connected {
		c.connectBtn.SetText("Отключиться")
		c.sendBtn.SetEnabled(true)
	} else {
		c.connectBtn.SetText("Подключиться")
		c.sendBtn.SetEnabled(false)
	}
}

func parseKey(keyStr string) ([]byte, error) {
	keyStr = strings.TrimSpace(keyStr)
	if len(keyStr) != 8 {
		return nil, fmt.Errorf("Недопустимый ключ: длина ключа должна быть 7 байт")
	}
	return []byte(keyStr), nil
}

func desEncrypt(text string, key []byte) string {
	block, err := des.NewCipher(key)
	if err != nil {
		log.Fatal(err)
	}

	src := []byte(text)
	src = padding(src, block.BlockSize())
	dst := make([]byte, len(src))

	blockMode := cipher.NewCBCEncrypter(block, key)
	blockMode.CryptBlocks(dst, src)

	return hex.EncodeToString(dst)
}

func desDecrypt(encryptedText string, key []byte) string {
	block, err := des.NewCipher(key)
	if err != nil {
		log.Fatal(err)
	}

	src, err := hex.DecodeString(encryptedText)
	if err != nil {
		log.Fatal(err)
	}

	dst := make([]byte, len(src))

	blockMode := cipher.NewCBCDecrypter(block, key)
	blockMode.CryptBlocks(dst, src)

	dst = unpadding(dst)

	return string(dst)
}

func padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func main() {
	client := &Client{}

	MainWindow{
		AssignTo: &client.mainWindow,
		Title:    "Чат",
		Size:     Size{Width: 600, Height: 400},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					LineEdit{AssignTo: &client.serverIP, Text: "IP"},
					LineEdit{AssignTo: &client.serverPort, Text: "Port"},
					LineEdit{AssignTo: &client.username, Text: "UserName"},
					PushButton{
						AssignTo: &client.connectBtn,
						Text:     "Подключиться",
						OnClicked: func() {
							if !client.connected {
								err := client.connectToServer()
								if err != nil {
									walk.MsgBox(client.mainWindow, "Ошибка", "Не удалось подключиться к серверу", walk.MsgBoxIconError)
									return
								}
								go client.readLoop()
							} else {
								client.disconnectFromServer()
							}
						},
					},
					CheckBox{
						AssignTo: &client.encryptCb,
						Text:     "Шифровать сообщения",
					},
					LineEdit{
						AssignTo: &client.keyEdit,
						Text:     "Ключ",
					},
				},
			},
			TextEdit{AssignTo: &client.chatTE, ReadOnly: true},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					TextEdit{AssignTo: &client.sendTE},
					PushButton{
						AssignTo:  &client.sendBtn,
						Text:      "Отправить",
						OnClicked: client.sendMessage,
					},
				},
			},
		},
	}.Run()
}
