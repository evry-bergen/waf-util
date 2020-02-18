package director

import (
	"fmt"
)

type TerminationTarget struct {
	Hosts     []string
	Port      int
	Secret    string
	Namespace string
	Target    string
}

func (t TerminationTarget) generateNameWithPrefix(prefix string, hostname string) string {
	name := fmt.Sprintf("%s-%s", prefix, hostname)
	return name
}

func (t TerminationTarget) generateSecretName(prefix string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, t.Namespace, t.Secret)
}
