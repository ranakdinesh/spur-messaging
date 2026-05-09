package services

// resolveString returns the first non-empty value in priority order:
// API request → tenant provider_config → platform default
func resolveString(apiValue, tenantValue, platformDefault string) string {
	if apiValue != "" {
		return apiValue
	}
	if tenantValue != "" {
		return tenantValue
	}
	return platformDefault
}

// resolveBool returns the first non-nil value in priority order.
// If all are nil, returns platformDefault.
func resolveBool(apiValue *bool, tenantValue *bool, platformDefault bool) bool {
	if apiValue != nil {
		return *apiValue
	}
	if tenantValue != nil {
		return *tenantValue
	}
	return platformDefault
}

// resolveInt returns the first non-zero value in priority order.
func resolveInt(apiValue, tenantValue, platformDefault int) int {
	if apiValue > 0 {
		return apiValue
	}
	if tenantValue > 0 {
		return tenantValue
	}
	return platformDefault
}
