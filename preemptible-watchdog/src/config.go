package main

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"regexp"
	"strconv"
	"time"
)

const envMaintenanceInterval = "MAINTENANCE_INTERVAL"
const envRestartLabel = "RESTART_LABEL"
const envOpsLimit = "OPERATIONS_LIMIT"
const envFolderID = "FOLDER_ID"
const envHoursToRestart = "HOURS_TO_RESTART"
const envLogLevel = "LOG_LEVEL"

type config struct {
	inMaintenance  bool
	restartLabel   string
	opsLimit       int
	folderID       string
	hoursToRestart int
}

var intervalPattern = regexp.MustCompile("^(\\d{1,2}):(\\d{1,2})-(\\d{1,2}):(\\d{1,2})$")
var conf *config
var log *zap.Logger

func init() {
	config := zap.NewProductionConfig()
	config.DisableCaller = true
	config.Level.SetLevel(zap.InfoLevel)
	var err error
	if levelStr := os.Getenv(envLogLevel); len(levelStr) > 0 {
		err = config.Level.UnmarshalText([]byte(levelStr))
	}
	log, _ = config.Build()
	if err != nil {
		log.Warn(fmt.Sprintf("failed to parse log level: %s", err))
	}
}

func initConf() {
	if conf != nil {
		return
	}
	conf = &config{
		inMaintenance:  isMaintenanceInterval(time.Now()),
		restartLabel:   requireEnvStr(envRestartLabel),
		opsLimit:       requireEnvInt(envOpsLimit),
		folderID:       requireEnvStr(envFolderID),
		hoursToRestart: requireEnvInt(envHoursToRestart),
	}
}

func main() {
	now := time.Now()
	for i := 0; i < 24; i++ {
		t := now.Add(time.Duration(i) * time.Hour)
		fmt.Printf("%v in maintenance: %v\n", t, isMaintenanceInterval(t))
	}
}

func isMaintenanceInterval(now time.Time) bool {
	intervalStr := requireEnvStr(envMaintenanceInterval)
	matches := intervalPattern.FindStringSubmatch(intervalStr)
	if len(matches) != 5 {
		log.Panic(fmt.Sprintf("expected hh:mm-hh:mm, got %s", intervalStr))
	}
	startH, _ := strconv.Atoi(matches[1])
	startM, _ := strconv.Atoi(matches[2])
	endH, _ := strconv.Atoi(matches[3])
	endM, _ := strconv.Atoi(matches[4])
	now = now.UTC()
	hour := now.Hour()
	minute := now.Minute()
	if localTimeGTE(startH, startM, endH, endM) {
		// map 23:00-01:00 interval to 23:00-25:00
		endH += 24
		if localTimeGTE(startH, startM, hour, minute) {
			// and map actual time to same interval if needed, e.g.:
			// 00:30 -> 24:30 (inside)
			// 23:30 -> 23:30 (no mapping, inside)
			// 22:59 -> 46:59 (outside)
			// 01:01 -> 25:01 (outside)
			hour += 24
		}
	}
	return localTimeGTE(hour, minute, startH, startM) && localTimeGTE(endH, endM, hour, minute)
}

func localTimeGTE(h1, m1, h2, m2 int) bool {
	return h1 > h2 || h1 == h2 && m1 >= m2
}

func requireEnvStr(envName string) string {
	res := os.Getenv(envName)
	if len(res) == 0 {
		log.Panic(fmt.Sprintf("env var is empty: %s", envName))
	}
	return res
}

func requireEnvInt(envName string) int {
	resStr := requireEnvStr(envName)
	res, err := strconv.Atoi(resStr)
	if err != nil {
		log.Panic(fmt.Sprintf("expected int in env var %s, got: %s", envName, resStr))
	}
	return res
}
