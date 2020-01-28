package director

import (
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/evry-bergen/waf-syncer/pkg/config"
)

func init() {

}

func TestDirector_HasNotPrefix(t *testing.T) {
	cfg := config.AzureWafConfig{
		ListenerPrefix:      "prix",
		BackendHttpSettings: "",
		FrontendPort:        "",
		BackendPool:         "",
		Name:                "",
		ResourceGroup:       "",
		SubscriptionID:      "",
	}
	d := &Director{}
	d.AzureWafConfig = &cfg
	input := "blabla"
	result := d.HasNotPrefix(input)
	assert.Equal(t, true, result, "string has prefix")
}
