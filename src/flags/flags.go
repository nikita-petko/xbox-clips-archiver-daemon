package flags

import (
	"flag"
	"time"
)

var (
	// HelpFlag prints the usage.
	HelpFlag = flag.Bool("help", false, "Print usage.")

	// Interval is the interval to wait between each request. (environment variable: INTERVAL)
	Interval = flag.Duration("interval", 5*time.Minute, "Interval to wait between each request. (environment variable: INTERVAL)")

	// Pulse determines if the daemon should run once and exit.
	Pulse = flag.Bool("pulse", false, "Run once and exit.")

	//// Unique Flags

	// CachePath is the path to the cache directory. Defaults to .cache. This is required. (environment variable: CACHE_PATH)
	CachePath = flag.String("cache-path", ".cache", "The path to the cache directory. Defaults to .cache. This is required. (environment variable: CACHE_PATH)")

	// ClearCache determines if the cache should be cleared before starting the daemon. (environment variable: CLEAR_CACHE)
	ClearCache = flag.Bool("clear-cache", false, "Clear the cache before starting the daemon. (environment variable: CLEAR_CACHE)")

	// ShouldLockCache determines if the cache should be locked before starting the daemon. (environment variable: SHOULD_LOCK_CACHE)
	ShouldLockCache = flag.Bool("should-lock-cache", true, "Determines if the cache should be locked before starting the daemon. (environment variable: SHOULD_LOCK_CACHE)")

	// XblApiKeys is a comma-separated list of Xbox API keys. This is required. (environment variable: XBL_API_KEYS)
	XblApiKeys = flag.String("xbl-api-keys", "", "A comma-separated list of Xbox API keys. This is required. (environment variable: XBL_API_KEYS)")

	//// Alerting

	// SendGridApiKey is the SendGrid API key. This is optional. (environment variable: SENDGRID_API_KEY)
	SendGridApiKey = flag.String("sendgrid-api-key", "", "The SendGrid API key. This is optional. (environment variable: SENDGRID_API_KEY)")

	// SendGridFrom is the name to use as the sender. This is required if the API Key is specified. (environment variable: SENDGRID_FROM)
	SendGridFrom = flag.String("sendgrid-from", "", "The name to use as the sender. This is required if the API Key is specified. (environment variable: SENDGRID_FROM)")

	// SendGridFromEmail is the email address to use as the sender. This is required if the API Key is specified. (environment variable: SENDGRID_FROM_EMAIL)
	SendGridFromEmail = flag.String("sendgrid-from-email", "", "The email address to use as the sender. This is required if the API Key is specified. (environment variable: SENDGRID_FROM_EMAIL)")

	// SendGridMailingList is the mailing list to send the emails to. This is required if the API Key is specified. (environment variable: SENDGRID_MAILING_LIST)
	SendGridMailingList = flag.String("sendgrid-mailing-list", "", "The mailing list to send the emails to. This is required if the API Key is specified. (environment variable: SENDGRID_MAILING_LIST)")

	// SnsTopicArn is yhe ARN to the topic created in AWS SNS. This is optional. Needs AWS_ACCESS_KEY and AWS_SECRET_ACCESS_KEY. (environment variable: SNS_TOPIC_ARN)
	SnsTopicArn = flag.String("sns-topic-arn", "", "The ARN to the topic created in AWS SNS. This is optional. Needs AWS_ACCESS_KEY and AWS_SECRET_ACCESS_KEY. (environment variable: SNS_TOPIC_ARN)")

	// AwsCredentialsFromProfile will load the AWS credentials from the system profile instead of environment variables. (enviornment variable: AWS_CREDENTIALS_FROM_PROFILE)
	AwsCredentialsFromProfile = flag.Bool("aws-credentials-from-profile", false, "Is the AWS SNS Credentials coming from a profile file? If not use enviornment variables. (environment variable: AWS_CREDENTIALS_FROM_PROFILE)")

	// DiscordWebHookUri is the url that was generated when creating a Discord WebHook. (environment variable: DISCORD_WEBHOOK_URI)
	DiscordWebHookUri = flag.String("discord-webhook-uri", "", "The url that was generated when creating a Discord WebHook. (environment variable: DISCORD_WEBHOOK_URI)")

	// DiscordAlertRoleId is the ID of the role that should be pinged when an alert is sent. (environment variable: DISCORD_ALERT_ROLE_ID)
	DiscordAlertRoleId = flag.Uint64("discord-alert-role-id", 0, "The ID of the role that should be pinged when an alert is sent. (environment variable: DISCORD_ALERT_ROLE_ID)")
)

// FlagUsageString is the usage string printed a longside the help command.
const FlagsUsageString string = `
	[-h|--help] [--interval[=5m]] [--pulse]
	[--cache-path[=]] [--clear-cache[=false]]
	[--url[=]] [--asset-prefix-override[=]] [--build-id-override[=]] [--fetch-mode[=pages]]
	[--sendgrid-api-key[=]] [--sendgrid-from[=]] [--sendgrid-from-email[=]] [--sendgrid-mailing-list[=]]
	[--sns-topic-arn[=]] [--aws-credentials-from-profile[=false]]
	[--discord-webhook-uri[=]] [--discord-alert-role-id[=0]]`
