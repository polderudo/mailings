package mail

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"

	"app/conf"
	"app/mq"
	"log"
	"net/smtp"
	"os"

	"github.com/polderudo/mailyak"
	"github.com/stephenafamo/bob"

	"github.com/nakami-lounge-GmbH/tools/helpers"
)

func SendMail(ctx context.Context, tx bob.Executor, from string,
	to []string, cc *string, bcc *string,
	subject string, bodyText string, bodyHTML string, filename string, filepath string) error {

	for _, e := range to {
		if err := sendMailBob(ctx, tx, from, e, cc, bcc, subject, bodyText, bodyHTML, filename, filepath); err != nil {
			return fmt.Errorf("error sending mail:%w", err)
		}
	}

	return nil
}

// sendMailBob should be called as a goroutine
// it sends the mail and stores errors and status in the database
func sendMailBob(ctx context.Context, tx bob.Executor, from string,
	to string, cc *string, bcc *string,
	subject string, bodyText string, bodyHTML string, filename string, filepath string) error {

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in intSendMail: ", r)
		}
	}()

	auth := smtp.PlainAuth("", conf.C.MailConfig.User, conf.C.MailConfig.Password, conf.C.MailConfig.Host)

	if conf.C.MailConfig.NoAuth {
		log.Println("Not using auth for mailing")
		auth = nil
	}

	mail := mailyak.New(
		fmt.Sprintf("%s:%d", conf.C.MailConfig.Host, conf.C.MailConfig.Port),
		auth,
	)

	if conf.C.IsDevSystem && conf.C.MailToDevelop != "" {
		//In develop we don't want to send any mail to external adresses
		to = conf.C.MailToDevelop
	}

	if conf.C.MailConfig.UseSSL {
		mail.UseTLS(&tls.Config{InsecureSkipVerify: conf.C.MailConfig.Insecure})
	}

	mail.To(to)

	if cc != nil {
		mail.Cc(*cc)
	}

	if bcc != nil {
		mail.Bcc(*bcc)
	}

	mail.From(from)
	mail.Subject(subject)

	if bodyHTML != "" {
		mail.HTML().Set(bodyHTML)
	}

	if bodyText != "" {
		mail.Plain().Set(bodyText)
	}

	if filename != "" && filepath != "" {
		if helpers.FileExists(filepath) {
			file, err := os.Open(filepath)
			if err != nil {
				log.Println("Could not open file:", filepath, err)
			} else {
				mail.Attach(filename, file)
				defer file.Close()
			}
		}
	}

	mDB := &mq.MailSetter{
		MailTo:         omit.From(to),
		MailSubject:    omit.From(subject),
		MailBody:       omit.From(fmt.Sprintf("Text:%s\n\nHTML:%s", bodyText, bodyHTML)),
		InternalStatus: omit.From(int64(1)),
		MailFrom:       omit.From(from),
	}

	var mailErr error
	if mailErr = mail.Send(); mailErr != nil {
		log.Printf("Error sending mail: %v\n", mailErr)
		mDB.ExternalStatus = omitnull.From(mailErr.Error())
		mDB.InternalStatus = omit.From(int64(-1))
	} else {
		log.Println("Mail sent to:", to)
	}

	_, err := mq.Mails.Insert(mDB).One(ctx, tx)
	if err != nil {
		return fmt.Errorf("error inserting mail to db:%w", err)
	}

	if mailErr != nil {
		return mailErr //error only if we could not send the mail, but not if storing the log did not work
	}

	return nil
}
