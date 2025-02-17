package dataset

import (
	"fmt"

	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type DataSetClient struct {
	Client *resourcemanager.Client
}

func NewDataSetClientWithBaseURI(api environments.Api) (*DataSetClient, error) {
	client, err := resourcemanager.NewResourceManagerClient(api, "dataset", defaultApiVersion)
	if err != nil {
		return nil, fmt.Errorf("instantiating DataSetClient: %+v", err)
	}

	return &DataSetClient{
		Client: client,
	}, nil
}
