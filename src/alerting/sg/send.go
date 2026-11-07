package sg

import (
	"context"

	"github.com/golang/glog"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendMail sends an email to the configured mailing list.
func SendMail(ctx context.Context, subject string, body string) {
	if gSendGridClient == nil {
		return
	}

	message := mail.NewV3Mail()
	message.SetFrom(gSenderEmail)
	message.Subject = subject

	content := mail.NewContent("text/plain", body)
	message.AddContent(content)

	personalization := mail.NewPersonalization()
	personalization.AddTos(gMailingList...)

	message.AddPersonalizations(personalization)

	response, err := gSendGridClient.SendWithContext(ctx, message)
	if err != nil {
		if response != nil {
			// If it's an access denied error, then just skip it
			if response.StatusCode == 401 || response.StatusCode > 500 {
				return
			}
		}

		// Some other error, needs to be reported!
		glog.Errorf("Error sending email: %v", err)
	}
}
