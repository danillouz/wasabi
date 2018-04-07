package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var namespace = "wasabi"

func main() {
	config, err := rest.InClusterConfig()

	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	})

	jobs := router.Group("/jobs")
	jobs.POST("/:name", func(c *gin.Context) {
		name := c.Param("name")
		secret, err := clientset.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": fmt.Sprintf("Job config not found for %s", name),
			})

			return
		}

		var body interface{}

		decoder := json.NewDecoder(c.Request.Body)

		if err := decoder.Decode(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Malformed request body",
			})

			return
		}

		c.Request.Body.Close()

		img := string(secret.Data["image"])
		now := time.Now().Unix()
		jobName := fmt.Sprintf("%s-%d", "wasabi", now)
		config := map[string]interface{}{}

		for k, v := range secret.Data {
			config[k] = string(v)
		}

		parsedBody := body.(map[string]interface{})

		var envVars []coreV1.EnvVar

		if parsedBody["containerEnvVars"] != nil {
			if err := mapstructure.Decode(parsedBody["containerEnvVars"], &envVars); err != nil {
				panic(err.Error())
			}

			delete(parsedBody, "containerEnvVars")
		}

		out := map[string]interface{}{}
		out["config"] = config
		out["payload"] = parsedBody
		outputJSON, _ := json.Marshal(out)

		args := []string{string(outputJSON)}

		if parsedBody["passContainerArg"] != nil {
			b, ok := parsedBody["passContainerArg"].(bool)

			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Malformed request body: 'passContainerArg' must be a Boolean",
				})

				return
			}

			if b == false {
				args = []string{}
			}
		}

		container := coreV1.Container{
			Name:  "wasabi-runtime",
			Image: img,
			Args:  args,
			Env:   envVars,
		}

		job := batchV1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
			},
			Spec: batchV1.JobSpec{
				Template: coreV1.PodTemplateSpec{
					Spec: coreV1.PodSpec{
						Containers:    []coreV1.Container{container},
						RestartPolicy: "Never",
					},
				},
			},
		}

		jobResult, err := clientset.BatchV1().Jobs(namespace).Create(&job)

		if err != nil {
			panic(err.Error())
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
			"name":    jobResult.ObjectMeta.Name,
			"UID":     jobResult.ObjectMeta.UID,
		})
	})

	router.Run()
}
