package flags

func applyEnvironmentVariableFlags() {
	getEnvironmentVariableOrFlag("INTERVAL", Interval)

	getEnvironmentVariableOrFlag("SEND_GRID_API_KEY", SendGridApiKey)
	getEnvironmentVariableOrFlag("SEND_GRID_FROM", SendGridFrom)
	getEnvironmentVariableOrFlag("SEND_GRID_FROM_EMAIL", SendGridFromEmail)
	getEnvironmentVariableOrFlag("SEND_GRID_MAILING_LIST", SendGridMailingList)

	getEnvironmentVariableOrFlag("SNS_TOPIC_ARN", SnsTopicArn)
	getEnvironmentVariableOrFlag("AWS_CREDENTIALS_FROM_PROFILE", AwsCredentialsFromProfile)

	getEnvironmentVariableOrFlag("DISCORD_WEBHOOK_URI", DiscordWebHookUri)
	getEnvironmentVariableOrFlag("DISCORD_ALERT_ROLE_ID", DiscordAlertRoleId)
}
