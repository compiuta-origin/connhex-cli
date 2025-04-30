package cli

import "github.com/compiuta-origin/connhex-cli/internal/sdk"

var chxsdk sdk.SDK

func SetSDK(s sdk.SDK) {
	chxsdk = s
}
