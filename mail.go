package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
)

func sendEmailWithAttachment(smtpServer, from, password, to, subject, htmlFilePath string, port int) error {

	// Read the attachment file
	attachmentFile, err := os.Open(htmlFilePath)
	if err != nil {
		return err
	}
	defer attachmentFile.Close()

	// Create a buffer to store the email body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// header part
	header := make(map[string]string)
	header["From"] = from
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary())

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body.String()

	// Create the body part
	bodyPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}})
	if err != nil {
		return err
	}
	if _, err := bodyPart.Write([]byte("here's an article for you")); err != nil {
		return err
	}

	// Create the attachment part
	attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/octet-stream"}, "Content-Disposition": {"attachment; filename=" + strings.ReplaceAll(filepath.Base(htmlFilePath), " ", "_")}})
	if err != nil {
		return err
	}
	if _, err := io.Copy(attachmentPart, attachmentFile); err != nil {
		return err
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return err
	}

	// Set up authentication information
	auth := smtp.PlainAuth("", from, password, smtpServer)

	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpServer,
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", smtpServer, port), tlsconfig)
	if err != nil {
		return err
	}

	c, err := smtp.NewClient(conn, smtpServer)
	if err != nil {
		return err
	}
	if err = c.Auth(auth); err != nil {
		return err
	}

	// To && From
	if err = c.Mail(from); err != nil {
		return err
	}

	if err = c.Rcpt(to); err != nil {
		return err
	}

	// Data
	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	_, err = w.Write(body.Bytes())
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	c.Quit()

	return nil
}
