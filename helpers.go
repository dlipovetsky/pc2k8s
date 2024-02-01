package main

// Adapted from https://github.com/nutanix-cloud-native/cluster-api-provider-nutanix/blob/5fd7ada9b99ab7f3b2b48dd15ae186c017296c4b/controllers/helpers.go

import (
	"context"
	"fmt"
	"strings"

	v3client "github.com/nutanix-cloud-native/prism-go-client/v3"

	"github.com/nutanix-cloud-native/prism-go-client/utils"
)

const (
	serviceNamePECluster = "AOS"

	subnetTypeOverlay = "OVERLAY"
)

// FindVMByName retrieves the VM with the given vm name
func FindVMByName(ctx context.Context, client *v3client.Client, vmName string) (*v3client.VMIntentResponse, error) {
	res, err := client.V3.ListVM(ctx, &v3client.DSMetadata{
		Filter: utils.StringPtr(fmt.Sprintf("vm_name==%s", vmName)),
	})
	if err != nil {
		return nil, err
	}

	if len(res.Entities) > 1 {
		return nil, fmt.Errorf("error: found more than one (%v) vms with name %s", len(res.Entities), vmName)
	}

	if len(res.Entities) == 0 {
		return nil, nil
	}

	return FindVMByUUID(ctx, client, *res.Entities[0].Metadata.UUID)
}

// FindVMByUUID retrieves the VM with the given vm UUID. Returns nil if not found
func FindVMByUUID(ctx context.Context, client *v3client.Client, uuid string) (*v3client.VMIntentResponse, error) {
	response, err := client.V3.GetVM(ctx, uuid)
	if err != nil {
		if strings.Contains(fmt.Sprint(err), "ENTITY_NOT_FOUND") {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return response, nil
}

// GetPEUUID returns the UUID of the Prism Element cluster with the given name
func GetPEUUID(ctx context.Context, client *v3client.Client, peName, peUUID *string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("cannot retrieve Prism Element UUID if nutanix client is nil")
	}
	if peUUID == nil && peName == nil {
		return "", fmt.Errorf("cluster name or uuid must be passed in order to retrieve the Prism Element UUID")
	}
	if peUUID != nil && *peUUID != "" {
		peIntentResponse, err := client.V3.GetCluster(ctx, *peUUID)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "ENTITY_NOT_FOUND") {
				return "", fmt.Errorf("failed to find Prism Element cluster with UUID %s: %v", *peUUID, err)
			}
		}
		return *peIntentResponse.Metadata.UUID, nil
	} else if peName != nil && *peName != "" {
		filter := getFilterForName(*peName)
		responsePEs, err := client.V3.ListAllCluster(ctx, filter)
		if err != nil {
			return "", err
		}
		// Validate filtered PEs
		foundPEs := make([]*v3client.ClusterIntentResponse, 0)
		for _, s := range responsePEs.Entities {
			peSpec := s.Spec
			if *peSpec.Name == *peName && hasPEClusterServiceEnabled(s, serviceNamePECluster) {
				foundPEs = append(foundPEs, s)
			}
		}
		if len(foundPEs) == 1 {
			return *foundPEs[0].Metadata.UUID, nil
		}
		if len(foundPEs) == 0 {
			return "", fmt.Errorf("failed to retrieve Prism Element cluster by name %s", *peName)
		} else {
			return "", fmt.Errorf("more than one Prism Element cluster found with name %s", *peName)
		}
	}
	return "", fmt.Errorf("failed to retrieve Prism Element cluster by name or uuid. Verify input parameters")
}

// GetSubnetUUID returns the UUID of the subnet with the given name
func GetSubnetUUID(ctx context.Context, client *v3client.Client, peUUID string, subnetName, subnetUUID *string) (string, error) {
	var foundSubnetUUID string
	if subnetUUID == nil && subnetName == nil {
		return "", fmt.Errorf("subnet name or subnet uuid must be passed in order to retrieve the subnet")
	}
	if subnetUUID != nil {
		subnetIntentResponse, err := client.V3.GetSubnet(ctx, *subnetUUID)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "ENTITY_NOT_FOUND") {
				return "", fmt.Errorf("failed to find subnet with UUID %s: %v", *subnetUUID, err)
			}
		}
		foundSubnetUUID = *subnetIntentResponse.Metadata.UUID
	} else if subnetName != nil {
		filter := getFilterForName(*subnetName)
		// Not using additional filtering since we want to list overlay and vlan subnets
		responseSubnets, err := client.V3.ListAllSubnet(ctx, filter, nil)
		if err != nil {
			return "", err
		}
		// Validate filtered Subnets
		foundSubnets := make([]*v3client.SubnetIntentResponse, 0)
		for _, subnet := range responseSubnets.Entities {
			if subnet == nil || subnet.Spec == nil || subnet.Spec.Name == nil || subnet.Spec.Resources == nil || subnet.Spec.Resources.SubnetType == nil {
				continue
			}
			if *subnet.Spec.Name == *subnetName {
				if *subnet.Spec.Resources.SubnetType == subnetTypeOverlay {
					// Overlay subnets are present on all PEs managed by PC.
					foundSubnets = append(foundSubnets, subnet)
				} else {
					// By default check if the PE UUID matches if it is not an overlay subnet.
					if *subnet.Spec.ClusterReference.UUID == peUUID {
						foundSubnets = append(foundSubnets, subnet)
					}
				}
			}
		}
		if len(foundSubnets) == 0 {
			return "", fmt.Errorf("failed to retrieve subnet by name %s", *subnetName)
		} else if len(foundSubnets) > 1 {
			return "", fmt.Errorf("more than one subnet found with name %s", *subnetName)
		} else {
			foundSubnetUUID = *foundSubnets[0].Metadata.UUID
		}
		if foundSubnetUUID == "" {
			return "", fmt.Errorf("failed to retrieve subnet by name or uuid. Verify input parameters")
		}
	}
	return foundSubnetUUID, nil
}

// GetImageUUID returns the UUID of the image with the given name
func GetImageUUID(ctx context.Context, client *v3client.Client, imageName, imageUUID *string) (string, error) {
	var foundImageUUID string

	if imageUUID == nil && imageName == nil {
		return "", fmt.Errorf("image name or image uuid must be passed in order to retrieve the image")
	}
	if imageUUID != nil {
		imageIntentResponse, err := client.V3.GetImage(ctx, *imageUUID)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "ENTITY_NOT_FOUND") {
				return "", fmt.Errorf("failed to find image with UUID %s: %v", *imageUUID, err)
			}
		}
		foundImageUUID = *imageIntentResponse.Metadata.UUID
	} else if imageName != nil {
		filter := getFilterForName(*imageName)
		responseImages, err := client.V3.ListAllImage(ctx, filter)
		if err != nil {
			return "", err
		}
		// Validate filtered Images
		foundImages := make([]*v3client.ImageIntentResponse, 0)
		for _, s := range responseImages.Entities {
			imageSpec := s.Spec
			if *imageSpec.Name == *imageName {
				foundImages = append(foundImages, s)
			}
		}
		if len(foundImages) == 0 {
			return "", fmt.Errorf("failed to retrieve image by name %s", *imageName)
		} else if len(foundImages) > 1 {
			return "", fmt.Errorf("more than one image found with name %s", *imageName)
		} else {
			foundImageUUID = *foundImages[0].Metadata.UUID
		}
		if foundImageUUID == "" {
			return "", fmt.Errorf("failed to retrieve image by name or uuid. Verify input parameters")
		}
	}
	return foundImageUUID, nil
}

func getFilterForName(name string) string {
	return fmt.Sprintf("name==%s", name)
}

func hasPEClusterServiceEnabled(peCluster *v3client.ClusterIntentResponse, serviceName string) bool {
	if peCluster.Status == nil ||
		peCluster.Status.Resources == nil ||
		peCluster.Status.Resources.Config == nil {
		return false
	}
	serviceList := peCluster.Status.Resources.Config.ServiceList
	for _, s := range serviceList {
		if s != nil && strings.ToUpper(*s) == serviceName {
			return true
		}
	}
	return false
}
