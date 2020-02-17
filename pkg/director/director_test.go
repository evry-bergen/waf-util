package director

import (
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/evry-bergen/waf-syncer/pkg/config"
)

func init() {

}

func TestDirector_HasPrefix_should_fail(t *testing.T) {
	cfg := config.AzureWafConfig{
		ListenerPrefix: "prix",
	}
	d := &Director{}
	d.AzureWafConfig = &cfg
	input := "blabla"
	result := d.hasPrefix(input)
	assert.Equal(t, false, result, "string has incorrect correct prefix")
}

func TestDirector_HasPrefix_should_succeed(t *testing.T) {
	cfg := config.AzureWafConfig{
		ListenerPrefix: "prix",
	}
	d := &Director{}
	d.AzureWafConfig = &cfg
	input := "prix-blalala"
	result := d.hasPrefix(input)
	assert.Equal(t, true, result, "string has correct prefix")
}
