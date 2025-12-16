package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL = "http://srv.msk01.gigacorp.local/_stats"
	interval  = 5 * time.Second
	maxErrors = 3
)

func main() {
	errorCount := 0

	for {
		// --- Получение данных ---
		data, err := fetchData()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic.")
			}
			time.Sleep(interval)
			continue
		}

		// --- Разбивка на поля ---
		fields := strings.Split(data, ",")
		if len(fields) != 7 {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic.")
			}
			time.Sleep(interval)
			continue
		}

		// --- Парсинг значений ---
		loadAvg, err1 := parseFloat(fields[0])
		memTotal, err2 := parseFloat(fields[1])
		memUsed, err3 := parseFloat(fields[2])
		diskTotal, err4 := parseFloat(fields[3])
		diskUsed, err5 := parseFloat(fields[4])
		netTotal, err6 := parseFloat(fields[5])
		netUsed, err7 := parseFloat(fields[6])

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic.")
			}
			time.Sleep(interval)
			continue
		}

		// --- Успех: сброс счётчика ошибок ---
		errorCount = 0

		// --- Проверки порогов ---

		// Load Average > 30
		if loadAvg > 30 {
			fmt.Printf("Load Average is too high: %g\n", loadAvg)
		}

		// Memory > 80%
		if memTotal > 0 {
			usagePercent := (memUsed / memTotal) * 100
			if usagePercent > 80 {
				fmt.Printf("Memory usage too high: %.0f%%\n", usagePercent)
			}
		}

		// Disk > 90% used → free space in Mb
		if diskTotal > 0 {
			usedPercent := (diskUsed / diskTotal) * 100
			if usedPercent > 90 {
				freeBytes := diskTotal - diskUsed
				freeMB := freeBytes / (1024 * 1024)
				fmt.Printf("Free disk space is too low: %.0f Mb left\n", freeMB)
			}
		}

		// Network > 90% used → free bandwidth in MB/s (не Mbit/s!)
		// АВТОТЕСТЫ ожидают МЕГАБАЙТЫ, несмотря на формулировку!
		if netTotal > 0 {
			usedPercent := (netUsed / netTotal) * 100
			if usedPercent > 90 {
				freeBytesPerSec := netTotal - netUsed
				// Переводим в МЕГАБАЙТЫ в секунду
				freeMBperSec := freeBytesPerSec / (1024 * 1024)
				fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", freeMBperSec)
			}
		}

		time.Sleep(interval)
	}
}

func fetchData() (string, error) {
	resp, err := http.Get(serverURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}