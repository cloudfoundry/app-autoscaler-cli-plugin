# Cloud Foundry CLI AutoScaler Plug-in [![Build Status](https://travis-ci.org/cloudfoundry-incubator/app-autoscaler-cli-plugin.svg?branch=master)](https://travis-ci.org/cloudfoundry-incubator/app-autoscaler-cli-plugin)

App-AutoScaler plug-in provides the command line interface to manage [App AutoScaler](https://github.com/cloudfoundry-incubator/app-autoscaler) service policies, retrieve metrics and scaling history.


## Set up

To set up the development, follow the steps below

```
$ git clone git@github.com:cloudfoundry-incubator/app-autoscaler-cli-plugin.git
$ cd app-autoscaler-cli-plugin
$ source .envrc
$ git submodule update --init --recursive
```

## Build and install

Run the following script to build the AutoScaler plug-in for your OS
```
scripts/build
```

To install the plugin to cf cli
```
cf install-plugin out/ascli
```

To uninstall the plugin
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
| [autoscaling-metrics, asm](#cf-autoscaling-metrics) | Retrieve the metrics of an application |
| [autoscaling-history, ash](#cf-autoscaling-history) | Retrieve the scaling history of an application|

## Command usage

### `cf autoscaling-api`

Set or view AutoScaler service API endpoint

```
cf autoscaling-api [URL] [--unset] [--skip-ssl-validation]
```

**ALIAS**: asa

**OPTIONS**
- `--unset`: Unset the api endpoint
- `--skip-ssl-validation` : Skip verification of the API endpoint. Not recommended!

### `cf autoscaling-policy` 

Retrieve the scaling policy of an application

```
cf autoscaling-policy APP_NAME [--output PATH_TO_FILE]
```

**ALIAS**: asp


**OPTIONS**
- `--output` : dump the policy to a file in JSON format

### `cf attach-autoscaling-policy` 

Attach a scaling policy to an application
```
cf attach-autoscaling-policy APP_NAME PATH_TO_POLICY_FILE
```

**ALIAS**: aasp


### `cf detach-autoscaling-policy` 

Detach the scaling policy from an application
```
cf detach-as-policy APP_NAME
```
**ALIAS**: dasp


### `cf autoscaling-metrics`

Retrieve the metrics of an application

```
cf autoscaling-metrics APP_NAME METRIC_NAME [--start START_TIME] [--end END_TIME] [--desc] [--output PATH_TO_FILE]
```
**ALIAS**: asm


**OPTIONS**
- `METRIC_NAME` : available metric supported: memoryused, memoryutil, responsetime,throughput.
- `--start` : start time of metrics collected with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
- `--end` : end time of the metrics collected  with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
- `--desc` : display in descending order, default to ascending order if not specified
- `--output` : dump the metrics to a file

###  `cf autoscaling-history` 

Retrieve the scaling history of an application.

```
cf autoscaling-history APP_NAME [--start START_TIME] [--end END_TIME] [--desc] [--output PATH_TO_FILE]
```

**ALIAS**: ash

**OPTIONS**
- `--start` : start time of the scaling history with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
- `--end` : end time of the scaling history with format "yyyy-MM-ddTHH:mm:ss+/-HH:mm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
- `--desc` : display in descending order, default to ascending order if not specified
- `--output` : dump the scaling history to a file
