package tellogger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CloseFileLogger func()
type Logger func(format string, args ...any)

var jsonExtractor = regexp.MustCompile(`(?s)(\{.*\}|\[.*\])`)

func formatTelegramJSON(rawJSON string) string {
	var obj any
	err := json.Unmarshal([]byte(rawJSON), &obj)
	if err != nil {
		return rawJSON
	}

	prettyBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return rawJSON
	}

	return string(prettyBytes)
}

func NewLogger(cmdLog bool) (Logger, CloseFileLogger) {
	logsDir := "logs/telegram"
	cleanlogsDir := "clean-logs/telegram"

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("ERR: Failed to create logs directory: %v", err)
	}

	timeStamp := time.Now().Format("2006-01-02_15-04-05")

	rawFileName := filepath.Join(logsDir, fmt.Sprintf("%s-%s.log", timeStamp, uuid.New().String()))
	rawFile, err := os.OpenFile(rawFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ERR: Failed to create raw log file: %v", err)
	}

	cleanFileName := filepath.Join(cleanlogsDir, fmt.Sprintf("%s-clean.log", timeStamp))
	cleanFile, err := os.OpenFile(cleanFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		rawFile.Close()
		log.Fatalf("ERR: Failed to create clean log file: %v", err)
	}

	rawLogger := log.New(rawFile, "", log.LstdFlags|log.Lmicroseconds)
	cleanLogger := log.New(cleanFile, "", log.LstdFlags|log.Lmicroseconds)
	termLogger := log.New(os.Stdout, "\033[36m[TELEGRAM]\033[0m ", log.Ltime)

	customDebugHandler := func(format string, args ...any) {
		fullMsg := fmt.Sprintf(format, args...)

		rawLogger.Println(fullMsg)

		cleanMsg := strings.ReplaceAll(fullMsg, "\r\n", "\n")
		cleanMsg = strings.ReplaceAll(cleanMsg, "\r", "")

		match := jsonExtractor.FindString(cleanMsg)

		if match != "" {
			prefix := cleanMsg[:strings.Index(cleanMsg, match)]
			prefix = strings.TrimSpace(prefix)

			formattedJSON := formatTelegramJSON(match)

			cleanLogger.Printf("%s\nPayload:\n%s\n%s\n", prefix, formattedJSON, strings.Repeat("-", 80))

			if cmdLog {
				termLogger.Printf("\033[33m%s\033[0m\n%s\n%s\n", prefix, formattedJSON, strings.Repeat("-", 50))
			}
		} else {
			cleanLogger.Println(cleanMsg)
			if cmdLog {
				termLogger.Println(cleanMsg)
			}
		}
	}

	cleanup := func() {
		rawFile.Close()
		cleanFile.Close()
	}

	return customDebugHandler, cleanup
}
