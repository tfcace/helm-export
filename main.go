package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	b64 "encoding/base64"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type HelmSecret struct {
	Manifest string `json:"manifest"`
}

func (h *HelmSecret) Unmsrshal(payload []byte) error {
	helmDecoded, err := b64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(helmDecoded)
	gzReader, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	d := yaml.NewYAMLOrJSONDecoder(gzReader, 256)
	d.Decode(h)
	return nil
}

func (h *HelmSecret) Export(outputPath string) error {
	docs := strings.Split(h.Manifest, "---")
	for _, d := range docs {
		if len(d) == 0 {
			continue
		}
		var obj metav1.PartialObjectMetadata
		dec := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(d), 256)
		if err := dec.Decode(&obj); err != nil {
			return err
		}
		if obj.Kind == "" {
			continue
		}
		filename := fmt.Sprintf("%s.%s.%s", strings.ToLower(obj.Kind), obj.Name, "yaml")
		ioutil.WriteFile(
			fmt.Sprintf("%s/%s", outputPath, filename),
			[]byte(d),
			0644,
		)
	}
	return nil
}

func defaultKubeConf() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return path.Join(homedir, ".kube", "config")
}

func defaultNamespace() string {
	return "default"
}

type Retriever interface {
	Retrieve() (v1.Secret, error)
}

type SecretRetriever struct {
	clientset *kubernetes.Clientset
	namespace string
}

func NewSecretRetriever(kubeconfig *rest.Config, namespace string) *SecretRetriever {
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil
	}
	sr := SecretRetriever{
		clientset: clientset,
		namespace: namespace,
	}
	return &sr
}

func (sr *SecretRetriever) Retrieve(secret string) (*HelmSecret, error) {
	s, err := sr.clientset.CoreV1().Secrets(sr.namespace).Get(context.TODO(), secret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var hs HelmSecret
	err = hs.Unmsrshal(s.Data["release"])
	return &hs, err
}

func main() {

	var kubeconfig string
	var namespace string
	var secret string
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeConf(), "help message for flag kubeconfig")
	flag.StringVar(&namespace, "n", defaultNamespace(), "help message for flag n")
	flag.Parse()

	if secret = flag.Arg(0); secret == "" {
		fmt.Println("secret is missing")
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	ret := NewSecretRetriever(config, namespace)
	hs, _ := ret.Retrieve(secret)
	outputDir, _ := os.Getwd()
	err = hs.Export(outputDir)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
