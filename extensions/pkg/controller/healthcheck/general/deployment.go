// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package general

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"

	"github.com/gardener/gardener/pkg/utils/kubernetes/health"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeploymentHealthChecker contains all the information for the Deployment HealthCheck
type DeploymentHealthChecker struct {
	logger      logr.Logger
	seedClient  client.Client
	shootClient client.Client
	name        string
	checkType   DeploymentCheckType
}

// DeploymentCheckType in which cluster the check will be executed
type DeploymentCheckType string

const (
	DeploymentCheckTypeSeed  DeploymentCheckType = "Seed"
	DeploymentCheckTypeShoot DeploymentCheckType = "Shoot"
)

// NewSeedDeploymentHealthChecker is a healthCheck function to check Deployments in the Seed cluster
func NewSeedDeploymentHealthChecker(deploymentName string) healthcheck.HealthCheck {
	return &DeploymentHealthChecker{
		name:      deploymentName,
		checkType: DeploymentCheckTypeSeed,
	}
}

// NewShootDeploymentHealthChecker is a healthCheck function to check Deployments in the Shoot cluster
func NewShootDeploymentHealthChecker(deploymentName string) healthcheck.HealthCheck {
	return &DeploymentHealthChecker{
		name:      deploymentName,
		checkType: DeploymentCheckTypeShoot,
	}
}

// InjectSeedClient injects the seed client
func (healthChecker *DeploymentHealthChecker) InjectSeedClient(seedClient client.Client) {
	healthChecker.seedClient = seedClient
}

// InjectShootClient injects the shoot client
func (healthChecker *DeploymentHealthChecker) InjectShootClient(shootClient client.Client) {
	healthChecker.shootClient = shootClient
}

// SetLoggerSuffix injects the logger
func (healthChecker *DeploymentHealthChecker) SetLoggerSuffix(provider, extension string) {
	healthChecker.logger = log.Log.WithName(fmt.Sprintf("%s-%s-healthcheck-deployment", provider, extension))
}

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *DeploymentHealthChecker) DeepCopy() healthcheck.HealthCheck {
	copy := *healthChecker
	return &copy
}

// Check executes the health check
func (healthChecker *DeploymentHealthChecker) Check(ctx context.Context, request types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	deployment := &appsv1.Deployment{}

	var err error
	if healthChecker.checkType == DeploymentCheckTypeSeed {
		err = healthChecker.seedClient.Get(ctx, client.ObjectKey{Namespace: request.Namespace, Name: healthChecker.name}, deployment)
	} else {
		err = healthChecker.shootClient.Get(ctx, client.ObjectKey{Namespace: request.Namespace, Name: healthChecker.name}, deployment)
	}
	if err != nil {
		err := fmt.Errorf("failed to retrieve deployment '%s' in namespace '%s': %v", healthChecker.name, request.Namespace, err)
		healthChecker.logger.Error(err, "Health check failed")
		return nil, err
	}

	if isHealthy, reason, err := deploymentIsHealthy(deployment); !isHealthy {
		healthChecker.logger.Error(err, "Health check failed")
		return &healthcheck.SingleCheckResult{
			IsHealthy: false,
			Detail:    err.Error(),
			Reason:    *reason,
		}, nil
	}

	return &healthcheck.SingleCheckResult{
		IsHealthy: true,
	}, nil
}

func deploymentIsHealthy(deployment *appsv1.Deployment) (bool, *string, error) {
	if err := health.CheckDeployment(deployment); err != nil {
		reason := "DeploymentUnhealthy"
		err := fmt.Errorf("deployment %s in namespace %s is unhealthy: %v", deployment.Name, deployment.Namespace, err)
		return false, &reason, err
	}
	return true, nil, nil
}
