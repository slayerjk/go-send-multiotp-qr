package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	// change this path for your project

	mailing "github.com/slayerjk/go-mailing"
	multiotp "github.com/slayerjk/go-multiotpwork"
	vafswork "github.com/slayerjk/go-vafswork"
)

const (
	appName = "send-multiotp-qr"
)

// define user
type User struct {
	name  string
	email string
}

func main() {
	// defining default values
	var (
		workDir         string    = vafswork.GetExePath()
		logsPathDefault string    = workDir + "/logs" + "_" + appName
		startTime       time.Time = time.Now()
		succeededUsers  []User
		failedUsers     []User
	)

	// logging flags
	logsDir := flag.String("log-dir", logsPathDefault, "set custom log dir")
	logsToKeep := flag.Int("keep-logs", 7, "set number of logs to keep after rotation")
	// multiotp flags
	multiOTPBinPath := flag.String("mpath", "/usr/local/bin/multiotp/multiotp.php", "full path to multiotp binary")
	qrCodesPath := flag.String("qrpath", "/etc/multiotp/qrcodes", "qr codes full path to save")
	usersPath := flag.String("upath", "/etc/multiotp/users", "MultiOTP users dir(*.db files)")
	tokenDescr := flag.String("tdescr", "TEST-SRV-OTP", "token description")
	// mail flags
	emailText := flag.String("etext", "Your OTP QR", "email text above QR code")
	mailHost := flag.String("mhost", "mail.example.com", "mail host(ip or hostname), must be valid")
	mailPort := flag.Int("mport", 25, "mail port")
	mailFrom := flag.String("mfrom", "multiotp@example.com", "mail from address, domain will be used as users' domain")
	mailSubject := flag.String("msubj", "Your QR Code", "mail subject, date and time will be added in the end")

	flag.Usage = func() {
		fmt.Println("Send MutltiOTP QRs")
		fmt.Println("Version = 0.0.1")
		fmt.Println("Usage: <app> [-opt] ...")
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// setting user domain
	userDomain := strings.Split(*mailFrom, "@")[1]
	if len(userDomain) == 0 {
		fmt.Println("check 'mailFrom' domain, cannot be empty after '@'")
		os.Exit(1)
	}

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

	// starting programm notification
	logger.Info("Program Started", "app name", appName)

	// rotate logs
	logger.Info("Log rotation first", "logsDir", *logsDir, "logs to keep", *logsToKeep)
	if err := vafswork.RotateFilesByMtime(*logsDir, *logsToKeep); err != nil {
		fmt.Fprintf(os.Stdout, "failed to rotate logs:\n\t%v", err)
	}

	// 1) resync ldap users at start
	logger.Info("start to resync LDAP users")
	err = multiotp.ResyncMultiOTPUsers(*multiOTPBinPath)
	if err != nil {
		logger.Error("failed to resync LDAP users of MultiOTP", "err", err)
		fmt.Println("err, check log")
		os.Exit(1)
	}
	logger.Info("done resync LDAP users")

	// 2) collect all users which are already in users' dir of multiotp
	logger.Info("started to collect all users which are already in users' dir of multiotp")
	correctUserFile := regexp.MustCompile(`^(\w+)\.db$`)
	dirEntry, err := os.ReadDir(*usersPath)
	if err != nil {
		logger.Error("failed to read Users dir of MultiOTP", "err", err)
		fmt.Println("err, check log")
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
				fmt.Println("err, check log")
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
						isUserAndQRMatched += 1
						break
					}
				}
			}
			if isUserAndQRMatched == 0 {
				newUser := strings.Trim(userMatch[1], " ")

				// 4) generate PNG qrs for user without qr
				logger.Info("generating qr for user", "user", newUser)
				err := multiotp.GenerateMultiOTPQRPng(*multiOTPBinPath, newUser, *qrCodesPath)
				if err != nil {
					logger.Warn("Failed to generate user's QR png, skipping", "user", newUser, "err", err)
					failedUsers = append(failedUsers, User{name: newUser})
					continue
				}
				userQrPngPath := fmt.Sprintf("%s/%s.png", *qrCodesPath, newUser)

				// 5) send email to user with new generated qr
				logger.Info("sending QR to user", "user", newUser)
				newUserMail := fmt.Sprintf("%s@%s", newUser, userDomain)
				body := fmt.Sprintf("<html><body><p>%s: %s</p></body></html>", *emailText, *tokenDescr)
				err = mailing.SendEmailWoAuth("html", *mailHost, *mailPort, *mailFrom, *mailSubject, string(body), []string{newUserMail}, []string{userQrPngPath})
				if err != nil {
					logger.Warn("failed to send email to user, skipping", "user", newUser, "err", err)
					failedUsers = append(failedUsers, User{name: newUser, email: newUserMail})
					continue
				}

				succeededUsers = append(succeededUsers, User{name: newUser, email: newUserMail})
			}
		}
	}

	if len(succeededUsers) != 0 {
		logger.Info("new users processed\n\t", "users", succeededUsers)
	} else {
		logger.Info("no new users processed")
	}

	if len(failedUsers) != 0 {
		logger.Info("failed users:\n\t", "failedUsers", failedUsers)
	}

	// count & print estimated time
	endTime := time.Now()
	logger.Info("Program Done", slog.Any("estimated time(sec)", endTime.Sub(startTime).Seconds()))
}
