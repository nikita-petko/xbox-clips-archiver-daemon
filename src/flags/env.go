package flags

func applyEnvironmentVariableFlags() {
	getEnvironmentVariableOrFlag("INTERVAL", Interval)

	getEnvironmentVariableOrFlag("CACHE_PATH", CachePath)
	getEnvironmentVariableOrFlag("CLEAR_CACHE", ClearCache)
	getEnvironmentVariableOrFlag("SHOULD_LOCK_CACHE", ShouldLockCache)
	getEnvironmentVariableOrFlag("XBL_API_KEYS", XblApiKeys)
	getEnvironmentVariableOrFlag("S3_BUCKET", S3Bucket)
	getEnvironmentVariableOrFlag("S3_REGION", S3Region)
	getEnvironmentVariableOrFlag("S3_ACCESS_KEY", S3AccessKey)
	getEnvironmentVariableOrFlag("S3_SECRET_ACCESS_KEY", S3SecretAccessKey)

	getEnvironmentVariableOrFlag("SEND_GRID_API_KEY", SendGridApiKey)
	getEnvironmentVariableOrFlag("SEND_GRID_FROM", SendGridFrom)
	getEnvironmentVariableOrFlag("SEND_GRID_FROM_EMAIL", SendGridFromEmail)
	getEnvironmentVariableOrFlag("SEND_GRID_MAILING_LIST", SendGridMailingList)

	getEnvironmentVariableOrFlag("SNS_TOPIC_ARN", SnsTopicArn)
	getEnvironmentVariableOrFlag("AWS_CREDENTIALS_FROM_PROFILE", AwsCredentialsFromProfile)

	getEnvironmentVariableOrFlag("DISCORD_WEBHOOK_URI", DiscordWebHookUri)
	getEnvironmentVariableOrFlag("DISCORD_ALERT_ROLE_ID", DiscordAlertRoleId)
}
