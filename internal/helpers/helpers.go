package helpers

import (
	"fmt"
	"os"

	mailing "github.com/slayerjk/go-mailing"
)

// form report from log file
func formReportFromFile(logPath string) (string, error) {
	// check if file exists
	if _, err := os.Stat(logPath); err != nil {
		return "", err
	}

	// read
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// send plain text mail report
// "html", *mailHost, *mailPort, *mailFrom, *mailSubject, body, []string{newUser.email}, []string{newUser.qrPath}
func SendReport(
	mailHost string, mailPort int, mailFrom string, mailSubject string,
	filePathToReport string, adminsList []string, attch []string) error {

	// get log text
	reportBody, err := formReportFromFile(filePathToReport)
	if err != nil {
		return fmt.Errorf("failed to form report from file: %s, %v", filePathToReport, err)
	}

	// send mail
	err = mailing.SendEmailWoAuth("plain", mailHost, mailPort, mailFrom, mailSubject, reportBody, adminsList, nil)
	if err != nil {
		return fmt.Errorf("failed to send report: %v, %v", adminsList, err)
	}

	return nil
}
