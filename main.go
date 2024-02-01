package main

import (
	"flag"
	"fmt"
	"os"

	prismgoclient "github.com/nutanix-cloud-native/prism-go-client"
	localEnv "github.com/nutanix-cloud-native/prism-go-client/environment/providers/local"
	envTypes "github.com/nutanix-cloud-native/prism-go-client/environment/types"
	v3client "github.com/nutanix-cloud-native/prism-go-client/v3"
)

const (
	defaultVMName     = "cluster-api-nutanix-provider"
	defaultVMImageURL = "https://cloud-images.ubuntu.com/minimal/releases/jammy/release-20240125/ubuntu-22.04-minimal-cloudimg-amd64.img"
)

func main() {
	var (
		vmName           string
		vmImageURL       string
		vmNutanixCluster string
		vmSubnet         string

		kubeconfig string
		namespace  string
	)

	flag.StringVar(&vmName, "vm-name", defaultVMName, "VM name.")
	flag.StringVar(&vmImageURL, "vm-image-url", defaultVMImageURL, "Disk image for VM root filesystem.")
	flag.StringVar(&vmNutanixCluster, "vm-nutanix-cluser", "", "VM nutanix cluster.")
	flag.StringVar(&vmSubnet, "vm-subnet", "", "VM subnet.")

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig file to pass to the CAPX controller.")
	flag.StringVar(&namespace, "namespace", "", "Namespace in which CAPX reconciles Cluster resources.")

	flag.Parse()

	if kubeconfig == "" {
		fmt.Fprintln(os.Stderr, "--kubeconfig must be a kubeconfig file to pass to the CAPX controller.")
		defer os.Exit(1)
		return
	}
	if namespace == "" {
		fmt.Fprintln(os.Stderr, "--namespace must be the namespace in which CAPX reconciles Cluster resources.")
		defer os.Exit(1)
		return
	}

	fmt.Fprintln(os.Stderr, "Reading kubeconfig...")
	kubeconfigContents, err := os.ReadFile(kubeconfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read kubeconfig:", err)
		defer os.Exit(1)
		return
	}
	fmt.Fprintln(os.Stderr, "Read kubeconfig")

	fmt.Fprintln(os.Stderr, "Reading Prism Central credentials from the local environment...")
	creds, additionalTrustBundle, err := ConfigFromLocalEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read credentials:", err)
		defer os.Exit(1)
		return
	}
	fmt.Fprintln(os.Stderr, "Client created")

	fmt.Fprintln(os.Stderr, "Creating Prism Central client...")
	client, err := CreateClient(creds, additionalTrustBundle)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create client:", err)
		defer os.Exit(1)
		return
	}
	fmt.Fprintln(os.Stderr, "Client created")

	fmt.Fprintln(os.Stderr, "Creating VM...")

	opts := &VMOptions{
		vmName:                vmName,
		vmImageURL:            vmImageURL,
		vmNutanixCluster:      vmNutanixCluster,
		vmSubnet:              vmSubnet,
		creds:                 creds,
		additionalTrustBundle: additionalTrustBundle,
		kubeconfigContents:    kubeconfigContents,
	}
	err = CreateVM(client, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create VM:", err)
		defer os.Exit(1)
		return
	}
	fmt.Fprintln(os.Stderr, "VM created")
}

func ConfigFromLocalEnv() (*prismgoclient.Credentials, string, error) {
	provider := localEnv.NewProvider()

	me, err := provider.GetManagementEndpoint(envTypes.Topology{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get management endpoint: %w", err)
	}

	return &prismgoclient.Credentials{
		URL:      me.Address.Host,
		Endpoint: me.Address.Host,
		Insecure: me.Insecure,
		Username: me.ApiCredentials.Username,
		Password: me.ApiCredentials.Password,
	}, me.AdditionalTrustBundle, nil
}

func CreateClient(creds *prismgoclient.Credentials, additionalTrustBundle string) (*v3client.Client, error) {
	opts := []v3client.ClientOption{}
	if len(additionalTrustBundle) > 0 {
		opts = append(opts, v3client.WithPEMEncodedCertBundle([]byte(additionalTrustBundle)))
	}

	return v3client.NewV3Client(*creds, opts...)
}

type VMOptions struct {
	vmName                string
	vmImageURL            string
	vmNutanixCluster      string
	vmSubnet              string
	kubeconfigContents    []byte
	creds                 *prismgoclient.Credentials
	additionalTrustBundle string
}

func CreateVM(client *v3client.Client, opts *VMOptions) error {
	
	return nil
}
