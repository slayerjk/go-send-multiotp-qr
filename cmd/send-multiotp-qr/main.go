package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	// change this path for your project

	// mailing "github.com/slayerjk/go-mailing"
	multiotp "github.com/slayerjk/go-multiotpwork"
	vafswork "github.com/slayerjk/go-vafswork"
	// vawebwork "github.com/slayerjk/go-vawebwork"
)

const (
	appName = "send-multiotp-qr"
)

// define user
type User struct {
	name       string
	email      string
	qrFailed   bool
	mailFailed bool
}

// for dep injection
// type application struct {
// 	logger *slog.Logger
// }

func main() {
	// defining default values
	var (
		workDir         string    = vafswork.GetExePath()
		logsPathDefault string    = workDir + "/logs" + "_" + appName
		startTime       time.Time = time.Now()
		// mailingFileDefault       string = workDir + "/data/mailing.json"
		newUsers []User
	)

	// flags
	logsDir := flag.String("log-dir", logsPathDefault, "set custom log dir")
	logsToKeep := flag.Int("keep-logs", 7, "set number of logs to keep after rotation")
	// mailingFile := flag.String("m-file", mailingFileDefault, "file with mailing settings")
	multiOTPBinPath := flag.String("mpath", "/usr/local/bin/multiotp/multiotp.php", "full path to multiotp binary")
	qrCodesPath := flag.String("qrpath", "/etc/multiotp/qrcodes", "qr codes full path to save")
	usersPath := flag.String("upath", "/etc/multiotp/users", "MultiOTP users dir(*.db files)")
	// user := flag.String("user", "NONE", "user name to generate qr(ususally in /etc/multiotp/users)")
	// descrString := flag.String("tdescr", "TEST", "token description")

	flag.Usage = func() {
		fmt.Println("Send MutltiOTP QRs")
		fmt.Println("Version = 0.0.0")
		fmt.Println("Usage: <app> [-opt] ...")
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// logging
	// create log dir
	if err := os.MkdirAll(*logsDir, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stdout, "failed to create log dir %s:\n\t%v", *logsDir, err)
		os.Exit(1)
	}
	// set current date
	dateNow := time.Now().Format("02.01.2006")
	// create log file
	logFilePath := fmt.Sprintf("%s/%s_%s.log", *logsDir, appName, dateNow)
	// open log file in append mode
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to open created log file %s:\n\t%v", logFilePath, err)
		os.Exit(1)
	}
	defer logFile.Close()
	// set logger
	logger := slog.New(slog.NewTextHandler(logFile, nil))
	// test logger
	// logger.Info("info test-1", slog.Any("val", "key"))

	// starting programm notification
	logger.Info("Program Started", "app name", appName)

	// rotate logs
	logger.Info("Log rotation first", "logsDir", *logsDir, "logs to keep", *logsToKeep)
	if err := vafswork.RotateFilesByMtime(*logsDir, *logsToKeep); err != nil {
		fmt.Fprintf(os.Stdout, "failed to rotate logs:\n\t%v", err)
	}

	// setting application struct with dep injection
	// app := &application{
	// 	logger: logger,
	// }

	// 1) resync ldap users at start
	err = multiotp.ResyncMultiOTPUsers(*multiOTPBinPath)
	if err != nil {
		logger.Error("Failed to resync LDAP users of MultiOTP", "err", err)
		os.Exit(1)
	}
	logger.Info("Success: resync LDAP users")

	// 2) collect all users which are already in users' dir of multiotp
	correctUserFile := regexp.MustCompile(`^(\w+)\.db$`)
	dirEntry, err := os.ReadDir(*usersPath)
	if err != nil {
		logger.Error("Failed to read Users dir of MultiOTP", "err", err)
		os.Exit(1)
	}

	for _, file := range dirEntry {
		if file.IsDir() {
			continue
		}
		userMatch := correctUserFile.FindStringSubmatch(file.Name())

		if userMatch != nil {
			// 3) check qr codes dir if users already have generated .png file
			correctUserQRFile := regexp.MustCompile(`^(\w+)\.png$`)
			dirEntry, err = os.ReadDir(*qrCodesPath)
			if err != nil {
				logger.Error("Failed to read QR codes dir of MultiOTP", "err", err)
				os.Exit(1)
			}

			isUserAndQRMatched := 0
			for _, file := range dirEntry {
				if file.IsDir() {
					continue
				}
				qRMatch := correctUserQRFile.FindStringSubmatch(file.Name())
				if qRMatch != nil {
					if userMatch[1] == qRMatch[1] {
						// users = append(users, User{name: userMatch[1], qrReady: true, email: fmt.Sprintf("%s@nurbank.kz", userMatch[1])})
						isUserAndQRMatched += 1
						break
					}
				}
			}
			if isUserAndQRMatched == 0 {
				// 4) generate qrs for user without qr (qrReady=false)
				err := multiotp.GenerateMultiOTPQRPng(*multiOTPBinPath, userMatch[1], *qrCodesPath)
				if err != nil {
					logger.Warn("Failed to generate user's QR png", "user", userMatch[1], "err", err)
					newUsers = append(newUsers, User{name: userMatch[1], qrFailed: true})
					continue
				}
				logger.Info("Success: QR generation", "user", userMatch[1])

				// send email to user with new generated qr
				// msg := []byte("TEST mail")
				// err = mailing.SendPlainEmailWoAuth(*mailingFile, "report", "send-multiotp-qr", msg)
				// if err != nil {
				// 	logging.Warn("failed to send email to user", "user", userMatch[1])
				// }

				// newUsers = append(newUsers, User{name: userMatch[1], email: fmt.Sprintf("%s@nurbank.kz", userMatch[1])})
			}
		}
	}

	fmt.Println(newUsers)

	// count & print estimated time
	endTime := time.Now()
	logger.Info("Program Done", slog.Any("estimated time(sec)", endTime.Sub(startTime).Seconds()))
}
