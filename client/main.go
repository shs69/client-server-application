package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func connectAndWork(ip string, port string) bool {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return false
	}
	defer conn.Close()

	fmt.Println("Подключено к", ip+":"+port)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Выберите режим: 1) ручной 2) периодический 3) push")
	modeChoice, _ := reader.ReadString('\n')
	modeChoice = strings.TrimSpace(modeChoice)

	switch modeChoice {
	case "2":
		_, err := conn.Write([]byte("mode:periodic"))
		if err != nil {
			fmt.Println("Ошибка отправки режима")
			return true
		}
	case "3":
		_, err := conn.Write([]byte("mode:push"))
		if err != nil {
			fmt.Println("Ошибка отправки режима")
			return true
		}
	default:
		_, err := conn.Write([]byte("mode:manual"))
		if err != nil {
			fmt.Println("Ошибка отправки режима")
			return true
		}
		fmt.Println("Режим: ручной")
	}

	done := make(chan struct{})
	input := make(chan struct{})
	stopInput := make(chan struct{})

	go func() {
		defer func() {
			fmt.Println("Соединение закрыто (сервер отключился или произошла ошибка).")
			close(done)
			close(stopInput)
		}()

		buf := make([]byte, 512)
		for {
			n, err := conn.Read(buf)
			if err != nil || n == 0 {
				return
			}
			fmt.Println("От сервера:\n", string(buf[:n]))
			if modeChoice != "3" {
				fmt.Println("Нажмите Enter для запроса информации (или Ctrl+C для завершения работы)...")
			}
		}
	}()

	if modeChoice == "1" || modeChoice == "2" {
		fmt.Println("Нажмите Enter для запроса информации (или Ctrl+C для завершения работы)...")
		go func() {
			for {
				select {
				case <-stopInput:
					return
				default:
					text, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					text = strings.TrimSpace(text)
					if text == "" {
						input <- struct{}{}
					}
				}
			}
		}()

		for {
			select {
			case <-done:
				return true
			case <-input:
				_, err := conn.Write([]byte("get_bytes"))
				if err != nil {
					fmt.Println("Ошибка отправки запроса:", err)
					return true
				}
			}
		}
	} else {
		<-done
	}

	fmt.Println("Клиент завершил работу.")
	return true
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Connect to: 1) Server1 (port 8081) 2) Server2 (port 6000) 3) Both")
	choice, _ := reader.ReadString('\n')
	ip, port := "", ""

	if choice[0] == '1' {
		ip = "127.0.0.1"
		port = "8081"
	} else if choice[0] == '2' {
		ip = "127.0.0.1"
		port = "6000"
	}

	for retry := 0; retry <= 5; retry++ {
		fmt.Println("Подключение")
		success := connectAndWork(ip, port)
		if success {
			fmt.Println("Завершено")
		}
		time.Sleep(2 * time.Second)
	}
}
