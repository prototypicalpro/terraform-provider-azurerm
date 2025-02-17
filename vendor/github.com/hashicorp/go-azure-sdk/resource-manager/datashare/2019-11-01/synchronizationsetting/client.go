package synchronizationsetting

import (
	"fmt"

	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type SynchronizationSettingClient struct {
	Client *resourcemanager.Client
}

func NewSynchronizationSettingClientWithBaseURI(api environments.Api) (*SynchronizationSettingClient, error) {
	client, err := resourcemanager.NewResourceManagerClient(api, "synchronizationsetting", defaultApiVersion)
	if err != nil {
		return nil, fmt.Errorf("instantiating SynchronizationSettingClient: %+v", err)
	}

	return &SynchronizationSettingClient{
		Client: client,
	}, nil
}
