# Azure Assets Input

## What does it do?

The Azure Assets Input collects data about Azure resources and their relationships to each other.
Information about the following resources is currently collected:

- Azure VM instances

## Configuration

```yaml
assetbeat.inputs:
  - type: assets_azure
    regions:
        - <region>
    subscription_id: <your subscription ID>
    client_id: <your client ID>
    client_secret: <your client secret>
    tenant_id: <your tenant ID>
```

The Azure Assets Input supports the following configuration options plus the [Common options](../README.md#Common options).

* `regions`: The list of Azure regions to collect data from.

**_Note_:** `client_id`, `client_secret` and `tenant_id` can be omitted if:
* The environment variables `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` and `AZURE_TENANT_ID` are set.
* `az login` was ran on the host where `assetbeat` is running.

## Asset schema

### VM instances

#### Exported fields

| Field      | Description                       | Example                                                                                                                        |
|------------|-----------------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| asset.type | The type of asset                 | `"azure.vm.instance"`                                                                                                          |
| asset.kind | The kind of asset                 | `"host`                                                                                                                        |
| asset.id   | The id of the Azure instance      | `"/subscriptions/12cabcb4-86e8-404f-111111111111/resourceGroups/TESTVM/providers/Microsoft.Compute/virtualMachines/test"`      |
| asset.ean  | The EAN of this specific resource | `"host:/subscriptions/12cabcb4-86e8-404f-111111111111/resourceGroups/TESTVM/providers/Microsoft.Compute/virtualMachines/test"` |

#### Example

```json
{
  "@timestamp": "2023-09-07T15:57:59.121Z",
  "asset.ean": "host:/subscriptions/12cabcb4-86e8-404f-111111111111/resourceGroups/TESTVM/providers/Microsoft.Compute/virtualMachines/test",
  "asset.type": "azure.vm.instance",
  "input": {
    "type": "assets_azure"
  },
  "agent": {
    "id": "9a7ef1a9-0cce-4857-90f9-699bc14d8df3",
    "name": "testhost",
    "type": "assetbeat",
    "version": "8.9.0",
    "ephemeral_id": "54bf2e30-2978-4c33-a465-26682acdd596"
  },
  "cloud.account.id": "70bd6e77-4b1e-4835-8896-111111111111",
  "cloud.region": "westeurope",
  "asset.kind": "host",
  "asset.id": "/subscriptions/12cabcb4-86e8-404f-111111111111/resourceGroups/TESTVM/providers/Microsoft.Compute/virtualMachines/test",
  "ecs": {
    "version": "8.0.0"
  },
  "host": {
    "name": "testhost"
  },
  "cloud.provider": "azure"
}
```