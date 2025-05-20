package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
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
	startTime := time.Now()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))

	if err != nil {
		panic("Сервер уже запущен или указанный порт занят")
	}

	defer func(ln net.Listener) {
		err := ln.Close()
		printError("Ошибка в процессе разрыва соединения: %w", err)
	}(ln)

	fmt.Println("Сервер №2 запущен и слушает порт", port)

	for {
		conn, err := ln.Accept()
		printError("Ошибка во время подключения клиента:", err)
		go handleClient(conn, startTime)
	}
}

func timeServer(startTime time.Time) string {
	return fmt.Sprintf("\n Продолжительность текущего сеанса работы:%d", int(time.Since(startTime).Seconds()))
}

func freeMem() string {
	switch runtime.GOOS {
	case "darwin":
		command := "vm_stat | awk " +
			"'/free/ {free_bytes=$3 * 4096} " +
			"/active/ {active_bytes=$3 * 4096} " +
			"/inactive/ {inactive_bytes=$3 * 4096} " +
			"/speculative/ {speculative_bytes=$3 * 4096} " +
			"END {total_mem=(free_bytes + active_bytes + inactive_bytes + speculative_bytes); " +
			"printf(\"Свободно: %.1fGB (%.1f%%)\\n\", free_bytes/1024/1024/1024, (free_bytes/total_mem)*100)}'"
		out, err := exec.Command(command).Output()
		if err != nil {
			return fmt.Sprint("Ошибка вызова команды", err)
		}
		return string(out)
	case "linux":
		out, err := exec.Command("free -h | awk '/^Mem:/ {printf(\"Свободно: %s (%.1f%%)\\n\", $4, ($4/$2)*100)}'\n").Output()
		if err != nil {
			return fmt.Sprint("Ошибка вызова команды", err)
		}
		return string(out)
	}

	zone, offset := time.Now().Zone()
	durationStr := fmt.Sprintf("\n Свободная физическая память : %s %d", zone, offset/3600)
	return durationStr
}

func currentTime() string {
	return fmt.Sprintf("Текущее время: %s", time.Now().Format("02-01-2006 15:04:05"))
}

func getRightInfo(typeInfo string, startTime time.Time) string {
	var currentInfo string
	switch typeInfo {
	case "servertime":
		currentInfo += timeServer(startTime)
	case "freemem":
		currentInfo += freeMem()
	case "both":
		currentInfo += timeServer(startTime) + freeMem()
	}

	return currentInfo
}

func handleClient(conn net.Conn, startTime time.Time) {
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
				info := getRightInfo(modeData, startTime)
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
			info := getRightInfo(strings.Split(mode, ":")[2], startTime)
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
				_, err = conn.Write([]byte(currentTime() + getRightInfo(modeData, startTime)))
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
	port := "6060"

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
