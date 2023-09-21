package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strings"
	"text/template"
	"time"
)

var (
	addr = os.Getenv("MORE_SMTP_ADDR")
	user = os.Getenv("MORE_SMTP_USER")
	pass = os.Getenv("MORE_SMTP_PASS")
	to   = os.Getenv("MORE_SMTP_TO")
	t    = template.Must(template.New("more").Parse("From: {{.From}}\r\nTo: {{.To}}\r\nSubject: {{.Subject}}\r\nContent-Type: {{.ContentType}}\r\n\r\n{{.Body}}"))
)

type Word struct {
	Title   string `json:"title"`
	Author  string `json:"author"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

type Collection struct {
	Words []*Word `json:"words"`
}

func main() {
	if len(os.Args) < 2 {
		slog.Error("no collection")
		os.Exit(1)
	}

	file := os.Args[1]
	b, err := os.ReadFile(file)
	if err != nil {
		slog.Error("read collection", "err", err, "file", file)
		os.Exit(1)
	}
	var c Collection
	err = json.Unmarshal(b, &c)
	if err != nil {
		slog.Error("unmarshal collection", "err", err, "file", file)
		os.Exit(1)
	}
	day := time.Now().YearDay()
	i := day % len(c.Words)
	word := c.Words[i]
	slog.Info("today's word", "day", day, "index", i, "title", word.Title, "author", word.Author, "type", word.Type)
	notification(word)
}

func notification(word *Word) {
	type Data struct {
		From        string
		To          string
		Subject     string
		ContentType string
		Body        string
		Word        *Word
	}

	if addr == "" {
		slog.Warn("send notification skip", "reason", "addr is empty")
		return
	}
	slog.Info("sending notification")
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		slog.Error("send notification fail", "err", err)
		return
	}

	body := ""
	subject := "早辰一读"
	body += strings.Join([]string{fmt.Sprintf("《%s》", word.Title), word.Author, word.Content}, "\n\n")
	data := Data{
		From:        fmt.Sprintf("%s <%s>", mime.BEncoding.Encode("UTF-8", "Monitor"), user),
		To:          to,
		Subject:     mime.BEncoding.Encode("UTF-8", fmt.Sprintf("「MORE」%s", subject)),
		ContentType: "text/plain; charset=utf-8",
		Body:        body,
		Word:        word,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		slog.Error("send notification fail", "err", err)
		return
	}

	auth := smtp.PlainAuth("", user, pass, host)
	if err := smtp.SendMail(addr, auth, user, strings.Split(to, ","), buf.Bytes()); err != nil {
		slog.Error("send notification fail", "err", err)
		return
	}
	slog.Info("send notification success")
}
