package main

import (
	"fmt"
	"net"
	"os"
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

func handleClient(conn net.Conn) {

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Ошибка в процессе разрыва соединения")
		} else {
			fmt.Fprintf(os.Stdout, "Клиент %s отключен \n", conn.RemoteAddr().String())
		}
	}(conn)

	fmt.Println("Подключен новый клиент:", conn.RemoteAddr().String())

	lastDuration := ""

	for {
		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		req := string(buf[:n])
		fmt.Println(req)

		if req == "mode:periodic" {
			for {
				buf := make([]byte, 512)
				n, err = conn.Read(buf)
				printError("", err)
				fmt.Println(string(buf[:n]))
				if string(buf[:n]) == "get_bytes" {
					durationStr := fmt.Sprintf("Текущее время: %s", time.Now().Format("02-01-2006 15:04:05"))
					info := currentLocation() + uptime()
					if info != lastDuration {
						durationStr += info
						conn.Write([]byte(durationStr))
						lastDuration = info
					} else {
						conn.Write([]byte("Данные не изменились"))
					}
				}
			}
		} else if req == "mode:push" {
			for {
				durationStr := fmt.Sprintf("Текущее время: %s", time.Now().Format("02-01-2006 15:04:05"))
				info := currentLocation() + uptime()
				if info != lastDuration {
					durationStr += info
					conn.Write([]byte(durationStr))
					lastDuration = info
				}
			}
		} else if req == "mode:manual" {
			for {
				buf := make([]byte, 512)
				n, err = conn.Read(buf)
				printError("", err)
				durationStr := currentLocation()
				durationStr += uptime()
				conn.Write([]byte(durationStr))
			}
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
