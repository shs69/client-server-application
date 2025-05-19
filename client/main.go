package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func connectAndWork(modeChoice, ip, port string) {
	for {
		conn, err := net.Dial("tcp", ip+":"+port)
		if err != nil {
			fmt.Println("Ошибка подключения:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Println("Подключено к", ip+":"+port)

		switch modeChoice {
		case "2":
			conn.Write([]byte("mode:periodic"))
		case "3":
			conn.Write([]byte("mode:push"))
		default:
			modeChoice = "1"
			conn.Write([]byte("mode:manual"))
			fmt.Println("Режим: ручной")
		}

		done := make(chan struct{})
		input := make(chan struct{})
		stopInput := make(chan struct{})

		go func() {
			defer func() {
				fmt.Println("Соединение закрыто.")
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
					fmt.Println("Нажмите Enter для запроса информации...")
				}
			}
		}()

		if modeChoice == "1" || modeChoice == "2" {
			fmt.Println("Нажмите Enter для запроса информации...")
			go func() {
				reader := bufio.NewReader(os.Stdin)
				for {
					select {
					case <-stopInput:
						return
					default:
						text, _ := reader.ReadString('\n')
						if strings.TrimSpace(text) == "" {
							input <- struct{}{}
						}
					}
				}
			}()

			for {
				select {
				case <-done:
					goto reconnect
				case <-input:
					_, err := conn.Write([]byte("get_bytes"))
					if err != nil {
						fmt.Println("Ошибка отправки запроса:", err)
						goto reconnect
					}
				}
			}
		} else {
			<-done
		}

	reconnect:
		conn.Close()
		fmt.Println("Переподключение через 2 секунды...")
		time.Sleep(2 * time.Second)
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nВыберите сервер для подключения:")
		fmt.Println("1) Сервер №1 (порт 8081)")
		fmt.Println("2) Сервер №2 (порт 6000)")
		fmt.Println("3) Выход")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		ip, port := "", ""
		if choice == "1" {
			ip = "127.0.0.1"
			port = "8081"
		} else if choice == "2" {
			ip = "127.0.0.1"
			port = "6000"
		} else {
			break
		}

		fmt.Println("Выберите режим: 1) ручной 2) периодический 3) push")
		modeChoice, _ := reader.ReadString('\n')
		modeChoice = strings.TrimSpace(modeChoice)

		connectAndWork(modeChoice, ip, port)
	}
}
