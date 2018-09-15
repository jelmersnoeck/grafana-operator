# Grafana Operator

[![Build Status](https://travis-ci.org/jelmersnoeck/grafana-operator.svg?branch=master)](https://travis-ci.org/jelmersnoeck/grafana-operator)

Grafana Operator is a Kubernetes Operator which takes care of installing a
Grafana instance accordingly and managing it's configuration. When the
configuration files change, the Operator will automatically restart the Grafana
instance to ensure the config is the latest.

## Goal

The goal of this operator is to manage Grafana Deployments and make it easier
to manage Dashboards and Data Sources by introducing CRDs which can be
configured separately.

The Grafana Operator will detect changes to these configurations and
automatically re-deploy the Grafana Deployment so it reflects these changes.
