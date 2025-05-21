package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type ServerConfig struct {
	IP    string
	Port  string
	Label string
	Data  string
}

func connectAndWork(servers []ServerConfig, mode string) string {
	conns := make([]net.Conn, len(servers))
	for i, srv := range servers {
		conn, err := net.Dial("tcp", srv.IP+":"+srv.Port)
		if err != nil {
			fmt.Printf("Ошибка подключения к %s (%s:%s): %v\n", srv.Label, srv.IP, srv.Port, err)
			continue
		}
		conns[i] = conn
		fmt.Printf("Подключено к %s (%s:%s)\n", srv.Label, srv.IP, srv.Port)

		var modeStr string
		switch mode {
		case "2":
			modeStr = "mode:periodic"
		case "3":
			modeStr = "mode:push:" + srv.Data
		default:
			modeStr = "mode:manual"
			fmt.Printf("Режим: ручной для %s\n", srv.Label)
		}
		conn.Write([]byte(modeStr))
	}

	done := make(chan struct{})

	for i, conn := range conns {
		if conn == nil {
			return "reconnect"
		}
		go func(i int, c net.Conn, srv ServerConfig) {
			buf := make([]byte, 1024)
			for {
				n, err := c.Read(buf)
				if err != nil {
					fmt.Printf("Соединение с %s (%s:%s) закрыто.\n", srv.Label, srv.IP, srv.Port)
					c.Close()
					done <- struct{}{}
					break
				}
				fmt.Printf("\n[%s:%s] От %s :\n%s\n", srv.IP, srv.Port, srv.Label, string(buf[:n]))
				if len(servers) != 2 {
					fmt.Println("\nВведите команду (mode/m, switch/s, d/data, exit/e) или нажмите Enter для запроса информации:")
				} else {
					if i == 1 {
						fmt.Println("\nВведите команду (mode/m, switch/s, d/data, exit/e) или нажмите Enter для запроса информации:")
					}
				}
			}
		}(i, conn, servers[i])
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nВведите команду (mode/m, switch/s, d/data, exit/e) или нажмите Enter для запроса информации:")
	for {
		select {
		case <-done:
			return "reconnect"
		default:
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)

			if text == "" && (mode == "1" || mode == "2") {
				for i, c := range conns {
					if c != nil {
						var request string
						request = "get_bytes:" + servers[i].Data + "\n"
						_, err := c.Write([]byte(request))
						if err != nil {
							fmt.Printf("Ошибка отправки запроса серверу %s: %v\n", servers[i].Label, err)
							return "reconnect"
						}
					}
				}
			} else {
				switch text {
				case "exit", "e":
					fmt.Println("Завершение работы клиента.")
					for _, c := range conns {
						if c != nil {
							c.Close()
						}
					}
					os.Exit(0)
				case "mode", "m":
					for _, c := range conns {
						if c != nil {
							c.Close()
						}
					}
					return "mode"
				case "switch", "s":
					for _, c := range conns {
						if c != nil {
							c.Close()
						}
					}
					return "switch"
				case "data", "d":
					for _, c := range conns {
						if c != nil {
							c.Close()
						}
					}
					return "data"
				default:
					fmt.Println("Неизвестная команда.")
				}
			}
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	//ip := "127.0.0.1"
	dockerIP := "host.docker.internal"

	for {
	chooseServer:
		fmt.Println("\nВыберите сервер для подключения:")
		fmt.Println("1) Только Сервер №1 (порт 8081)")
		fmt.Println("2) Только Сервер №2 (порт 6060)")
		fmt.Println("3) Подключиться к обоим серверам")
		fmt.Println("4) Выход")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "4" {
			fmt.Println("Завершение работы клиента.")
			return
		}

		var servers []ServerConfig

		switch choice {
		case "1":
			dataChoice := chooseDataForServer(reader, "1", "Сервер №1 (127.0.0.1:8081)")
			servers = []ServerConfig{
				{IP: dockerIP, Port: "8081", Label: "Сервер №1", Data: dataChoice},
			}
		case "2":
			dataChoice := chooseDataForServer(reader, "2", "Сервер №2 (127.0.0.1:6060)")
			servers = []ServerConfig{
				{IP: dockerIP, Port: "6060", Label: "Сервер №2", Data: dataChoice},
			}
		case "3":
			dataChoice1 := chooseDataForServer(reader, "1", "Сервер №1 (127.0.0.1:8081)")
			dataChoice2 := chooseDataForServer(reader, "2", "Сервер №2 (127.0.0.1:6060)")
			servers = []ServerConfig{
				{IP: dockerIP, Port: "8081", Label: "Сервер №1", Data: dataChoice1},
				{IP: dockerIP, Port: "6060", Label: "Сервер №2", Data: dataChoice2},
			}
		default:
			fmt.Println("Некорректный выбор. Попробуйте ещё раз.")
			continue
		}

	reconnect:
		fmt.Println("Выберите режим: 1) Ручной 2) Периодический 3) Push")
		modeChoice, _ := reader.ReadString('\n')
		modeChoice = strings.TrimSpace(modeChoice)

	modeLoop:
		for {
			cmd := connectAndWork(servers, modeChoice)

			switch cmd {
			case "data":
				fmt.Println("Смена данных")
				for i := range servers {
					servers[i].Data = chooseDataForServer(reader, fmt.Sprintf("%d", i+1), servers[i].Label)
				}
			case "mode":
				fmt.Println("Смена режима.")
				goto reconnect
			case "switch":
				fmt.Println("Смена сервера.")
				goto chooseServer
			default:
				fmt.Println("Переподключение...")
				time.Sleep(1 * time.Second)
				goto modeLoop
			}
		}
	}
}

func chooseDataForServer(reader *bufio.Reader, serverNum string, serverLabel string) string {
	fmt.Println(serverNum)
	if serverNum == "1" {
		fmt.Printf("Какие данные вы хотите получать от %s?\n", serverLabel)
		fmt.Println("1) Продолжительность текущего сеанса работы")
		fmt.Println("2) Текущий часовой пояс")
		fmt.Println("3) Все данные")
	} else if serverNum == "2" {
		fmt.Printf("Какие данные вы хотите получать от %s?\n", serverLabel)
		fmt.Println("1) Количество и процент свободной физической памяти")
		fmt.Println("2) Время работы серверного процесса")
		fmt.Println("3) Все данные")
	}

	dataChoice, _ := reader.ReadString('\n')
	dataChoice = strings.TrimSpace(dataChoice)
	return dataChoice
}
