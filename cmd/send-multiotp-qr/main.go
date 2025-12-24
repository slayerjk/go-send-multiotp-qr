package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	// change this path for your project

	mailing "github.com/slayerjk/go-mailing"
	multiotp "github.com/slayerjk/go-multiotpwork"
	"github.com/slayerjk/go-send-multiotp-qr/internal/helpers"
	vafswork "github.com/slayerjk/go-vafswork"
)

const (
	appName = "send-multiotp-qr"
)

// define user
type User struct {
	name   string
	email  string
	qrPath string
}

func main() {
	// defining default values
	var (
		workDir         string    = vafswork.GetExePath()
		logsPathDefault string    = workDir + "/logs" + "_" + appName
		startTime       time.Time = time.Now()
		succeededUsers  []User
		failedUsers     []User
		wg              sync.WaitGroup
	)

	// setting channels
	chanDone := make(chan string)
	chanGenQr := make(chan string)
	chanSendMail := make(chan User)

	wg.Add(1)

	// logging flags
	logsDir := flag.String("log-dir", logsPathDefault, "set custom log dir")
	logsToKeep := flag.Int("keep-logs", 7, "set number of logs to keep after rotation")
	// multiotp flags
	multiOTPBinPath := flag.String("mpath", "/usr/local/bin/multiotp/multiotp.php", "full path to multiotp binary")
	qrCodesPath := flag.String("qrpath", "/etc/multiotp/qrcodes", "qr codes full path to save")
	usersPath := flag.String("upath", "/etc/multiotp/users", "MultiOTP users dir(*.db files)")
	issuerDescr := flag.String("idesc", "TEST-SRV-OTP", "issuer(your MultiOTP server) description")
	// mail flags
	emailText := flag.String("etext", "Your OTP QR", "email text in email body, will be used along with 'idesc'")
	mailHost := flag.String("mhost", "mail.example.com", "mail host(ip or hostname), must be valid")
	mailPort := flag.Int("mport", 25, "mail port")
	mailFrom := flag.String("mfrom", "multiotp@example.com", "mail from address, domain will be used as users' domain")
	mailSubject := flag.String("msubj", "Your QR Code", "mail subject, date and time will be added in the end")
	mailAdmins := flag.String("madmins", "NONE", "admins' emails separated by coma")

	flag.Usage = func() {
		fmt.Println("Send MutltiOTP QRs")
		fmt.Println("Version = 0.1.1")
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

	// starting programm notification
	logger.Info("Program Started", "app name", appName)

	// rotate logs
	logger.Info("Log rotation first", "logsDir", *logsDir, "logs to keep", *logsToKeep)
	if err := vafswork.RotateFilesByMtime(*logsDir, *logsToKeep); err != nil {
		fmt.Fprintf(os.Stdout, "failed to rotate logs:\n\t%v", err)
	}

	// check admins mails
	// skip if "NONE" or wrong mails
	mailToAdminIsOn := false
	adminsList := make([]string, 0)
	reportSubject := fmt.Sprintf("Report - %s", *issuerDescr)
	if *mailAdmins != "NONE" {
		adminsList = strings.Split(*mailAdmins, ",")
		mailToAdminIsOn = true
	}

	// setting user domain
	userDomain := strings.Split(*mailFrom, "@")[1]
	if len(userDomain) == 0 {
		fmt.Println("check 'mailFrom' domain, cannot be empty after '@'")
		logger.Error("wrong mailFrom domain", "mailFrom", *mailFrom)

		// send report to admin
		if mailToAdminIsOn {
			logger.Info("sending admin report")
			err := helpers.SendReport(*mailHost, *mailPort, *mailFrom, reportSubject, logFilePath, adminsList, nil)
			if err != nil {
				logger.Warn("failed to send mail to admins", "admins", adminsList, "err", err)
			}
		}

		os.Exit(1)
	}

	// 1) resync ldap users at start
	go func() {
		logger.Info("start to resync LDAP users")
		err = multiotp.ResyncMultiOTPUsers(*multiOTPBinPath)
		if err != nil {
			logger.Error("failed to resync LDAP users of MultiOTP", "err", err)
			fmt.Println("err, check log")
			// send report to admin
			if mailToAdminIsOn {
				logger.Info("sending admin report")
				err := helpers.SendReport(*mailHost, *mailPort, *mailFrom, reportSubject, logFilePath, adminsList, nil)
				if err != nil {
					logger.Warn("failed to send mail to admins", "admins", adminsList, "err", err)
				}
			}

			os.Exit(1)
		}
		logger.Info("done resync LDAP users")

		chanDone <- "DONE"
		close(chanDone)
	}()

	// searching for new users
	go func() {
		// waiting resync ldap is done
		<-chanDone

		// 2) collecting all users in users dir of multiotp
		logger.Info("collecting all NEW users")

		correctUserFile := regexp.MustCompile(`^(\w+)\.db$`)

		dirEntry, err := os.ReadDir(*usersPath)
		if err != nil {
			logger.Error("failed to read Users dir of MultiOTP", "err", err)
			fmt.Println("err, check log")
			// send report to admin
			if mailToAdminIsOn {
				logger.Info("sending admin report")
				err := helpers.SendReport(*mailHost, *mailPort, *mailFrom, reportSubject, logFilePath, adminsList, nil)
				if err != nil {
					logger.Warn("failed to send mail to admins", "admins", adminsList, "err", err)
				}
			}

			os.Exit(1)
		}

		for _, file := range dirEntry {
			if file.IsDir() {
				continue
			}

			userMatch := correctUserFile.FindStringSubmatch(file.Name())

			if userMatch != nil {
				// 3) check qr codes dir if users already have generated .png file
				// if user don't hanve .png qr thus it's a new user
				correctUserQRFile := regexp.MustCompile(`^(\w+)\.png$`)
				dirEntry, err = os.ReadDir(*qrCodesPath)

				if err != nil {
					logger.Error("Failed to read QR codes dir of MultiOTP", "err", err)
					fmt.Println("err, check log")
					// send report to admin
					if mailToAdminIsOn {
						logger.Info("sending admin report")
						err := helpers.SendReport(*mailHost, *mailPort, *mailFrom, reportSubject, logFilePath, adminsList, nil)
						if err != nil {
							logger.Warn("failed to send mail to admins", "admins", adminsList, "err", err)
						}
					}

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
					chanGenQr <- newUser
				}
			}
		}
		close(chanGenQr)
	}()

	// 4) generate PNG qrs for user without qr
	go func() {
		for {
			newUser, ok := <-chanGenQr
			if !ok {
				break
			}

			logger.Info("generating qr for user", "user", newUser)
			err := multiotp.GenerateMultiOTPQRPng(*multiOTPBinPath, newUser, *qrCodesPath)
			if err != nil {
				logger.Warn("Failed to generate user's QR png, skipping", "user", newUser, "err", err)
				failedUsers = append(failedUsers, User{name: newUser})
				chanSendMail <- User{name: "SKIP"}
				continue
			}
			userQrPngPath := fmt.Sprintf("%s/%s.png", *qrCodesPath, newUser)
			chanSendMail <- User{name: newUser, qrPath: userQrPngPath}
		}
		close(chanSendMail)
	}()

	// 5) send email to user with new generated qr
	go func() {
		for {
			newUser, ok := <-chanSendMail
			if !ok {
				break
			}

			if newUser.name == "SKIP" {
				continue
			}

			logger.Info("sending QR to user", "user", newUser.name)

			newUser.email = fmt.Sprintf("%s@%s", newUser.name, userDomain)
			body := fmt.Sprintf("<html><body><p>%s: %s</p></body></html>", *emailText, *issuerDescr)

			err = mailing.SendEmailWoAuth("html", *mailHost, *mailPort, *mailFrom, *mailSubject, body, []string{newUser.email}, []string{newUser.qrPath})
			if err != nil {
				logger.Warn("failed to send email to user, skipping", "user", newUser.name, "err", err)
				failedUsers = append(failedUsers, User{name: newUser.name, email: newUser.email})
				// deleting generated qr
				logger.Info("deleting generated QR png file for failed user", "qrpath", newUser.qrPath)
				err := os.Remove(newUser.qrPath)
				if err != nil {
					logger.Warn("failed to delete generated QR png file for failed user", "qrpath", newUser.qrPath, "err", err)
				}
				continue
			}

			succeededUsers = append(succeededUsers, User{name: newUser.name, email: newUser.email})
		}
		wg.Done()
	}()

	// wait all goroutines did their job
	wg.Wait()

	// data for report
	if len(succeededUsers) != 0 {
		logger.Info("new users processed", "users", succeededUsers)
	} else {
		logger.Info("no new users processed")
	}
	// data for report
	if len(failedUsers) != 0 {
		logger.Info("failed users:", "failedUsers", failedUsers)
	}

	// count & print estimated time
	logger.Info("Program Done", slog.Any("estimated time(sec)", time.Since(startTime).Seconds()))

	// send report to admin
	if len(succeededUsers) != 0 || len(failedUsers) != 0 {
		if mailToAdminIsOn {
			logger.Info("sending FINAL report to admin")
			reportSubject += "(FINAL)"
			finalReportBody := fmt.Sprintf("Succeeded users:\n\t%v\nFailed users:\n\t%v", succeededUsers, failedUsers)
			err := mailing.SendEmailWoAuth("plain", *mailHost, *mailPort, *mailFrom, reportSubject, finalReportBody, adminsList, nil)
			if err != nil {
				logger.Warn("failed to send FINAL report to admins", "admins", adminsList, "err", err)
			}
		}
	}
}
