package emailtemplate

type CommonContext struct {
	AppName string
	BaseURL string
}

func NewCommonContext(appName, baseURL string) CommonContext {
	return CommonContext{
		AppName: appName,
		BaseURL: baseURL,
	}
}
