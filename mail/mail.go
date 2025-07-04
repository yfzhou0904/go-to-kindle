package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
)

func SendEmailWithAttachment(smtpServer, from, password, to, subject, htmlFilePath string, port int) error {
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
	message += "\r\n"

	// Create the body part
	bodyPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}})
	if err != nil {
		return err
	}
	if _, err := bodyPart.Write([]byte("here's an article for you")); err != nil {
		return err
	}

	// Create the attachment part
	// Encode the file name to handle most characters.
	htmlFileName := filepath.Base(htmlFilePath)
	encodedHTMLFileName := mime.QEncoding.Encode("utf-8", htmlFileName)
	attachmentPartHeader := textproto.MIMEHeader{
		"Content-Type": {"application/octet-stream"},
		"Content-Disposition": {
			"attachment; filename=\"" + htmlFileName + "\"; filename*=UTF-8''" + encodedHTMLFileName,
		},
	}
	attachmentPart, err := writer.CreatePart(attachmentPartHeader)
	if err != nil {
		return err
	}
	htmlContentBs, err := os.ReadFile(htmlFilePath)
	if err != nil {
		return err
	}
	htmlContentAscii := escapeNonASCII(string(htmlContentBs))
	if _, err := attachmentPart.Write([]byte(htmlContentAscii)); err != nil {
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
		slog.Error("email auth error", "err", err, "auth", auth)
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

func escapeNonASCII(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if r > 127 {
			buf.WriteString(fmt.Sprintf("&#%d;", r))
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
