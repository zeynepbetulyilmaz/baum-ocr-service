package mailer

import (
	"fmt"
	"net/smtp"
)

// Config, SMTP üzerinden e-posta göndermek için gereken bilgileri tutar.
// Alanlar boş bırakılırsa (yerel geliştirme ortamı gibi) Configured() false
// döner ve çağıran taraf gerçek e-posta göndermek yerine başka bir yola
// (ör. loglara yazma) düşmelidir.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (c Config) Configured() bool {
	return c.Host != "" && c.Port != "" && c.Username != "" && c.Password != "" && c.From != ""
}

// Send, net/smtp.SendMail ile e-posta gönderir. Go'nun net/smtp paketi
// sunucu STARTTLS destekliyorsa (Gmail, Outlook, SendGrid vb. 587 portunda
// destekler) bağlantıyı otomatik olarak şifreliyor, ayrıca TLS kurulumu
// yapmaya gerek yok.
func (c Config) Send(to, subject, body string) error {
	auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		c.From, to, subject, body,
	)
	addr := fmt.Sprintf("%s:%s", c.Host, c.Port)
	return smtp.SendMail(addr, auth, c.From, []string{to}, []byte(msg))
}
