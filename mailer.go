package main

import (
	"log"
	"net/smtp"
	"os"
)

func SendMail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	if smtpHost == "" || smtpPort == "" {
		log.Println("smtp not configured, skipping send to", to)
		return nil
	}
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	msg := "From: " + smtpUser + "\nTo: " + to + "\nSubject: " + subject + "\n\n" + body
	addr := smtpHost + ":" + smtpPort
	return smtp.SendMail(addr, auth, smtpUser, []string{to}, []byte(msg))
}
