package auth

import (
	"app/conf"
	"app/constants"
	"app/core/auth/values"
	"app/db"
	. "app/i18n"
	"app/mail"
	"app/mq"
	"app/srvCtx"
	"app/templates"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/nakami-lounge-GmbH/tools/helpers"
	"github.com/stephenafamo/bob"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserAllreadyExists = errors.New("user already exists")

type Auth struct{}

func NewAuth() *Auth {
	return &Auth{}
}
func (m *Auth) CreateProfile(srv *srvCtx.BobContext, email string, password string, salutation string, forename string, lastname string,
	disabled bool, emailValidated bool, sync bool, currentUserID null.Val[int32]) (*mq.UserProfile, error) {

	user, err := m.createProfile(srv, salutation, forename, lastname, email, password == "", disabled, emailValidated, password)
	if err != nil {
		return nil, err
	}
	if password == "" {
		validTo := time.Now().Add(values.TokenNameRegistrationValid)

		tt := helpers.RandomString(10)
		webLink := fmt.Sprintf(conf.C.WebURL.CreatePassword, tt)

		token, err := m.CreateToken(srv, db.NullVal(user.ID), db.NullVal(user.ID), values.TokenNameUserPassword, tt, webLink, validTo)
		if err != nil {
			return nil, fmt.Errorf("error creating token: %w", err)
		}

		data := struct {
			User    *mq.UserProfile
			WebLink string
			ValidTo time.Time
		}{
			user,
			webLink,
			validTo,
		}

		body, err := templates.ExecuteTextTemplate(templates.TplNewUserPasswordTokenMail, data)
		if err != nil {
			return nil, err
		}

		subject := fmt.Sprintf("Neuer Benutzer angelegt für %s", conf.C.ApplicationName)
		f := func(fromEmail, toEmail string, subject string, body string) {
			log.Println("Sending mail to:", toEmail+" with token: "+token.TokenValue)
			if err := mail.SendMail(srv.Ctx, srv.DbTx(), fromEmail, []string{toEmail}, nil, nil, subject, body, "", "", ""); err != nil {
				log.Println("Error sending mail:", err)
			}
		}

		if sync {
			f(conf.C.MailConfig.Sender, user.Email, subject, body)
		} else {
			go f(conf.C.MailConfig.Sender, user.Email, subject, body)
		}
	}

	return user, nil
}

func (m *Auth) ResetPassword(ctx context.Context, tx bob.Executor, user *mq.UserProfile) (*mq.UserProfile, error) {
	if err := user.Update(ctx, tx, &mq.UserProfileSetter{
		MustResetPassword: omit.From(true),
		UpdatedAt:         omitnull.From(time.Now()),
	}); err != nil {
		return user, fmt.Errorf("error updating user: %w", err)
	}

	validTo := time.Now().Add(values.TokenPasswordsetValid)

	tt := helpers.RandomString(10)
	webLink := fmt.Sprintf(conf.C.WebURL.CreatePassword, tt)

	tokenSetter := &mq.TokenSetter{
		UserID:     omitnull.From(user.ID),
		TokenName:  omit.From(values.TokenNameUserPassword),
		TokenValue: omit.From(tt),
		TokenURI:   omit.From(webLink),

		ValidTo: omit.From(validTo),
	}

	ts, err := mq.Tokens.Insert(tokenSetter).One(ctx, tx)
	if err != nil {
		return user, fmt.Errorf("error inserting token: %w", err)
	}

	if err := ts.AttachUserUserProfile(ctx, tx, user); err != nil {
		return user, fmt.Errorf("error attaching user to token: %w", err)
	}

	data := struct {
		User    *mq.UserProfile
		WebLink string
		ValidTo time.Time
	}{
		user,
		webLink,
		validTo,
	}

	body, err := templates.ExecuteTextTemplate(templates.TplPasswordChangeTokenMail, data)
	if err != nil {
		return user, fmt.Errorf("executing template: %w", err)
	}

	subject := fmt.Sprintf("Passwort geändert für %s", conf.C.ApplicationName)
	if err := mail.SendMail(ctx, tx, conf.C.MailConfig.Sender, []string{user.Email}, nil, nil, subject, body, "", "", ""); err != nil {
		return user, fmt.Errorf("sending mail: %w", err)
	}

	return user, nil
}

func (m *Auth) CryptPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), constants.PasswordWeight)
	if err != nil {
		return "", fmt.Errorf("could not generate password hash: %w", err)
	}
	return string(hash), nil
}

func (m *Auth) createProfile(srv *srvCtx.BobContext, Salutation string, Forename string, Lastname string,
	Email string, MustResetPassword bool, Disabled bool, EmailValidated bool, Password string) (*mq.UserProfile, error) {

	exist, err := mq.UserProfiles.Query(mq.SelectWhere.UserProfiles.Email.EQ(Email)).Exists(srv.Ctx, db.DBob)
	if err != nil {
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if exist {
		return nil, errors.New(Tr(LK_email_already_exists))
	}
	passwd, err := m.CryptPassword(Password)
	if err != nil {
		return nil, fmt.Errorf("error encrypting password: %w", err)
	}

	user, err := mq.UserProfiles.Insert(&mq.UserProfileSetter{
		Salutation:        omit.From(Salutation),
		Forename:          omit.From(Forename),
		Lastname:          omit.From(Lastname),
		Email:             omit.From(Email),
		UpdatedAt:         omitnull.From(time.Now()),
		MustResetPassword: omit.From(MustResetPassword),
		Disabled:          omit.From(Disabled),
		EmailValidated:    omit.From(EmailValidated),
		Password:          omit.From(passwd),
	}).One(srv.Ctx, srv.DbTx())
	if err != nil {
		return nil, fmt.Errorf("error adding user: %w", err)
	}

	return user, nil
}

func (m *Auth) CreateToken(srv *srvCtx.BobContext, userID null.Val[int32], objectID null.Val[int32], TokenName string,
	TokenValue string, TokenURI string, ValidTo time.Time) (*mq.Token, error) {
	setter := mq.TokenSetter{
		UserID:     omitnull.FromNull(userID),
		ObjectID:   omitnull.FromNull(objectID),
		TokenName:  omit.From(TokenName),
		TokenValue: omit.From(TokenValue),
		TokenURI:   omit.From(TokenURI),
		ValidTo:    omit.From(ValidTo),
	}

	return mq.Tokens.Insert(&setter).One(srv.Ctx, srv.DbTx())
}
