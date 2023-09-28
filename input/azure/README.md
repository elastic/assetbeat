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
* `subscription_id`: The unique identifier for the azure subscription
* `client_id`: The unique identifier for the application (also known as Application Id) 
* `client_secret`: The client/application secret/key
* `tenant_id`: The unique identifier of the Azure Active Directory instance

**_Note_:** `client_id`, `client_secret` and `tenant_id` can be omitted if:
* The environment variables `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` and `AZURE_TENANT_ID` are set.
* `az login` was ran on the host where `assetbeat` is running.

**_Note_:** if `subscription_id` is omitted, the input will collect data from all the subscriptions you have access to.

**_Note_:** if no region is provided under `regions` is omitted, the input will collect data from all the regions.


## Asset schema

### VM instances

#### Exported fields

| Field                         | Description                       | Example                                       |
|-------------------------------|-----------------------------------|-----------------------------------------------|
| asset.type                    | The type of asset                 | `"azure.vm.instance"`                         |
| asset.kind                    | The kind of asset                 | `"host`                                       |
| asset.name                    | The name of the Azure instance    | `"my_instance"`                               |
| asset.id                      | The VM id of the Azure instance   | `"00830b08-f63d-495b-9b04-989f83c50111"`      |
| asset.ean                     | The EAN of this specific resource | `"host:00830b08-f63d-495b-9b04-989f83c50111"` |
| asset.metadata.resource_group | The Azure resource group          | `TESTVM`                                      |
| asset.metadata.state          | The status of the VM instance     | `"VM running"`                                |

#### Example

```json
{
  "@timestamp": "2023-09-13T14:42:51.494Z",
  "asset.metadata.resource_group": "TESTVM",
  "host": {
    "name": "host"
  },
  "cloud.region": "westeurope",
  "cloud.provider": "azure",
  "agent": {
    "ephemeral_id": "a80c69df-22dd-4f97-bfd2-14572af2b9d4",
    "id": "9a7ef1a9-0cce-4857-90f9-699bc14d8df3",
    "name": "host",
    "type": "assetbeat",
    "version": "8.9.0"
  },
  "input": {
    "type": "assets_azure"
  },
  "cloud.account.id": "12cabcb4-86e8-404f-a3d2-111111111111",
  "asset.kind": "host",
  "asset.id": "00830b08-f63d-495b-9b04-989f83c50111",
  "asset.ean": "host:00830b08-f63d-495b-9b04-989f83c50111",
  "asset.metadata.state": "VM running",
  "asset.type": "azure.vm.instance",
  "asset.name": "my_instance",
  "ecs": {
    "version": "8.0.0"
  }
}
```