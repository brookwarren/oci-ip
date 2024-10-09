package main

import (
	"context"
	"fmt"
	"os"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <root_compartment_ocid>")
		os.Exit(1)
	}

	rootCompartmentOCID := os.Args[1]
	configProvider := common.DefaultConfigProvider()

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(configProvider)
	if err != nil {
		fmt.Printf("Error creating IdentityClient: %v\n", err)
		return
	}

	vcnClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		fmt.Printf("Error creating VirtualNetworkClient: %v\n", err)
		return
	}

	ctx := context.Background()
	findIPsInCompartments(ctx, identityClient, vcnClient, rootCompartmentOCID)
}

func findIPsInCompartments(ctx context.Context, identityClient identity.IdentityClient, vcnClient core.VirtualNetworkClient, compartmentID string) {
	// List sub-compartments
	listCompartmentsRequest := identity.ListCompartmentsRequest{
		CompartmentId:          common.String(compartmentID),
		AccessLevel:            identity.ListCompartmentsAccessLevelAccessible,
		CompartmentIdInSubtree: common.Bool(true),
	}

	listCompartmentsResponse, err := identityClient.ListCompartments(ctx, listCompartmentsRequest)
	if err != nil {
		fmt.Printf("Error listing compartments: %v\n", err)
		return
	}

	// Include the root compartment
	compartments := listCompartmentsResponse.Items
	compartments = append(compartments, identity.Compartment{Id: common.String(compartmentID)})

	for _, compartment := range compartments {
		fmt.Printf("Processing compartment: %s\n", *compartment.Id)
		enumerateIPsInCompartment(ctx, vcnClient, *compartment.Id)
	}
}

func enumerateIPsInCompartment(ctx context.Context, vcnClient core.VirtualNetworkClient, compartmentID string) {
	// List all VNICs in the compartment
	listVnicsRequest := core.ListVnicAttachmentsRequest{
		CompartmentId: common.String(compartmentID),
	}

	vnics, err := vcnClient.ListVnicAttachments(ctx, listVnicsRequest)
	if err != nil {
		fmt.Printf("Error listing VNIC attachments: %v\n", err)
		return
	}

	for _, vnicAttachment := range vnics.Items {
		vnic, err := vcnClient.GetVnic(ctx, core.GetVnicRequest{VnicId: vnicAttachment.VnicId})
		if err != nil {
			fmt.Printf("Error getting VNIC: %v\n", err)
			continue
		}

		fmt.Printf("Found VNIC: %s with IP: %s\n", *vnic.Id, *vnic.PrivateIp)
		if vnic.PublicIp != nil {
			fmt.Printf(" - Public IP: %s\n", *vnic.PublicIp)
		}
	}

	// List all public IPs in the compartment
	listPublicIPsRequest := core.ListPublicIpsRequest{
		CompartmentId: common.String(compartmentID),
		Scope:         core.ListPublicIpsScopeRegion,
	}

	publicIPs, err := vcnClient.ListPublicIps(ctx, listPublicIPsRequest)
	if err != nil {
		fmt.Printf("Error listing public IPs: %v\n", err)
		return
	}

	for _, publicIP := range publicIPs.Items {
		fmt.Printf("Found Reserved Public IP: %s\n", *publicIP.IpAddress)
	}
}
