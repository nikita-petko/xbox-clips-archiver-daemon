package alerting

import (
	"fmt"
	"net/url"
	"os"

	"github.com/golang/glog"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting/discord"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting/sg"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting/sns"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

var gAlertingEnabled bool = false

// Setup sets up alerting.
func Setup() {
	if *flags.SnsTopicArn != "" {
		if !*flags.AwsCredentialsFromProfile {
			if _, exists := os.LookupEnv("AWS_ACCESS_KEY"); !exists {
				panic("If you are using AWS SNS, you must specify AWS_ACCESS_KEY environment variable.")
			}

			if _, exists := os.LookupEnv("AWS_SECRET_ACCESS_KEY"); !exists {
				panic("If you are using AWS SNS, you must specify AWS_SECRET_ACCESS_KEY environment variable.")
			}
		}

		sns.SetupSnsTopic()

		gAlertingEnabled = true
	}

	if *flags.SendGridApiKey != "" {
		if *flags.SendGridFrom == "" {
			panic("If you are using SendGrid, you must specify SENDGRID_FROM or -sendgrid-from.")
		}

		if *flags.SendGridFromEmail == "" {
			panic("If you are using SendGrid, you must specify SENDGRID_FROM_EMAIL or -sendgrid-from-email.")
		}

		if *flags.SendGridMailingList == "" {
			panic("If you are using SendGrid, you must specify SENDGRID_MAILING_LIST or -sendgrid-mailing-list.")
		}

		_, err := mail.ParseEmail(*flags.SendGridFromEmail)

		if err != nil {
			panic(fmt.Sprintf("If you are using SendGrid, the from email you supply must be valid: %s", err))
		}

		sg.SetupSendGrid()

		glog.Infof("Setup SendGrid with From = %s, FromEmail = %s, MailingList = %s", *flags.SendGridFrom, *flags.SendGridFromEmail, *flags.SendGridMailingList)

		gAlertingEnabled = true
	}

	if *flags.DiscordWebHookUri != "" {
		_, err := url.Parse(*flags.DiscordWebHookUri)

		if err != nil {
			panic(fmt.Sprintf("If you are using Discord, the from webhook url you supply must be valid: %v", err))
		}

		discord.SetupDiscordClient()

		glog.Infof("Setup Discord webhook alerting with url: %s", *flags.DiscordWebHookUri)

		gAlertingEnabled = true
	}

	if !gAlertingEnabled {
		glog.Warning("AWS SNS, SendGrid and Discord alerting is disabled!")
	}

	glog.Infof("Alerting setup complete, alerting enabled: %v", gAlertingEnabled)
}
