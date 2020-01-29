package director

import (
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/evry-bergen/waf-syncer/pkg/config"
)

func init() {

}

func TestDirector_HasNotPrefix_should_succeed(t *testing.T) {
	cfg := config.AzureWafConfig{
		ListenerPrefix: "prix",
	}
	d := &Director{}
	d.AzureWafConfig = &cfg
	input := "blabla"
	result := d.hasNotPrefix(input)
	assert.Equal(t, true, result, "string has correct prefix")
}

func TestDirector_HasNotPrefix_should_fail(t *testing.T) {
	cfg := config.AzureWafConfig{
		ListenerPrefix: "prix",
	}
	d := &Director{}
	d.AzureWafConfig = &cfg
	input := "prixblabla"
	result := d.hasNotPrefix(input)
	assert.Equal(t, false, result, "string has not correct prefix")
}
