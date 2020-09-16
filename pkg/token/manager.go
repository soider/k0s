package token

import (
	"fmt"
	"time"

	k8sutil "github.com/Mirantis/mke/pkg/kubernetes"
	"github.com/Mirantis/mke/pkg/util"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NewManager creates a new token manager using given kubeconfig
func NewManager(kubeconfig string) (*Manager, error) {
	logrus.Debugf("loading kubeconfig from: %s", kubeconfig)
	client, err := k8sutil.Client(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &Manager{
		client: client,
	}, nil
}

// Manager is responsible to manage the join tokens in kube API as secrets in kube-system namespace
type Manager struct {
	client *kubernetes.Clientset
}

// Create creates a new bootstrap token
func (m *Manager) Create(valid time.Duration, role string) (string, error) {
	tokenID := util.RandomString(6)
	tokenSecret := util.RandomString(16)

	token := fmt.Sprintf("%s.%s", tokenID, tokenSecret)

	data := make(map[string]string)
	data["token-id"] = tokenID
	data["token-secret"] = tokenSecret
	if valid != 0 {
		data["expiration"] = time.Now().Add(valid).UTC().Format(time.RFC3339)
		logrus.Debugf("Set expiry to %s", data["expiration"])
	}

	if role == "worker" {
		data["description"] = "Worker bootstrap token generated by mke"
		data["usage-bootstrap-authentication"] = "true"
		data["usage-bootstrap-signing"] = "true"
	} else {
		data["description"] = "Controller bootstrap token generated by mke"
		data["usage-bootstrap-authentication"] = "false"
		data["usage-bootstrap-signing"] = "false"
		data["usage-controller-join"] = "true"
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("bootstrap-token-%s", tokenID),
			Namespace: "kube-system",
		},
		Type:       v1.SecretTypeBootstrapToken,
		StringData: data,
	}

	_, err := m.client.CoreV1().Secrets("kube-system").Create(secret)
	if err != nil {
		return "", err
	}

	return token, nil
}
