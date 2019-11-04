# Cloud Foundry CLI AutoScaler Plug-in [![Build Status](https://travis-ci.org/cloudfoundry-incubator/app-autoscaler-cli-plugin.svg?branch=master)](https://travis-ci.org/cloudfoundry-incubator/app-autoscaler-cli-plugin)

App-AutoScaler plug-in provides the command line interface to manage [App AutoScaler](https://github.com/cloudfoundry-incubator/app-autoscaler) policies, retrieve metrics and scaling event history.


## Install plugin

### From CF-Community

```
cf install-plugin -r CF-Community app-autoscaler-plugin
```

## From source code


```
$ git clone git@github.com:cloudfoundry-incubator/app-autoscaler-cli-plugin.git
$ cd app-autoscaler-cli-plugin
$ git submodule update --init --recursive
$ source .envrc
$ scripts/build
$ cf install-plugin out/ascli
```

## Uninstall plugin

```
cf uninstall-plugin AutoScaler
```

## Command List

| Command | Description |
|---------|-------------|
| [autoscaling-api, asa](#cf-autoscaling-api) | Set or view AutoScaler service API endpoint |
| [autoscaling-policy, asp](#cf-autoscaling-policy) | Retrieve the scaling policy of an application |
| [attach-autoscaling-policy, aasp](#cf-attach-autoscaling-policy) | Attach a scaling policy to an application |
| [detach-autoscaling-policy, dasp](#cf-detach-autoscaling-policy) | Detach the scaling policy from an application |
| [create-autoscaling-credential, casc](#cf-create-autoscaling-credential) | Create custom metric credential for an application |
| [delete-autoscaling-credential, dasc](#cf-delete-autoscaling-credential) | Delete the custom metric credential of an application |
| [autoscaling-metrics, asm](#cf-autoscaling-metrics) | Retrieve the metrics of an application |
| [autoscaling-history, ash](#cf-autoscaling-history) | Retrieve the scaling history of an application|

## Command usage

### `cf autoscaling-api`

Set or view AutoScaler service API endpoint. If the CF API endpoint is https://api.example.com, then typically the autoscaler API endpoint will be https://autoscaler.example.com. Check the manifest when autoscaler is deployed to get the autoscaler service API endpoint. 

```
cf autoscaling-api [URL] [--unset] [--skip-ssl-validation]
```

#### ALIAS: asa

#### OPTIONS:
- `--unset`: Unset the api endpoint
- `--skip-ssl-validation` : Skip verification of the API endpoint. Not recommended!

#### EXAMPLES:

- Set AutoScaler API endpoint, replace `DOMAIN` with the domain of your Cloud Foundry environment:

```
$ cf autoscaling-api https://autoscaler.<DOMAIN>
Setting AutoScaler api endpoint to https://autoscaler.<DOMAIN>
OK
```

- View AutoScaler API endpoint:

```
$ cf autoscaling-api
Autoscaler api endpoint: https://autoscaler.<DOMAIN>
```

- Unset AutoScaler API endpoint:

Note you will get a error prompt if the AutoScaler API endpoint is not set when you execute other commands.

```
$ cf autoscaling-api --unset
Unsetting AutoScaler api endpoint.
OK

$ cf autoscaling-api
No api endpoint set. Use 'cf autoscaling-api' to set an endpoint.

$ cf autoscaling-policy APP_NAME
FAILED
Error: No api endpoint set. Use 'cf autoscaling-api' to set an endpoint.
```


### `cf autoscaling-policy` 

Retrieve the scaling policy of an application, the policy will be displayed in JSON format.

```
cf autoscaling-policy APP_NAME [--output PATH_TO_FILE]
```

#### ALIAS: asp


#### OPTIONS:
- `--output` : dump the policy to a file in JSON format


#### EXAMPLES:
- View scaling policy, replace `APP_NAME` with the name of your application:
```
$ cf autoscaling-policy APP_NAME

Showing policy for app APP_NAME...
{
	"instance_min_count": 1,
	"instance_max_count": 5,
	"scaling_rules": [
		{
			"metric_type": "memoryused",
			"breach_duration_secs": 120,
			"threshold": 15,
			"operator": ">=",
			"cool_down_secs": 120,
			"adjustment": "+1"
		},
		{
			"metric_type": "memoryused",
			"breach_duration_secs": 120,
			"threshold": 10,
			"operator": "<",
			"cool_down_secs": 120,
			"adjustment": "-1"
		}
	]
}
```
- Dump the scaling policy to a file in JSON format:
```
$ cf asp APP_NAME --output PATH_TO_FILE

Saving policy for app APP_NAME to PATH_TO_FILE...
OK
```

### `cf attach-autoscaling-policy` 

Attach a scaling policy to an application, the policy file must be a JSON file, refer to [policy specification](https://github.com/cloudfoundry/app-autoscaler/blob/develop/docs/policy.md) for the policy format.

```
cf attach-autoscaling-policy APP_NAME PATH_TO_POLICY_FILE
```

#### ALIAS: aasp

#### EXAMPLES:
```
$ cf attach-autoscaling-policy APP_NAME PATH_TO_POLICY_FILE

Attaching policy for app APP_NAME...
OK
```


### `cf detach-autoscaling-policy` 

Detach the scaling policy from an application, the policy will be **deleted** when detached.

```
cf detach-autoscaling-policy APP_NAME
```
#### ALIAS: dasp

#### EXAMPLES:
```
$ cf detach-autoscaling-policy APP_NAME

Detaching policy for app APP_NAME...
OK
```


### `cf create-autoscaling-credential`

Credential is required when submitting custom metrics to app-autoscaler.
If an application is connecting to autoscaler through a service binding approach, the required credential could be found in Cloud Foundry `VCAP_SERVICES` environment variables.
This command is used to generate autoscaler credential and display it in JSON format. Then you need to set this credential to your application through environment variable or user-provide-service.
Auto-scaler only grants access with the most recent credential, so the newly generated credential will overwritten the old pairs. Please ensure to update the credential setting in your application once you launch the command `create-autoscaling-credential`.
Random credential pair will be created by default when username and password are not specified by `--username` and `--password` option.

```
cf create-autoscaling-credential APP_NAME [--username USERNAME --password PASSWORD] [--output PATH_TO_FILE]
```
#### ALIAS: casc


#### OPTIONS:
- `--username, -u` : username of the custom metric credential, random username will be set if not specified
- `--password, -p` : password of the custom metric credential, random password will be set if not specified
- `--output`       : Dump the credential to a file in JSON format

#### EXAMPLES:
- Create and view custom credential with user-defined username and password:
```
$ cf create-autoscaling-credential APP_NAME --username MY_USERNAME --password MY_PASSWORD

Creating custom metric credential for app APP_NAME...
{
	"app_id": "<APP_ID>",
	"username": "MY_USERNAME",
	"password": "MY_PASSWORD",
	"url": "https://autoscalermetrics.<DOMAIN>"
}
```
- Create random username and password and dump the credential to a file:
```
$ cf create-autoscaling-credential APP_NAME --output PATH_TO_FILE

Saving new created credential for app APP_NAME to PATH_TO_FILE...
OK
```


### `cf delete-autoscaling-credential`

Delete the custom metric credential of an application.

```
cf delete-autoscaling-credential APP_NAME
```
#### ALIAS: dasc

#### EXAMPLES:
```
$ cf delete-autoscaling-credential APP_NAME

Deleting custom metric credential for app APP_NAME...
OK
```


### `cf autoscaling-metrics`

Retrieve the aggregated metrics of an application. You can specify the start/end time of the returned query result,  and the display order(ascending or descending). The metrics will be shown in a table.

```
cf autoscaling-metrics APP_NAME METRIC_NAME [--start START_TIME] [--end END_TIME] [--asc] [--output PATH_TO_FILE]
```
#### ALIAS: asm


#### OPTIONS:
- `METRIC_NAME` : default metrics "memoryused, memoryutil, responsetime, throughput, cpu" or customized name for your own metrics.
- `--start` : start time of metrics collected with format `yyyy-MM-ddTHH:mm:ss+/-HH:mm` or `yyyy-MM-ddTHH:mm:ssZ`, default to very beginning if not specified.
- `--end` : end time of the metrics collected with format `yyyy-MM-ddTHH:mm:ss+/-HH:mm` or `yyyy-MM-ddTHH:mm:ssZ`, default to current time if not speficied.
- `--asc` : display in ascending order, default to descending order if not specified
- `--output` : dump the metrics to a file

#### EXAMPLES:
```
$ cf autoscaling-metrics APP_NAME memoryused --start 2018-12-27T11:49:00+08:00 --end 2018-12-27T11:52:20+08:00 --asc

Retriving aggregated metrics for app APP_NAME...
Metrics Name     	Value     	Timestamp
memoryused       	62MB      	2018-12-27T11:49:00+08:00
memoryused       	62MB      	2018-12-27T11:49:40+08:00
memoryused       	61MB      	2018-12-27T11:50:20+08:00
memoryused       	62MB      	2018-12-27T11:51:00+08:00
memoryused       	62MB      	2018-12-27T11:51:40+08:00
```
- `Metrics Name`: name of the current metric item
- `Value`: the value of the current metric item with unit
- `Timestamp`: collect time of the current metric item

###  `cf autoscaling-history`

Retrieve the scaling event history of an application. You can specify the start/end time of the returned query result,  and the display order(ascending or descending). The scaling event history will be shown in a table.
```
cf autoscaling-history APP_NAME [--start START_TIME] [--end END_TIME] [--asc] [--output PATH_TO_FILE]
```

#### ALIAS: ash

#### OPTIONS:
- `--start` : start time of the scaling history with format `yyyy-MM-ddTHH:mm:ss+/-HH:mm` or `yyyy-MM-ddTHH:mm:ssZ`, default to very beginning if not specified.
- `--end` : end time of the scaling history with format `yyyy-MM-ddTHH:mm:ss+/-HH:mm` or `yyyy-MM-ddTHH:mm:ssZ`, default to current time if not speficied.
- `--asc` : display in ascending order, default to descending order if not specified
- `--output` : dump the scaling history to a file

#### EXAMPLES:
```
$ cf autoscaling-history APP_NAME --start 2018-08-16T17:58:53+08:00 --end 2018-08-16T18:01:00+08:00 --asc

Showing history for app APP_NAME...
Scaling Type     	Status        	Instance Changes     	Time                          	Action                                                        	Error
dynamic          	failed        	2->-1                	2018-08-16T17:58:53+08:00     	-1 instance(s) because throughput < 10rps for 120 seconds     	app does not have policy set
dynamic          	succeeded     	2->3                 	2018-08-16T17:59:33+08:00     	+1 instance(s) because memoryused >= 15MB for 120 seconds
scheduled        	succeeded     	3->6                 	2018-08-16T18:00:00+08:00     	3 instance(s) because limited by min instances 6
```
- `Scaling Type`: the trigger type of the scaling action, possible scaling types: `dynamic` and `scheduled`
  - `dynamic`: the scaling action is triggered by a dynamic rule (memoryused, memoryutil, responsetime or throughput)
  - `scheduled`: the scaling action is triggered by a recurring schedule or specific date rule
- `Status`: the result of the scaling action: `succeeded` or `failed`
- `Instance Changes`: how the instances number get changed (e.g. `1->2` means the application was scaled out from 1 instance to 2)
- `Time`: the finish time of scaling action, no mater succeeded or failed
- `Action`: the detail information about why and how the application scaled
- `Error`: the reason why scaling failed

