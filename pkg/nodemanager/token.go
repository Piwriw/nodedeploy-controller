package nodemanager

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcertutil "k8s.io/client-go/util/cert"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	"time"

	"strings"
)

type BootstrapTokenString struct {
	ID     string `json:"-"`
	Secret string `json:"-" datapolicy:"token"`
}
type BootstrapToken struct {
	// Token is used for establishing bidirectional trust between nodes and control-planes.
	// Used for joining nodes in the cluster.
	Token *BootstrapTokenString `json:"token" datapolicy:"token"`
	// Description sets a human-friendly message why this token exists and what it's used
	// for, so other administrators can know its purpose.
	Description string `json:"description,omitempty"`
	// TTL defines the time to live for this token. Defaults to 24h.
	// Expires and TTL are mutually exclusive.
	TTL *metav1.Duration `json:"ttl,omitempty"`
	// Expires specifies the timestamp when this token expires. Defaults to being set
	// dynamically at runtime based on the TTL. Expires and TTL are mutually exclusive.
	Expires *metav1.Time `json:"expires,omitempty"`
	// Usages describes the ways in which this token can be used. Can by default be used
	// for establishing bidirectional trust, but that can be changed here.
	Usages []string `json:"usages,omitempty"`
	// Groups specifies the extra groups that this token will authenticate as when/if
	// used for authentication
	Groups []string `json:"groups,omitempty"`
}

type CloudCoreConfig struct {
	Modules struct {
		CloudHub struct {
			AdvertiseAddress []string `yaml:"advertiseAddress"`
			Websocket        struct {
				Address string `yaml:"address"`
				Enable  bool   `yaml:"enable"`
				Port    string `yaml:"port"`
			} `yaml:"websocket"`
		} `yaml:"cloudHub"`
	} `yaml:"modules"`
}

func GetKubeEdgeJoinInfo() (cloudHost []string, Port string, Token string, err error) {
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, "", "", errors.Errorf("Get InClusterConfig Failed,err:%s ", err)
	}
	clientSet, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return nil, "", "", errors.Errorf("Create ClientSet  By InClusterConfig Failed,err:%s ", err)
	}

	cloudcoreCfg, err := clientSet.CoreV1().ConfigMaps("kubeedge").Get(context.TODO(), "cloudcore", metav1.GetOptions{})
	if err != nil {
		return nil, "", "", err
	}
	bytesCfg := []byte(cloudcoreCfg.Data["cloudcore.yaml"])
	cloudCfg := &CloudCoreConfig{}
	err = yaml.Unmarshal(bytesCfg, cloudCfg)
	if err != nil {
		return nil, "", "", err
	}
	secret, err := clientSet.CoreV1().Secrets("kubeedge").Get(context.Background(), "tokensecret", metav1.GetOptions{})
	if err != nil {
		return
	}
	token := string(secret.Data["tokendata"])
	return cloudCfg.Modules.CloudHub.AdvertiseAddress, cloudCfg.Modules.CloudHub.Websocket.Port, token, nil
}
func GetWorkJoinInfo() (masterHostAndPort, token, certHash string, err error) {
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return "", "", "", errors.Errorf("Get InClusterConfig Failed,err:%s ", err)
	}
	clientSet, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return "", "", "", errors.Errorf("Create ClientSet  By InClusterConfig Failed,err:%s ", err)
	}
	K8sConfMap, err := clientSet.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "cluster-info", metav1.GetOptions{})
	if err != nil {
		return "", "", "", errors.Errorf("Get K8sConfigMap Failed,err:%s ", err)
	}
	kubeconfigBytes := []byte(K8sConfMap.Data["kubeconfig"])
	fromInClusterConfig, err := GetClusterFromInClusterConfig(kubeconfigBytes)
	if err != nil {
		return "", "", "", errors.Errorf("Get GetClusterFromInClusterConfig Failed,err:%s ", err)
	}
	_, clusterConfig := GetClusterFromKubeConfig(fromInClusterConfig)
	if clusterConfig == nil {
		return "", "", "", errors.New("failed to get default cluster config")
	}
	masterHostAndPort = strings.Replace(clusterConfig.Server, "https://", "", -1)

	// load CA certificates from the kubeconfig (either from PEM data or by file path)
	var caCerts []*x509.Certificate
	if clusterConfig.CertificateAuthorityData != nil {
		caCerts, err = clientcertutil.ParseCertsPEM(clusterConfig.CertificateAuthorityData)
		if err != nil {
			return "", "", "", errors.Wrap(err, "failed to parse CA certificate from kubeconfig")
		}
	} else if clusterConfig.CertificateAuthority != "" {
		caCerts, err = clientcertutil.CertsFromFile(clusterConfig.CertificateAuthority)
		if err != nil {
			return "", "", "", errors.Wrap(err, "failed to load CA certificate referenced by kubeconfig")
		}
	} else {
		return "", "", "", errors.New("no CA certificates found in kubeconfig")
	}

	// hash all the CA certs and include their public key pins as trusted values
	publicKeyPins := make([]string, 0, len(caCerts))
	for _, caCert := range caCerts {
		publicKeyPins = append(publicKeyPins, Hash(caCert))
	}
	token, err = GetToken(clientSet)
	if err != nil {
		return "", "", "", errors.Wrap(err, "get Token failed")
	}

	return masterHostAndPort, token, publicKeyPins[0], nil
}

// BootstrapTokenToSecret 通过BootstrapToken去创建Secret
func BootstrapTokenToSecret(bt *BootstrapToken) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstraputil.BootstrapTokenSecretName(bt.Token.ID),
			Namespace: metav1.NamespaceSystem,
		},
		Type: bootstrapapi.SecretTypeBootstrapToken,
		Data: encodeTokenSecretData(bt, time.Now()),
	}
}

// encodeTokenSecretData takes the token discovery object and an optional duration and returns the .Data for the Secret
// now is passed in order to be able to used in unit testing
func encodeTokenSecretData(token *BootstrapToken, now time.Time) map[string][]byte {
	data := map[string][]byte{
		bootstrapapi.BootstrapTokenIDKey:     []byte(token.Token.ID),
		bootstrapapi.BootstrapTokenSecretKey: []byte(token.Token.Secret),
	}

	if len(token.Description) > 0 {
		data[bootstrapapi.BootstrapTokenDescriptionKey] = []byte(token.Description)
	}

	// If for some strange reason both token.TTL and token.Expires would be set
	// (they are mutually exclusive in validation so this shouldn't be the case),
	// token.Expires has higher priority, as can be seen in the logic here.
	if token.Expires != nil {
		// Format the expiration date accordingly
		// TODO: This maybe should be a helper function in bootstraputil?
		expirationString := token.Expires.Time.UTC().Format(time.RFC3339)
		data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(expirationString)

	} else if token.TTL != nil && token.TTL.Duration > 0 {
		// Only if .Expires is unset, TTL might have an effect
		// Get the current time, add the specified duration, and format it accordingly
		expirationString := now.Add(token.TTL.Duration).UTC().Format(time.RFC3339)
		data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(expirationString)
	}

	for _, usage := range token.Usages {
		data[bootstrapapi.BootstrapTokenUsagePrefix+usage] = []byte("true")
	}

	if len(token.Groups) > 0 {
		data[bootstrapapi.BootstrapTokenExtraGroupsKey] = []byte(strings.Join(token.Groups, ","))
	}
	return data
}

// CreateOrUpdateSecret creates a Secret if the target resource doesn't exist. If the resource exists already, this function will update the resource instead.
func CreateOrUpdateSecret(client clientset.Interface, secret *v1.Secret) error {

	if _, err := client.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create secret")
		}

		if _, err := client.CoreV1().Secrets(secret.ObjectMeta.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update secret")
		}
	}
	return nil
}

// GetToken 获取Kubeadm 的 Token
func GetToken(clientSet *kubernetes.Clientset) (string, error) {
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", errors.Wrap(err, "couldn't generate random token")
	}
	tokenBoot, err := NewBootstrapToken(tokenStr)
	if err != nil {
		return "", err
	}
	secretName := bootstraputil.BootstrapTokenSecretName(tokenBoot.Token.ID)
	secret, err := clientSet.CoreV1().Secrets(metav1.NamespaceSystem).Get(context.TODO(), secretName, metav1.GetOptions{})
	if secret != nil && err == nil {
		return "", errors.Errorf("a token with id %q already exists", tokenBoot.Token.ID)
	}
	tokenBoot.TTL = &metav1.Duration{Duration: 24 * time.Hour}
	tokenBoot.Usages = []string{"signing", "authentication"}
	tokenBoot.Groups = []string{"system:bootstrappers:kubeadm:default-node-token"}
	secret = BootstrapTokenToSecret(tokenBoot)
	err = CreateOrUpdateSecret(clientSet, secret)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", tokenBoot.Token.ID, tokenBoot.Token.Secret), nil
}

// NewBootstrapToken 获取BootstrapToken
func NewBootstrapToken(token string) (*BootstrapToken, error) {
	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(token)
	if len(substrs) != 3 {
		return nil, errors.Errorf("the bootstrap token %q was not of the form %q", token, bootstrapapi.BootstrapTokenPattern)
	}
	return &BootstrapToken{Token: &BootstrapTokenString{ID: substrs[1], Secret: substrs[2]}}, nil
}
func GetClusterFromInClusterConfig(kubeconfigBytes []byte) (*clientcmdapi.Config, error) {

	config, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return nil, err
	}

	// set LocationOfOrigin on every Cluster, User, and Context
	for key, obj := range config.AuthInfos {
		config.AuthInfos[key] = obj
	}
	for key, obj := range config.Clusters {
		config.Clusters[key] = obj
	}
	for key, obj := range config.Contexts {
		config.Contexts[key] = obj
	}

	if config.AuthInfos == nil {
		config.AuthInfos = map[string]*clientcmdapi.AuthInfo{}
	}
	if config.Clusters == nil {
		config.Clusters = map[string]*clientcmdapi.Cluster{}
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*clientcmdapi.Context{}
	}

	return config, nil
}

// GetClusterFromKubeConfig returns the default Cluster of the specified KubeConfig
func GetClusterFromKubeConfig(config *clientcmdapi.Config) (string, *clientcmdapi.Cluster) {
	// If there is an unnamed cluster object, use it
	if config.Clusters[""] != nil {
		return "", config.Clusters[""]
	}
	currentContext := config.Contexts[config.CurrentContext]
	if currentContext != nil {
		return currentContext.Cluster, config.Clusters[currentContext.Cluster]
	}
	return "", nil
}

func Hash(certificate *x509.Certificate) string {
	spkiHash := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	return "sha256" + ":" + strings.ToLower(hex.EncodeToString(spkiHash[:]))
}
