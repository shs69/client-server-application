package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func connectAndWork(modeChoice string, ip string, port string, reader *bufio.Reader) bool {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return false
	}
	defer conn.Close()

	fmt.Println("Подключено к", ip+":"+port)

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
		modeChoice = "1"
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

	for {
		fmt.Println("Выберите сервер для подключения: ")
		fmt.Println("1) Сервер №1 (порт 8081) 2) Сервер №2 (порт 6000) 3) К обоим одновременно")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		ip, port := "", ""

		if choice == "1" {
			ip = "127.0.0.1"
			port = "8081"
		} else if choice == "2" {
			ip = "127.0.0.1"
			port = "6000"
		} else if choice == "3" {
			return
		}

		for retry := 0; retry <= 5; retry++ {
			fmt.Println("Подключение")
			fmt.Println("Выберите режим: 1) ручной 2) периодический 3) push")

			modeChoice, _ := reader.ReadString('\n')
			modeChoice = strings.TrimSpace(modeChoice)
			success := connectAndWork(modeChoice, ip, port, reader)
			if success {
				fmt.Println("Завершено")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
