package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_urlScheme_Transfer(t *testing.T) {
	userID := UuidNewV4().String()
	url := SchemeTransfer(userID)
	assert.Equal(t, "mixin://transfer/"+userID, url)
}

func Test_urlScheme_Apps(t *testing.T) {
	t.Run("default action", func(t *testing.T) {
		appID := UuidNewV4().String()
		action := ""
		url := SchemeApps(appID, action, nil)
		assert.Equal(t, "mixin://apps/"+appID+"?action=open", url)
	})
	t.Run("specify action", func(t *testing.T) {
		appID := UuidNewV4().String()
		action := "close"
		url := SchemeApps(appID, action, nil)
		assert.Equal(t, "mixin://apps/"+appID+"?action="+action, url)
	})
	t.Run("specify params", func(t *testing.T) {
		appID := UuidNewV4().String()
		action := ""
		params := map[string]string{"k1": "v1", "k2": "v2"}
		url := SchemeApps(appID, action, params)
		assert.Contains(t, url, "mixin://apps/"+appID+"?action=open")
		assert.Contains(t, url, "k1=v1")
		assert.Contains(t, url, "k2=v2")
	})
}
