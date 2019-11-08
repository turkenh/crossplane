package host

import (
	"os"

	"k8s.io/client-go/rest"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetHostClients is the function to get Host configuration in case workload and resource API's are different
func GetHostClients() (hostKube client.Client, hostClient kubernetes.Interface, err error) {
	hostKubeConfig := os.Getenv("HOST_KUBECONFIG")
	if hostKubeConfig != "" {
		// This is definitely not a good idea, since it can effect other controllers startup since env vars are set
		// process-wide.
		err = os.Setenv("KUBECONFIG", hostKubeConfig)
		if err != nil {
			err = errors.Wrap(err, "failed to set KUBECONFIG env var with HOST_KUBECONFIG")
			return
		}
		var cfg *rest.Config
		cfg, err = config.GetConfig()
		if err != nil {
			err = errors.Wrap(err, "failed to initialize config with HOST_KUBECONFIG")
			return
		}
		hostKube, err = client.New(cfg, client.Options{})
		if err != nil {
			err = errors.Wrap(err, "failed to initialize client with HOST_KUBECONFIG")
			return
		}
		hostClient = kubernetes.NewForConfigOrDie(cfg)
		err = os.Unsetenv("KUBECONFIG")
		if err != nil {
			err = errors.Wrap(err, "failed to unset KUBECONFIG env var after configuring with HOST_KUBECONFIG")
			return
		}
	}
	return
}
