package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

func printError(text string, err error) {
	if err != nil {
		if text != "" {
			fmt.Println(text, err)
		}
		return
	}
}

func startServer(port string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))

	if err != nil {
		panic("Сервер уже запущен или указанный порт занят")
	}

	defer func(ln net.Listener) {
		err := ln.Close()
		printError("Ошибка в процессе разрыва соединения: %w", err)
	}(ln)

	fmt.Println("Сервер №1 запущен и слушает порт", port)

	for {
		conn, err := ln.Accept()
		printError("Ошибка во время подключения клиента:", err)
		go handleClient(conn)
	}
}

func uptime() string {
	out, err := exec.Command("uptime").Output()
	if err != nil {
		return fmt.Sprint("Ошибка вызова команды", err)
	}
	out = []byte(strings.Split(string(out), ",")[0][9:])
	return fmt.Sprintf("\n Продолжительность текущего сеанса работы:%s", string(out))
}

func currentLocation() string {
	zone, offset := time.Now().Zone()
	durationStr := fmt.Sprintf("\n Текущий часовой пояс: %s %d", zone, offset/3600)
	return durationStr
}

func currentTime() string {
	return fmt.Sprintf("Текущее время: %s", time.Now().Format("02-01-2006 15:04:05"))
}

func getRightInfo(typeInfo string) string {
	var currentInfo string
	switch typeInfo {
	case "1":
		currentInfo += uptime()
	case "2":
		currentInfo += currentLocation()
	case "3":
		currentInfo += uptime() + currentLocation()
	}

	return currentInfo
}

func handleClient(conn net.Conn) {
	defer func() {
		err := conn.Close()
		if err != nil {
			fmt.Println("Ошибка в процессе разрыва соединения")
		} else {
			fmt.Printf("Клиент %s отключен\n", conn.RemoteAddr().String())
		}
	}()

	fmt.Println("Подключен новый клиент:", conn.RemoteAddr().String())

	lastDuration := ""
	mode := ""
	buf := make([]byte, 512)

	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Ошибка чтения режима:", err)
		return
	}
	mode = strings.TrimSpace(string(buf[:n]))
	fmt.Println("Выбран режим:", mode)

	for {
		switch mode {
		case "mode:periodic":
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("Ошибка чтения в режиме periodic:", err)
				return
			}

			cmd := strings.TrimSpace(string(buf[:n]))
			fmt.Println("Periodic cmd:", cmd)
			cmdArray := strings.Split(cmd, ":")
			reqStr, modeData := cmdArray[0], cmdArray[1]

			if reqStr == "get_bytes" {
				info := getRightInfo(modeData)
				if info != lastDuration {
					_, err := conn.Write([]byte(currentTime() + info))
					if err != nil {
						fmt.Println("Ошибка записи в режиме periodic:", err)
						return
					}
					lastDuration = info
				} else {
					_, err := conn.Write([]byte("Данные не изменились"))
					if err != nil {
						fmt.Println("Ошибка записи в режиме periodic:", err)
						return
					}
				}
			} else {
				fmt.Println("Неизвестная команда в режиме periodic:", cmd)
			}

		case "mode:push:1", "mode:push:2", "mode:push:3":
			fmt.Println()
			info := getRightInfo(strings.Split(mode, ":")[2])
			if info != lastDuration {
				_, err := conn.Write([]byte(currentTime() + info))
				if err != nil {
					fmt.Println("Ошибка записи в режиме push:", err)
					return
				}
				lastDuration = info
			}
			time.Sleep(2 * time.Second)

		case "mode:manual":
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("Ошибка чтения в режиме manual:", err)
				return
			}
			cmd := strings.TrimSpace(string(buf[:n]))
			fmt.Println("Periodic cmd:", cmd)
			cmdArray := strings.Split(cmd, ":")
			reqStr, modeData := cmdArray[0], cmdArray[1]

			if reqStr == "get_bytes" {
				_, err = conn.Write([]byte(currentTime() + getRightInfo(modeData)))
				if err != nil {
					fmt.Println("Ошибка записи в режиме manual:", err)
					return
				}
			}

		default:
			fmt.Println("Неизвестный режим, завершаем соединение")
			return
		}
	}
}

func main() {
	port := "8081"

	for {
		err := startServer(port)
		if err != nil {
			fmt.Println("Критическая ошибка. Перезапуск сервера через 2 секунды")
			time.Sleep(2 * time.Second)
		} else {
			break
		}
	}
}
