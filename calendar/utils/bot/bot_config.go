// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package bot

type Config struct {
	// AdminUserIDs contains a comma-separated list of user IDs that are allowed
	// to administer plugin functions, even if not Mattermost sysadmins.
	AdminUserIDs string

	// AdminLogLevel is "debug", "info", "warn", or "error".
	AdminLogLevel string

	// AdminLogVerbose: set to include full context with admin log messages.
	AdminLogVerbose bool
}

func (c Config) ToStorableConfig(configMap map[string]interface{}) map[string]interface{} {
	if configMap == nil {
		configMap = map[string]interface{}{}
	}
	configMap["AdminUserIDs"] = c.AdminUserIDs
	configMap["AdminLogLevel"] = c.AdminLogLevel
	configMap["AdminLogVerbose"] = c.AdminLogVerbose
	return configMap
}
