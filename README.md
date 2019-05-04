[![Build Status](https://travis-ci.org/kubedev/line-bot-operator.svg?branch=master)](https://travis-ci.org/kubedev/line-bot-operator) [![codecov](https://codecov.io/gh/kubedev/line-bot-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/kubedev/line-bot-operator) [![Docker Pulls](https://img.shields.io/docker/pulls/kubedev/line-bot-operator.svg)](https://hub.docker.com/r/kubedev/line-bot-operator/) ![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)

# LINE Bot Operator 
An operator provides LINE bot that makes it easy to deploy on Kubernetes.

## Concepts

<p align="center"><img src="images/concepts.png"></p>

This operator has three fundamental concepts:

* **Bot** defines the desired spec of the Bot deployment.
* **Event** defines eventing rules for a bot instance.
* **EventBinding** defines the set of events to be used by the bot. You select Events to be bound using labels and label selectors.

## Building from Source
Clone repo into your go path under `$GOPATH/src`:
```sh
$ git clone https://github.com/kubedev/line-bot-operator.git $GOPATH/src/github.com/kubedev/line-bot-operator
$ cd $GOPATH/src/github.com/kubedev/line-bot-operator
$ make dep
$ make
```