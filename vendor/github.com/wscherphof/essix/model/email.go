package model

import (
	"github.com/wscherphof/essix/util"
)

/*
CreateEmailToken generates a token that is needed to change the Account's email
address.
*/
func (a *Account) CreateEmailToken(newEmail string) (err error, conflict bool) {
	var empty bool
	email := initEmail(newEmail)
	if err, empty = email.Read(email); err != nil {
		if empty {
			a.NewEmail = newEmail
			a.EmailToken = util.NewToken()
			err = a.Update(a)
		}
	} else {
		err, conflict = ErrEmailTaken, true
	}
	return
}

/*
ClearEmailToken clears the token to cancel the email address changing process.
*/
func (a *Account) ClearEmailToken(token string) (err error, conflict bool) {
	if token == "" || a.EmailToken != token {
		err, conflict = ErrInvalidCredentials, true
	} else {
		a.NewEmail = ""
		a.EmailToken = ""
		err = a.Update(a)
	}
	return
}

/*
ChangeEmail sets the Account's Email to the NewEmail, if the given token is
correct.
*/
func (a *Account) ChangeEmail(token string) (err error, conflict bool) {
	email := initEmail(a.NewEmail)
	if token == "" || a.EmailToken != token || a.NewEmail == "" {
		err, conflict = ErrInvalidCredentials, true
	} else if err, conflict = email.Create(email); err != nil {
		if conflict {
			err = ErrEmailTaken
		}
	} else {
		a.Email = a.NewEmail
		a.EmailToken = ""
		if err = a.Update(a); err == nil {
			err = email.Delete(email)
		}
	}
	return
}
