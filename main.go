package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/signal"

	prismgoclient "github.com/nutanix-cloud-native/prism-go-client"
	localEnv "github.com/nutanix-cloud-native/prism-go-client/environment/providers/local"
	envTypes "github.com/nutanix-cloud-native/prism-go-client/environment/types"
	"github.com/nutanix-cloud-native/prism-go-client/utils"
	v3client "github.com/nutanix-cloud-native/prism-go-client/v3"
)

const (
	defaultVMName           = "cluster-api-nutanix-provider"
	defaultVMDiskSizeMib    = 8192
	defaultVMMemoryMib      = 4096
	defaultVMSockets        = 2
	defaultVMVcpusPerSocket = 2
)

func main() {
	var (
		vmName           string
		vmImageName      string
		vmNutanixCluster string
		vmSubnet         string

		kubeconfig string
		namespace  string
	)

	flag.StringVar(&vmName, "vm-name", defaultVMName, "VM name.")
	flag.StringVar(&vmImageName, "vm-image-name", "", "VM image to use.")
	flag.StringVar(&vmNutanixCluster, "vm-nutanix-cluster", "", "VM nutanix cluster.")
	flag.StringVar(&vmSubnet, "vm-subnet", "", "VM subnet.")

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig file to pass to the CAPX controller.")
	flag.StringVar(&namespace, "namespace", "", "Namespace in which CAPX reconciles Cluster resources.")

	flag.Parse()

	if kubeconfig == "" {
		fmt.Fprintln(os.Stderr, "-kubeconfig must be a kubeconfig file to pass to the CAPX controller.")
		defer os.Exit(1)
		return
	}
	if namespace == "" {
		fmt.Fprintln(os.Stderr, "-namespace must be the namespace in which CAPX reconciles Cluster resources.")
		defer os.Exit(1)
		return
	}
	if vmImageName == "" {
		fmt.Fprintln(os.Stderr, "-vm-image-name must be the name of the VM image to use.")
		defer os.Exit(1)
		return

	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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
		vmImageName:           vmImageName,
		vmNutanixCluster:      vmNutanixCluster,
		vmSubnet:              vmSubnet,
		creds:                 creds,
		additionalTrustBundle: additionalTrustBundle,
		kubeconfigContents:    kubeconfigContents,
	}
	uuid, err := CreateVM(ctx, client, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create VM:", err)
		defer os.Exit(1)
		return
	}
	fmt.Fprintln(os.Stderr, "VM created")
	fmt.Fprintln(os.Stderr, "UUID:", uuid)
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
	vmImageName           string
	vmNutanixCluster      string
	vmSubnet              string
	kubeconfigContents    []byte
	creds                 *prismgoclient.Credentials
	additionalTrustBundle string
}

func CreateVM(ctx context.Context, client *v3client.Client, opts *VMOptions) (string, error) {
	peUUID, err := GetPEUUID(ctx, client, &opts.vmNutanixCluster, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get UUID for cluster %s: %w", opts.vmNutanixCluster, err)
	}
	subnetUUID, err := GetSubnetUUID(ctx, client, peUUID, &opts.vmSubnet, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get UUID for subnet %s: %w", opts.vmSubnet, err)
	}
	imageUUID, err := GetImageUUID(ctx, client, &opts.vmImageName, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get UUID for image %s: %w", opts.vmImageName, err)
	}

	metadata := fmt.Sprintf("{\"hostname\": \"%s\"}", opts.vmName)
	// TODO Create userdata.
	userdata := ""

	input := &v3client.VMIntentInput{
		Metadata: &v3client.Metadata{
			Kind:        utils.StringPtr("vm"),
			SpecVersion: utils.Int64Ptr(1),
		},
		Spec: &v3client.VM{
			Name: utils.StringPtr(opts.vmName),
			ClusterReference: &v3client.Reference{
				Kind: utils.StringPtr("cluster"),
				UUID: utils.StringPtr(peUUID),
			},
			Resources: &v3client.VMResources{
				BootConfig: &v3client.VMBootConfig{
					BootType: utils.StringPtr("UEFI"),
				},
				NicList: []*v3client.VMNic{
					{
						SubnetReference: &v3client.Reference{
							UUID: utils.StringPtr(subnetUUID),
							Kind: utils.StringPtr("subnet"),
						},
					},
				},
				DiskList: []*v3client.VMDisk{
					{
						DataSourceReference: &v3client.Reference{
							Kind: utils.StringPtr("image"),
							UUID: utils.StringPtr(imageUUID),
						},
						DiskSizeMib: utils.Int64Ptr(defaultVMDiskSizeMib),
					},
				},
				MemorySizeMib:         utils.Int64Ptr(defaultVMMemoryMib),
				NumVcpusPerSocket:     utils.Int64Ptr(defaultVMVcpusPerSocket),
				NumSockets:            utils.Int64Ptr(defaultVMSockets),
				PowerState:            utils.StringPtr("ON"),
				HardwareClockTimezone: utils.StringPtr("UTC"),
				GuestCustomization: &v3client.GuestCustomization{
					IsOverridable: utils.BoolPtr(true),
					CloudInit: &v3client.GuestCustomizationCloudInit{
						UserData: utils.StringPtr(base64.StdEncoding.EncodeToString([]byte(userdata))),
						MetaData: utils.StringPtr(base64.StdEncoding.EncodeToString([]byte(metadata))),
					},
				},
			},
		},
	}
	resp, err := client.V3.CreateVM(ctx, input)
	if err != nil {
		return "", err
	}
	return *resp.Metadata.UUID, nil
}
