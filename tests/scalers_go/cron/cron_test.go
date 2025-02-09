//go:build e2e
// +build e2e

package cron_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "cron-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)

	now   = time.Now().Local()
	start = (now.Minute() + 1)
	end   = (start + 1)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	StartMin         string
	EndMin           string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.DeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      pod: {{.DeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.DeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'nginx'
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 1
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: {{.StartMin}} * * * *
      end: {{.EndMin}} * * * *
      desiredReplicas: '4'
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates...)

	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1 after a minute")

	// test scaling
	testScaleUp(t, kc)
	testScaleDown(t, kc)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates...)
}

func getTemplateData() (templateData, []string) {
	return templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		StartMin:         strconv.Itoa(start),
		EndMin:           strconv.Itoa(end),
	}, []string{deploymentTemplate, scaledObjectTemplate}
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale up ---")
	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 4, 60, 2),
		"replica count should be 4 after a minute")
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale down ---")
	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1 after a minute")
}
