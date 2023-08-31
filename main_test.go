package main

import (
	"encoding/json"

	"fmt"

	"io/ioutil"

	"net/http"

	"os"

	"os/exec"

	"strings"

	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"

	"github.com/stretchr/testify/assert"
)

var (
	apiVersion = "2023-03-01"

	expected_provisioningState = "Succeeded" // vm status

)

// const (

//  poolIDMK = 53

// )

func TestAzureVMWithLogic(t *testing.T) {

	t.Parallel()

	terraformOptions := &terraform.Options{

		TerraformDir: "../module",

		Vars: map[string]interface{}{

			"vm_size": "Standard_DS1_v2",
		},
	}

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// getting resource group name from the output.tf file

	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")

	// fmt.Println("Resource Group Name:", resourceGroupName)

	// getting subscription ID from the Environment variable

	subscriptionID := os.Getenv("SUBSCRIPTION_ID")

	// fmt.Println("THis is your subscription ID:-", subscriptionID)

	if subscriptionID == "" {

		t.Fatal("AZURE_SUBSCRIPTION_ID environment variable is not set")

	}

	// getting token ID from the Environment variable for ADO

	personalAccessTokenMK := os.Getenv("TF_VAR_adotoken")

	// fmt.Println("Personal Access Token for the ADO:-", personalAccessTokenMK)

	if personalAccessTokenMK == "" {

		t.Fatal("AZURE_PERSONAL_ACCESS_TOKEN environment variable is not set")

	}

	// getting token using the subscriptionID

	accessToken, err := getAccessToken(subscriptionID)

	// fmt.Println(accessToken)

	if err != nil {

		t.Errorf("Failed to get access token: %s", err.Error())

		return

	}

	// FIRST TEST CASE:- Testing the Vm status..

	// getting vm-name from the output.tf file and run the test on all the vm.

	vmNames := terraform.OutputList(t, terraformOptions, "vm_names")

	for _, name := range vmNames {

		// fmt.Println(name)

		url := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s?api-version=%s", subscriptionID, resourceGroupName, name, apiVersion)

		// fmt.Println(url)

		vmJSON, err := getVirtualMachineDetails(url, accessToken)

		// fmt.Println(vmJSON)

		if err != nil {

			t.Errorf("Failed to get virtual machine details: %s", err.Error())

			return

		}

		vmData, err := printVirtualMachineDetails([]byte(vmJSON))

		// fmt.Println(vmData)

		if err != nil {

			fmt.Println("Error:", err)

			return

		}

		actual_provisioningState, ok := vmData["properties"].(map[string]interface{})["provisioningState"].(string)

		// fmt.Println(actual_provisioningState)

		if !ok {

			t.Errorf("Failed to get provisioning state for VM %s", name)

			return

		}

		t.Run(fmt.Sprintf("State has been matched for VM: %s", name), func(t *testing.T) {

			assert.Equal(t, expected_provisioningState, actual_provisioningState, fmt.Sprintf("State has been mis-matched for VM: %s", name))

		})

	}

}

func getAccessToken(subscriptionID string) (string, error) {

	cmd := exec.Command("az", "account", "get-access-token", "--query", "accessToken", "--output", "tsv", "--subscription", subscriptionID)

	output, err := cmd.Output()

	if err != nil {

		return "", err

	}

	return strings.TrimSpace(string(output)), nil

}

// function to fetch the details of the VM..

func getVirtualMachineDetails(url, accessToken string) (string, error) {

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return "", err

	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)

	if err != nil {

		return "", err

	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {

		return "", err

	}

	return string(body), nil

}

func printVirtualMachineDetails(vmJSON []byte) (map[string]interface{}, error) {

	var vmData map[string]interface{}

	err := json.Unmarshal(vmJSON, &vmData)

	if err != nil {

		fmt.Printf("Failed to unmarshal JSON response: %s\n", err.Error())

		return nil, err

	}

	return vmData, nil

}
