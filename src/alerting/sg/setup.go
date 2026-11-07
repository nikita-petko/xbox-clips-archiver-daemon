package sg

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

var (
	gMailingList           []*mail.Email    = []*mail.Email{}
	gSendGridClient        *sendgrid.Client = nil
	gSenderEmail           *mail.Email      = nil
	gSetupSendGridOnceFlag sync.Once
)

// SetupSendGrid sets up the SendGrid mailing client.
// Can only occur once.
func SetupSendGrid() {
	gSetupSendGridOnceFlag.Do(func() {
		// Setup the sendgrid client
		gSendGridClient = sendgrid.NewSendClient(*flags.SendGridApiKey)

		// Setup the from email
		gSenderEmail = mail.NewEmail(*flags.SendGridFrom, *flags.SendGridFromEmail)

		// Setup the mailing list
		for email := range strings.SplitSeq(*flags.SendGridMailingList, ",") {
			email, err := mail.ParseEmail(email)
			if err != nil {
				panic(fmt.Sprintf("Invalid sendgrid mailing list email address: %s!", email))
			}

			gMailingList = append(gMailingList, email)
		}

		if len(gMailingList) == 0 {
			panic("Invalid sendgrid mailing list, no emails found!")
		}

		glog.Info("SendGrid setup complete!")
	})
}
