package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	url             = "http://srv.msk01.gigacorp.local/_stats"
	checkInterval   = 5 * time.Second
	maxFailures     = 3
)

func main() {
	failCount := 0

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := client.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
			}
			continue
		}

		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		resp.Body.Close()
		if err != nil && err.Error() != "EOF" {
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
			}
			continue
		}

		body := strings.TrimSpace(string(buf[:n]))
		parts := strings.Split(body, ",")
		if len(parts) != 7 {
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
			}
			continue
		}

		// Сброс счётчика ошибок при успешном парсинге
		failCount = 0

		// Parse all values
		loadAvg, err1 := parseFloat64(parts[0])
		totalMem, err2 := parseFloat64(parts[1])
		usedMem, err3 := parseFloat64(parts[2])
		totalDisk, err4 := parseFloat64(parts[3])
		usedDisk, err5 := parseFloat64(parts[4])
		totalNet, err6 := parseFloat64(parts[5])
		usedNet, err7 := parseFloat64(parts[6])

		if err1 != nil || err2 != nil || err3 != nil ||
			err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
			}
			continue
		}

		// 1. Load Average
		if loadAvg > 30 {
			fmt.Printf("Load Average is too high: %.6g\n", loadAvg)
		}

		// 2. Memory usage > 80%
		if totalMem > 0 {
			memPerc := (usedMem / totalMem) * 100
			if memPerc > 80 {
				fmt.Printf("Memory usage too high: %d%%\n", int(memPerc))
			}
		}

		// 3. Free disk space < 10% → output free MB
		if totalDisk > 0 {
			freeDisk := totalDisk - usedDisk
			freePerc := (freeDisk / totalDisk) * 100
			if freePerc < 10 {
				freeMB := int64(freeDisk / (1024 * 1024))
				fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
			}
		}

		// 4. Network usage > 90% → output free bandwidth in Mbit/s
		if totalNet > 0 {
			netPerc := (usedNet / totalNet) * 100
			if netPerc > 90 {
				freeBps := totalNet - usedNet                      // bytes per second
				freeMbitps := (freeBps * 8) / (1000 * 1000)        // convert to Mbit/s
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", int(freeMbitps))
			}
		}
	}
}

func parseFloat64(s string) (float64, error) {
	// Удаляем возможные пробелы
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}