package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func connectAndWork(modeChoice, ip, port, dataChoice string) string {
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
			conn.Write([]byte("mode:push:" + dataChoice))
		default:
			modeChoice = "1"
			conn.Write([]byte("mode:manual"))
			fmt.Println("Режим: ручной")
		}

		if modeChoice == "1" || modeChoice == "2" {
			fmt.Println("Введите команду (mode/m, switch/s, d/data, exit/e) или нажмите Enter для запроса информации...")
		}

		done := make(chan struct{})
		command := make(chan string)
		stopInput := make(chan struct{})

		go func() {
			defer func() {
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
				fmt.Println("Введите команду (mode/m, switch/s, d/data, exit/e) или нажмите Enter для запроса информации...")
			}
		}()

		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				select {
				case <-stopInput:
					return
				default:
					text, _ := reader.ReadString('\n')
					text = strings.TrimSpace(text)

					switch text {
					case "mode", "m", "switch", "s", "exit", "e", "data", "d":
						command <- text
					case "":
						if modeChoice == "1" || modeChoice == "2" {
							command <- "get_bytes"
						} else {
							fmt.Println("В режиме push Enter ничего не отправляет. Используйте команды mode/m, switch/s, exit/e.")
						}
					default:
						fmt.Println("Неизвестная команда.")
					}
				}
			}
		}()

		for {
			select {
			case <-done:
				return "reconnect"
			case cmd := <-command:
				var request string
				if cmd == "get_bytes" {
					switch dataChoice {
					case "1":
						request = "get_bytes:uptime"
					case "2":
						request = "get_bytes:tz"
					case "3":
						request = "get_bytes:both"
					}
					_, err := conn.Write([]byte(request))
					if err != nil {
						fmt.Println("Ошибка отправки запроса:", err)
						return "reconnect"
					}
				} else {
					conn.Close()
					return cmd
				}
			}
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
	chooseServer:
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
			goto chooseData1
		} else if choice == "2" {
			ip = "127.0.0.1"
			port = "6000"
			goto chooseData2
		} else {
			fmt.Println("Завершение работы клиента.")
			break
		}

	chooseData1:
		fmt.Println("Какие данные вы хотите получить?")
		fmt.Println("1) Продолжительность текущего сеанса работы")
		fmt.Println("2) Текущий часовой пояс")
		fmt.Println("3) Все данные")
		dataChoice, _ := reader.ReadString('\n')
		dataChoice = strings.TrimSpace(dataChoice)

	chooseData2:
		fmt.Println("Какие данные вы хотите получить?")
		fmt.Println("1) Количество и процент свободной физической памяти")
		fmt.Println("2) Время работы серверного процесса")
		fmt.Println("3) Все данные")
		dataChoices, _ := reader.ReadString('\n')
		dataChoices = strings.TrimSpace(dataChoices)

	reconnect:
		fmt.Println("Выберите режим: 1) ручной 2) периодический 3) push")
		modeChoice, _ := reader.ReadString('\n')
		modeChoice = strings.TrimSpace(modeChoice)

	modeLoop:
		for {
			cmd := connectAndWork(modeChoice, ip, port, dataChoice)

			switch cmd {
			case "data", "d":
				fmt.Println("Смена данных")
				switch modeChoice {
				case "1":
					goto chooseData1
				case "2":
					goto chooseData2
				}
			case "mode", "m":
				fmt.Println("Смена режима.")
				goto reconnect
			case "switch", "s":
				fmt.Println("Смена сервера.")
				goto chooseServer
			case "exit", "e":
				fmt.Println("Завершение работы клиента.")
				return
			default:
				fmt.Println("Переподключение...")
				time.Sleep(2 * time.Second)
				goto modeLoop
			}
		}
	}
}
