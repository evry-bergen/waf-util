package director

import "fmt"

type TerminationTarget struct {
	Host      string
	Port      int
	Secret    string
	Namespace string
	Target    string
}

func (t *TerminationTarget) generateName() string {
	return fmt.Sprintf("%s-tls", t.Host)
}

func (t TerminationTarget) generateNameWithPrefix(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, t.generateName())
}

func (t TerminationTarget) generateSecretName(prefix string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, t.Namespace, t.Secret)
}
