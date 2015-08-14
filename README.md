# Godyn
This little utility updates [Dynect](http://dyn.com) FQDN in zones.
We use it currently to let Docker containers run on [Joyent Triton](https://joyent.com)
update the names under which they are reachable themselves. If you want to run
just a few containers you don't want to have a big infrastructure with SkyDNS, Consul
etc. You just want you container to be reachable via a resolvable name.
For this reason we created Godyn. It determines the public IPv4 address from `/etc/hosts`
(assuming that the first valid entry is the public IP) and the updates your Dynect
zone.

## How to use

* Add the godyn binary to your images (for example by downloading it from [Bintray](https://bintray.com/artifact/download/yetu/maven/godyn))
* Make sure that godyn is executed before your application
* Create a file with environment variables for your container (see below)
* Run your container with the command line option `--env-file=<your dynect variables file>`

## Environment variables

Currently the only way to pass necessary configuration is via environment variables.
Necessary variables are:

Name | Description
---- | -----------
GODYN_ZONE | The Dynect managed zone you wish to modify
GODYN_FQDN | The fqdn you want to set the A record for

And for the Dynect provider set:

Name | Description
---- | -----------
DYNECT_CUSTOMER | Your Dynect customer name
DYNECT_USERNAME | Your Dynect user name
DYNECT_PASSWORD | Your Dynect password

All environment variables are required.

# Development

This tool is build [gb](http://getgb.io/) and follows the necessary folder structure.
