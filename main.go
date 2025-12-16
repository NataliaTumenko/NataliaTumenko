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
		resp, err := http.Get(serverURL)
		if err != nil {
			errorCount++
			goto checkError
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			errorCount++
			goto checkError
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errorCount++
			goto checkError
		}

		data := strings.TrimSpace(string(body))
		fields := strings.Split(data, ",")
		if len(fields) != 7 {
			errorCount++
			goto checkError
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
			goto checkError
		}

		// --- Успех: сбрасываем счётчик ошибок ---
		errorCount = 0

		// --- Проверки порогов ---

		// Load Average
		if loadAvg > 30 {
			fmt.Printf("Load Average is too high: %g\n", loadAvg)
		}

		// Memory (80%)
		if memTotal > 0 {
			usage := (memUsed / memTotal) * 100
			if usage > 80 {
				fmt.Printf("Memory usage too high: %.0f%%\n", usage)
			}
		}

		// Disk (90%)
		if diskTotal > 0 {
			usedPerc := (diskUsed / diskTotal) * 100
			if usedPerc > 90 {
				freeBytes := diskTotal - diskUsed
				freeMB := freeBytes / (1024 * 1024)
				fmt.Printf("Free disk space is too low: %.0f Mb left\n", freeMB)
			}
		}

		// Network (90%)
		if netTotal > 0 {
			usedPerc := (netUsed / netTotal) * 100
			if usedPerc > 90 {
				freeBytesPerSec := netTotal - netUsed
				// байты/сек → мегабиты/сек: *8 / 1_000_000
				freeMbitPerSec := freeBytesPerSec * 8 / 1_000_000.0
				fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", freeMbitPerSec)
			}
		}

		time.Sleep(interval)
		continue

	checkError:
		if errorCount >= maxErrors {
			fmt.Println("Unable to fetch server statistic.")
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